package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/logging"
)

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeError maps a domain error (or any other error) to a JSON error response,
// used as the StrictHTTPServerOptions.ResponseErrorHandlerFunc for the generated server.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	log := logging.FromContext(r.Context(), slog.Default())

	var derr *domain.Error
	if errors.As(err, &derr) {
		status := httpStatusForCode(derr.Code)
		if status >= 500 {
			log.Error("request failed", slog.String("code", string(derr.Code)), slog.Any("error", derr.Err))
		}
		writeJSON(log, w, status, errorBody{Error: string(derr.Code), Message: derr.Message})
		return
	}
	log.Error("unexpected error", slog.Any("error", err))
	writeJSON(log, w, http.StatusInternalServerError, errorBody{Error: "INTERNAL_ERROR", Message: "internal server error"})
}

func writeJSON(log *slog.Logger, w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode response body", slog.Any("error", err))
	}
}

func httpStatusForCode(code domain.ErrorCode) int {
	switch code {
	case domain.ErrCodeInvalidUUID, domain.ErrCodeInvalidType, domain.ErrCodeInvalidParameter:
		return http.StatusBadRequest
	case domain.ErrCodeNotFound:
		return http.StatusNotFound
	case domain.ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case domain.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case domain.ErrCodeForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
