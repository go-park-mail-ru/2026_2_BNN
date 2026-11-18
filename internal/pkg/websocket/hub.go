package ws

import (
	"bnn/internal/models"
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
	GetUpdatedBlocks(ctx context.Context, noteID uuid.UUID, since time.Time) ([]models.Block, error)
}

type Client struct {
	conn   *websocket.Conn
	userID uuid.UUID
	noteID uuid.UUID
	send   chan []byte
}

type Hub struct {
	connect       sync.Map
	currentOffset time.Time
	repo          Repo
}

func NewHub(repo Repo) *Hub {
	return &Hub{
		currentOffset: time.Now().UTC().Add(-1 * time.Second),
		repo:          repo,
	}
}

func (h *Hub) AddClient(userID, noteID uuid.UUID, conn *websocket.Conn) {
	client := &Client{
		conn:   conn,
		userID: userID,
		noteID: noteID,
		send:   make(chan []byte, 256),
	}

	h.connect.Store(client, userID)

	conn.SetCloseHandler(func(code int, text string) error {
		h.connect.Delete(client)
		return nil
	})

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
		h.connect.Delete(c)
		close(c.send)
		_ = c.conn.Close()
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
			since := h.currentOffset.Add(-1 * time.Second)
			nextOffset := time.Now().UTC()

			updatesByNote := make(map[uuid.UUID][]models.Block)

			h.connect.Range(func(key, value any) bool {
				client := key.(*Client)

				blocks, cached := updatesByNote[client.noteID]
				if !cached {
					var err error
					blocks, err = h.repo.GetUpdatedBlocks(ctx, client.noteID, since)
					if err != nil {
						slog.ErrorContext(ctx, "WS failed to poll updates from DB", "error", err, "note_id", client.noteID)
						updatesByNote[client.noteID] = []models.Block{}
						return true
					}
					updatesByNote[client.noteID] = blocks
				}

				if len(blocks) == 0 {
					return true
				}

				payload, err := json.Marshal(blocks)
				if err != nil {
					slog.ErrorContext(ctx, "WS failed to marshal blocks", "error", err, "note_id", client.noteID)
					return true
				}

				msg := Message{
					Type:    EventNoteUpdated,
					NoteID:  client.noteID,
					Payload: payload,
				}

				rawMsg, err := json.Marshal(msg)
				if err != nil {
					return true
				}

				select {
				case client.send <- rawMsg:
				default:
				}

				return true
			})

			h.currentOffset = nextOffset

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
