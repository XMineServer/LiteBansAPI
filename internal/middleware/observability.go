package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xmine/litebans-api/internal/logging"
	"xmine/litebans-api/internal/metrics"
)

// Observability wraps every request with a request id, structured logging
// and Prometheus metrics, labeled by route pattern (not raw path) to keep
// metric cardinality bounded for parameterized routes like /punishments/{type}/{id}.
func Observability(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			reqID := r.Header.Get("X-Request-Id")
			if reqID == "" {
				reqID = NewRequestID()
			}
			w.Header().Set("X-Request-Id", reqID)

			ctx := WithRequestID(r.Context(), reqID)
			reqLog := log.With(slog.String("request_id", reqID))
			ctx = logging.IntoContext(ctx, reqLog)
			r = r.WithContext(ctx)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			duration := time.Since(start)
			path := routeLabel(r)
			status := strconv.Itoa(rec.status)

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			metrics.HTTPRequestDurationSeconds.WithLabelValues(r.Method, path, status).Observe(duration.Seconds())

			level := slog.LevelInfo
			switch {
			case rec.status >= 500:
				level = slog.LevelError
			case rec.status >= 400:
				level = slog.LevelWarn
			}

			reqLog.LogAttrs(r.Context(), level, "http request",
				slog.String("method", r.Method),
				slog.String("path", path),
				slog.Int("status", rec.status),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

func routeLabel(r *http.Request) string {
	pattern := r.Pattern
	if pattern == "" {
		return r.URL.Path
	}
	if _, path, ok := strings.Cut(pattern, " "); ok {
		return path
	}
	return pattern
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}
