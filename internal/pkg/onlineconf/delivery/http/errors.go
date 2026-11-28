package delivery

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"
	"bnn/internal/pkg/onlineconf/delivery/http/api"
	"bnn/internal/pkg/utils/httpjson"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	codeNotFound        = "NOT_FOUND"
	codeKeyExists       = "KEY_EXISTS"
	codeVersionConflict = "VERSION_CONFLICT"
	codeInternal        = "INTERNAL"
	codeUnavailable     = "SERVICE_UNAVAILABLE"
	codePayloadTooLarge = "PAYLOAD_TOO_LARGE"
)

func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/onlineconf/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func keyHTTPError(ctx context.Context, err error) (int, api.OnlineConfError) {
	switch {
	case errors.Is(err, models.ErrOnlineConfRolloutPlanNotFound):
		return http.StatusNotFound, api.OnlineConfError{Code: codeNotFound, Message: err.Error()}
	case errors.Is(err, models.ErrOnlineConfScheduleNotFound):
		return http.StatusNotFound, api.OnlineConfError{Code: codeNotFound, Message: err.Error()}
	case errors.Is(err, models.ErrOnlineConfRolloutPlanExists):
		return http.StatusConflict, api.OnlineConfError{Code: onlineconf.CodeRolloutPlanExists, Message: err.Error()}
	case errors.Is(err, models.ErrOnlineConfRolloutPlanInUse):
		return http.StatusConflict, api.OnlineConfError{Code: onlineconf.CodeRolloutPlanInUse, Message: err.Error()}
	case errors.Is(err, models.ErrOnlineConfRolloutNotReady):
		return http.StatusConflict, api.OnlineConfError{Code: onlineconf.CodeRolloutNotReady, Message: err.Error()}
	case errors.Is(err, models.ErrOnlineConfScheduleFinished):
		return http.StatusConflict, api.OnlineConfError{Code: onlineconf.CodeScheduleFinished, Message: err.Error()}
	case errors.Is(err, models.ErrOnlineConfRolloutStepNotAllowed):
		return http.StatusBadRequest, api.OnlineConfError{Code: onlineconf.CodeRolloutStepNotAllowed, Message: err.Error()}
	case errors.Is(err, models.ErrValidation):
		return http.StatusBadRequest, api.OnlineConfError{
			Code: onlineconf.CodeValidation, Message: err.Error(),
		}

	case errors.Is(err, models.ErrForbidden):
		return http.StatusForbidden, api.OnlineConfError{
			Code: onlineconf.CodeForbidden, Message: messageForbidden,
		}

	case errors.Is(err, models.ErrOnlineConfKeyNotFound):
		return http.StatusNotFound, api.OnlineConfError{
			Code: codeNotFound, Message: "key not found",
		}

	case errors.Is(err, models.ErrOnlineConfKeyExists):
		return http.StatusConflict, api.OnlineConfError{
			Code: codeKeyExists, Message: "key already exists in this scope",
		}

	case errors.Is(err, models.ErrOnlineConfVersionConflict):
		return http.StatusConflict, api.OnlineConfError{
			Code:    codeVersionConflict,
			Message: "key version changed; reload the key and retry",
		}

	case errors.Is(err, models.ErrOnlineConfSnapshotNotReady),
		errors.Is(err, context.DeadlineExceeded),
		status.Code(err) == codes.Unavailable,
		status.Code(err) == codes.DeadlineExceeded:
		return http.StatusServiceUnavailable, api.OnlineConfError{
			Code: codeUnavailable, Message: "configuration service unavailable",
		}

	default:
		slog.ErrorContext(ctx, "OnlineConf request failed", "error", err)
		return http.StatusInternalServerError, api.OnlineConfError{
			Code: codeInternal, Message: "internal error",
		}
	}
}

func RequestError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Cache-Control", "no-store")

	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		httpjson.WriteError(
			w, http.StatusRequestEntityTooLarge,
			codePayloadTooLarge, "request body is too large",
		)
		return
	}

	httpjson.WriteError(
		w, http.StatusBadRequest, onlineconf.CodeValidation, err.Error(),
	)
}

func ResponseError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Cache-Control", "no-store")

	statusCode, body := keyHTTPError(r.Context(), err)
	httpjson.WriteError(w, statusCode, body.Code, body.Message)
}
