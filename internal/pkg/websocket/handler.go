package ws

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"bnn/internal/models"
	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/middleware"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

type NotesAccessChecker interface {
	CheckAccess(ctx context.Context, userID, noteID uuid.UUID) (string, error)
}

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
	notes    NotesAccessChecker
}

func NewHandler(hub *Hub, allowedOrigins string, notes NotesAccessChecker) *Handler {
	allowed := middleware.ParseOrigins(allowedOrigins)

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

	role, err := h.notes.CheckAccess(r.Context(), userID, noteID)
	if err != nil {
		if errors.Is(err, models.ErrNoteNotFound) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		slog.ErrorContext(r.Context(), "Failed to check role", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to upgrade connection", "error", err)
		return
	}

	h.hub.AddClient(userID, noteID, role, conn)
}
