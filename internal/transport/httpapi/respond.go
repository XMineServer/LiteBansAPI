package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"xmine/litebans-api/internal/domain"
)

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode response body", slog.Any("error", err))
	}
}

func writeError(w http.ResponseWriter, err error) {
	var derr *domain.Error
	if errors.As(err, &derr) {
		status := httpStatusForCode(derr.Code)
		if status >= 500 {
			slog.Error("request failed", slog.String("code", string(derr.Code)), slog.Any("error", derr.Err))
		}
		writeJSON(w, status, errorBody{Error: string(derr.Code), Message: derr.Message})
		return
	}
	slog.Error("unexpected error", slog.Any("error", err))
	writeJSON(w, http.StatusInternalServerError, errorBody{Error: "INTERNAL_ERROR", Message: "internal server error"})
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
