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
	role   string
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

func (h *Hub) AddClient(userID, noteID uuid.UUID, role string, conn *websocket.Conn) {
	client := &Client{
		conn:   conn,
		userID: userID,
		noteID: noteID,
		role:   role,
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
			return
		}

		var incoming Message
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			continue
		}

		incoming.UserID = c.userID
		incoming.NoteID = c.noteID

		if incoming.Type == EventCursorMove {
			if c.role == "viewer" {
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

func (h *Hub) removeClient(client *Client) {
	h.connect.Delete(client)
}

func (h *Hub) notifyPresence(client *Client, eventType string) {
	msg := Message{Type: eventType, NoteID: client.noteID, UserID: client.userID}
	raw, _ := json.Marshal(msg)
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
	raw, _ := json.Marshal(msg)
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
				activeNotes[client.noteID] = true
				return true
			})

			for noteID := range activeNotes {
				updatedAt, err := h.repo.GetNoteUpdatedAt(ctx, noteID)
				if err != nil {
					slog.ErrorContext(ctx, "WS: failed to get updated_at", "error", err)
					continue
				}

				h.mu.Lock()
				lastUpdate, exists := h.lastKnownUpdate[noteID]
				if !exists {
					h.lastKnownUpdate[noteID] = updatedAt
					h.mu.Unlock()
				} else if updatedAt.After(lastUpdate) {
					h.lastKnownUpdate[noteID] = updatedAt
					h.mu.Unlock()

					msg, _ := json.Marshal(Message{Type: EventNoteUpdated, NoteID: noteID})
					h.connect.Range(func(key, value any) bool {
						peer := key.(*Client)
						if peer.noteID == noteID {
							select {
							case peer.send <- msg:
							case <-peer.done:
							default:
							}
						}
						return true
					})
				} else {
					h.mu.Unlock()
				}

				h.connect.Range(func(key, value any) bool {
					client := key.(*Client)
					if client.noteID == noteID {
						newRole, err := h.repo.CheckAccess(ctx, client.userID, noteID)
						if err != nil {
							slog.Info("WS: access revoked", "user_id", client.userID, "note_id", noteID)
							_ = client.conn.Close()
							h.connect.Delete(client)
							return true
						}
						client.role = newRole
					}
					return true
				})
			}

		case <-ctx.Done():
			h.connect.Range(func(key, value any) bool {
				client := key.(*Client)
				_ = client.conn.Close()
				h.connect.Delete(client)
				return true
			})
			return
		}
	}
}
