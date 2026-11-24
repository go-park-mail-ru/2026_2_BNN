package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"bnn/internal/models"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

const (
	pollInterval = 5 * time.Second
	writeWait    = 10 * time.Second
	pongWait     = 60 * time.Second
	pingPeriod   = (pongWait * 9) / 10
	maxMsgSize   = 512 * 1024
)

type Repo interface {
	GetNoteUpdates(ctx context.Context, noteID uuid.UUID, since *time.Time) (time.Time, []models.Block, error)
	CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error)
}

type noteUpdatedPayload struct {
	UpdatedAt time.Time      `json:"updated_at"`
	Blocks    []models.Block `json:"blocks,omitempty"`
}

type Client struct {
	conn   *websocket.Conn
	userID uuid.UUID
	noteID uuid.UUID

	roleMu sync.RWMutex
	role   string

	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func (c *Client) getRole() string {
	c.roleMu.RLock()
	defer c.roleMu.RUnlock()
	return c.role
}

func (c *Client) setRole(role string) {
	c.roleMu.Lock()
	defer c.roleMu.Unlock()
	c.role = role
}

func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

type Hub struct {
	connect         sync.Map
	lastKnownUpdate map[uuid.UUID]time.Time
	mu              sync.Mutex
	repo            Repo
}

func NewHub(repo Repo) *Hub {
	return &Hub{
		lastKnownUpdate: make(map[uuid.UUID]time.Time),
		repo:            repo,
	}
}

func (h *Hub) AddClient(userID, noteID uuid.UUID, role string, conn *websocket.Conn) {
	client := &Client{
		conn:   conn,
		userID: userID,
		noteID: noteID,
		role:   role,
		send:   make(chan []byte, 256),
		done:   make(chan struct{}),
	}

	h.mu.Lock()
	if _, exists := h.lastKnownUpdate[noteID]; !exists {
		h.lastKnownUpdate[noteID] = time.Now().UTC()
	}
	h.mu.Unlock()

	h.connect.Store(client, userID)
	conn.SetCloseHandler(func(code int, text string) error {
		h.removeClient(client)
		return nil
	})

	h.notifyPresence(client, EventUserJoined)

	go client.writePump()
	go client.readPump(h)
}

func (c *Client) readPump(h *Hub) {
	defer func() {
		h.removeClient(c)
		c.shutdown()
		h.notifyPresence(c, EventUserLeft)
	}()

	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var incoming Message
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			continue
		}

		incoming.UserID = c.userID
		incoming.NoteID = c.noteID

		if incoming.Type == EventCursorMove {
			if c.getRole() == "viewer" {
				continue
			}
			h.broadcastCursor(c, incoming)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.shutdown()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-c.done:
			return
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) removeClient(client *Client) {
	h.connect.Delete(client)
}

func (h *Hub) sendToPeer(peer *Client, raw []byte) {
	if peer.send == nil {
		return
	}
	select {
	case peer.send <- raw:
	case <-peer.done:
	default:
	}
}

func (h *Hub) notifyPresence(client *Client, eventType string) {
	msg := Message{Type: eventType, NoteID: client.noteID, UserID: client.userID}
	raw, err := json.Marshal(msg)
	if err != nil {
		slog.Error("WS: failed to marshal presence message", "error", err)
		return
	}
	h.connect.Range(func(key, value any) bool {
		peer := key.(*Client)
		if peer != client && peer.noteID == client.noteID {
			h.sendToPeer(peer, raw)
		}
		return true
	})
}

func (h *Hub) broadcastCursor(sender *Client, msg Message) {
	raw, err := json.Marshal(msg)
	if err != nil {
		slog.Error("WS: failed to marshal cursor message", "error", err)
		return
	}
	h.connect.Range(func(key, value any) bool {
		peer := key.(*Client)
		if peer == sender || peer.noteID != sender.noteID {
			return true
		}
		h.sendToPeer(peer, raw)
		return true
	})
}

func (h *Hub) closeNoteClients(noteID uuid.UUID) {
	h.connect.Range(func(key, value any) bool {
		client := key.(*Client)
		if client.noteID == noteID {
			h.removeClient(client)
			client.shutdown()
		}
		return true
	})

	h.mu.Lock()
	delete(h.lastKnownUpdate, noteID)
	h.mu.Unlock()
}

func (h *Hub) Run(ctx context.Context) {
	t := time.NewTicker(pollInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			activeNotes := make(map[uuid.UUID]bool)
			h.connect.Range(func(key, value any) bool {
				client := key.(*Client)
				activeNotes[client.noteID] = true
				return true
			})

			h.mu.Lock()
			for noteID := range h.lastKnownUpdate {
				if !activeNotes[noteID] {
					delete(h.lastKnownUpdate, noteID)
				}
			}
			h.mu.Unlock()

			for noteID := range activeNotes {
				h.connect.Range(func(key, value any) bool {
					client := key.(*Client)
					if client.noteID == noteID {
						newRole, err := h.repo.CheckAccess(ctx, client.userID, noteID)
						if err != nil {
							slog.InfoContext(ctx, "WS: access revoked", "user_id", client.userID, "note_id", noteID)
							h.removeClient(client)
							client.shutdown()
							return true
						}
						client.setRole(newRole)
					}
					return true
				})

				h.mu.Lock()
				var since *time.Time
				if last, ok := h.lastKnownUpdate[noteID]; ok {
					tCopy := last
					since = &tCopy
				}
				h.mu.Unlock()

				updatedAt, blocks, err := h.repo.GetNoteUpdates(ctx, noteID, since)
				if err != nil {
					if errors.Is(err, models.ErrNoteNotFound) {
						h.closeNoteClients(noteID)
						continue
					}
					slog.ErrorContext(ctx, "WS: failed to get note updates", "error", err)
					continue
				}

				h.mu.Lock()
				lastUpdate, exists := h.lastKnownUpdate[noteID]
				hasNewUpdate := !exists || updatedAt.After(lastUpdate)
				if hasNewUpdate {
					h.lastKnownUpdate[noteID] = updatedAt
				}
				h.mu.Unlock()

				if hasNewUpdate {
					payloadBytes, err := json.Marshal(noteUpdatedPayload{
						UpdatedAt: updatedAt,
						Blocks:    blocks,
					})
					if err != nil {
						slog.ErrorContext(ctx, "WS: failed to marshal note_updated payload", "error", err)
						continue
					}

					msgBytes, err := json.Marshal(Message{
						Type:    EventNoteUpdated,
						NoteID:  noteID,
						Payload: payloadBytes,
					})
					if err != nil {
						slog.ErrorContext(ctx, "WS: failed to marshal note_updated", "error", err)
						continue
					}

					h.connect.Range(func(key, value any) bool {
						peer := key.(*Client)
						if peer.noteID == noteID {
							h.sendToPeer(peer, msgBytes)
						}
						return true
					})
				}
			}

		case <-ctx.Done():
			h.connect.Range(func(key, value any) bool {
				client := key.(*Client)
				h.removeClient(client)
				client.shutdown()
				return true
			})
			return
		}
	}
}
