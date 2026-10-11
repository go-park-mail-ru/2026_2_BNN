package delivery

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"bnn/internal/pkg/ctxutil"
	"bnn/internal/pkg/httpjson"
	"bnn/internal/pkg/notes"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

type Handler struct {
	uc notes.Usecase
}

func NewHandler(uc notes.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxutil.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	limit, offset, err := parsePagination(r.URL.Query())
	if err != nil {
		slog.Warn("Invalid pagination parameters", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := h.uc.GetNotes(r.Context(), userID, limit, offset)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, result)
}

func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := ctxutil.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	noteID, err := uuid.FromString(mux.Vars(r)["id"])
	if err != nil {
		slog.Warn("Invalid note id", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	note, err := h.uc.GetNote(r.Context(), userID, noteID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, note)
}

func parsePagination(query url.Values) (int64, int64, error) {
	limit := int64(defaultLimit)
	offset := int64(0)

	if query.Has("limit") {
		value, err := strconv.ParseInt(query.Get("limit"), 10, 64)
		if err != nil || value < 1 || value > maxLimit {
			return 0, 0, errors.New("invalid limit")
		}

		limit = value
	}

	if query.Has("offset") {
		value, err := strconv.ParseInt(query.Get("offset"), 10, 64)
		if err != nil || value < 0 {
			return 0, 0, errors.New("invalid offset")
		}

		offset = value
	}

	return limit, offset, nil
}
