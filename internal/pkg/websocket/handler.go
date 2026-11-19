package ws

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

type NotesGetter interface {
	GetNote(ctx context.Context, userID, noteID uuid.UUID) (models.Note, error)
}

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
	notes    NotesGetter
}

func NewHandler(hub *Hub, allowedOrigin string, notes NotesGetter) *Handler {
	allowed := make([]string, 0)
	for _, origin := range strings.Split(allowedOrigin, ",") {
		origin = strings.TrimSpace(origin)
		origin = strings.TrimSuffix(origin, "/")
		if origin != "" {
			allowed = append(allowed, origin)
		}
	}

	return &Handler{
		hub:   hub,
		notes: notes,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				return slices.Contains(allowed, origin)
			},
		},
	}
}

func (h *Handler) ServeNoteWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxutil.GetUserID(r.Context())

	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	noteID, err := uuid.FromString(vars["noteID"])
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	_, err = h.notes.GetNote(r.Context(), userID, noteID)
	if err != nil {
		if errors.Is(err, models.ErrNoteNotFound) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "Failed to check note access", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to upgrade connection to WebSocket", "error", err)
		return
	}

	h.hub.AddClient(userID, noteID, conn)
}
