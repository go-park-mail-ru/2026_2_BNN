package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"bnn/internal/pkg/auth"
)

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	slog.Info("Processing get current user request")

	userID, ok := auth.GetUserID(r)
	if !ok {
		slog.Warn("User not authenticated")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, exists := auth.GetUserByID(userID)
	if !exists {
		slog.Warn("User not found", "user_id", userID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, err := json.Marshal(user)
	if err != nil {
		slog.Error("Failed to marshal user", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(body); err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}
