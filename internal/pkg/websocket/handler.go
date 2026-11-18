package ws

import (
	"log/slog"
	"net/http"

	"bnn/internal/pkg/ctxutil"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

type TokenValidator interface {
	ValidateAccessToken(tokenStr string) (uuid.UUID, error)
}

type Handler struct {
	hub       *Hub
	upgrader  websocket.Upgrader
	validator TokenValidator
}

func NewHandler(hub *Hub, allowedOrigin string, validator TokenValidator) *Handler {
	return &Handler{
		hub:       hub,
		validator: validator,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" || allowedOrigin == "" || allowedOrigin == "*" {
					return true
				}
				return origin == allowedOrigin
			},
		},
	}
}

func (h *Handler) ServeNoteWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxutil.GetUserID(r.Context())

	if !ok {
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		uid, err := h.validator.ValidateAccessToken(tokenStr)
		if err != nil {
			slog.WarnContext(r.Context(), "WS auth failed", "error", err)
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}
		userID = uid
	}

	vars := mux.Vars(r)
	noteID, err := uuid.FromString(vars["noteID"])
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to upgrade connection to WebSocket", "error", err)
		return
	}

	h.hub.AddClient(userID, noteID, conn)
}
