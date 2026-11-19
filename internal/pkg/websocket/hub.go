package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

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
	GetNoteUpdatedAt(ctx context.Context, noteID uuid.UUID) (time.Time, error)
	CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error)
}

type Client struct {
	conn   *websocket.Conn
	userID uuid.UUID
	noteID uuid.UUID
	send   chan []byte
	done   chan struct{}
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

func (h *Hub) AddClient(userID, noteID uuid.UUID, conn *websocket.Conn) {
	client := &Client{
		conn:   conn,
		userID: userID,
		noteID: noteID,
		send:   make(chan []byte, 256),
		done:   make(chan struct{}),
	}

	h.connect.Store(client, userID)

	conn.SetCloseHandler(func(code int, text string) error {
		h.connect.Delete(client)
		return nil
	})

	h.notifyPresence(client, EventUserJoined)

	go client.writePump()
	go client.readPump(h)
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
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

func (c *Client) readPump(h *Hub) {
	defer func() {
		h.removeClient(c)
		close(c.done)
		_ = c.conn.Close()
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket unexpected close", "error", err, "user_id", c.userID)
			}
			return
		}

		role, err := h.repo.CheckAccess(context.Background(), c.userID, c.noteID)
		if err != nil || role == "viewer" {
			slog.Warn("WS: No edit access", "user_id", c.userID, "note_id", c.noteID)
			continue
		}

		var incoming Message
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			slog.Warn("Invalid WS JSON payload", "error", err, "user_id", c.userID)
			continue
		}

		incoming.UserID = c.userID
		incoming.NoteID = c.noteID

		if incoming.Type == EventCursorMove {
			h.broadcastCursor(c, incoming)
		}
	}
}

func (h *Hub) notifyPresence(client *Client, eventType string) {
	msg := Message{
		Type:   eventType,
		NoteID: client.noteID,
		UserID: client.userID,
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.connect.Range(func(key, value any) bool {
		peer := key.(*Client)
		if peer != client && peer.noteID == client.noteID {
			select {
			case peer.send <- raw:
			case <-peer.done:
			default:
			}
		}
		return true
	})
}

func (h *Hub) broadcastCursor(sender *Client, msg Message) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.connect.Range(func(key, value any) bool {
		peer := key.(*Client)
		if peer == sender || peer.noteID != sender.noteID {
			return true
		}

		select {
		case peer.send <- raw:
		case <-peer.done:
		default:
		}
		return true
	})
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
				role, err := h.repo.CheckAccess(ctx, client.userID, client.noteID)
				if err != nil || role == "viewer" {
					if err != nil {
						client.conn.Close()
						h.connect.Delete(client)
						return true
					}
				}

				activeNotes[client.noteID] = true
				return true
			})

			for noteID := range activeNotes {
				updatedAt, err := h.repo.GetNoteUpdatedAt(ctx, noteID)
				if err != nil {
					slog.ErrorContext(ctx, "WS failed to poll note update time", "error", err, "note_id", noteID)
					continue
				}

				h.mu.Lock()
				lastUpdate, exists := h.lastKnownUpdate[noteID]

				if !exists {
					h.lastKnownUpdate[noteID] = updatedAt
					h.mu.Unlock()
					continue
				}

				if updatedAt.After(lastUpdate) {
					h.lastKnownUpdate[noteID] = updatedAt
					h.mu.Unlock()

					msg := Message{
						Type:   EventNoteUpdated,
						NoteID: noteID,
					}
					rawMsg, err := json.Marshal(msg)
					if err != nil {
						continue
					}

					h.connect.Range(func(key, value any) bool {
						peer := key.(*Client)
						if peer.noteID == noteID {
							select {
							case peer.send <- rawMsg:
							case <-peer.done:
							default:
							}
						}
						return true
					})
				} else {
					h.mu.Unlock()
				}
			}

		case <-ctx.Done():
			h.connect.Range(func(key, value any) bool {
				client := key.(*Client)
				_ = client.conn.Close()
				h.connect.Delete(client)
				return true
			})
			h.mu.Lock()
			h.lastKnownUpdate = make(map[uuid.UUID]time.Time)
			h.mu.Unlock()

			return
		}
	}
}

func (h *Hub) removeClient(client *Client) {
	h.connect.Delete(client)

	hasActiveClients := false
	h.connect.Range(func(key, value any) bool {
		peer := key.(*Client)
		if peer.noteID == client.noteID {
			hasActiveClients = true
			return false
		}
		return true
	})

	if !hasActiveClients {
		h.mu.Lock()
		delete(h.lastKnownUpdate, client.noteID)
		h.mu.Unlock()
		slog.Info("Room is empty, cleared lastKnownUpdate", "note_id", client.noteID)
	}
}
