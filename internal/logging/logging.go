// Package logging provides the unified structured slog logger used across the
// XMine services: JSON or text output, level-filtered, with records at Error
// and above routed to stderr and everything else to stdout.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// New builds the service logger.
//   - service: static "service" field value (e.g. "litebans-api")
//   - level:   minimum level to emit
//   - format:  text or json (json on unknown/zero value)
//
// Records at LevelError and above are written to stderr; everything else goes
// to stdout. Source location is added only at debug level.
func New(service string, level slog.Level, format LogFormat) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	handler := newSplitHandler(
		newHandler(os.Stdout, format, opts),
		newHandler(os.Stderr, format, opts),
	)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return slog.New(handler).With(
		slog.String("service", service),
		slog.String("hostname", hostname),
	)
}

func newHandler(w io.Writer, format LogFormat, opts *slog.HandlerOptions) slog.Handler {
	if format == LogFormatText {
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(w, opts)
}

// splitHandler routes records at LevelError and above to errHandler (stderr)
// and everything below to outHandler (stdout).
type splitHandler struct {
	out slog.Handler
	err slog.Handler
}

func newSplitHandler(out, err slog.Handler) slog.Handler {
	return splitHandler{out: out, err: err}
}

func (h splitHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.out.Enabled(ctx, level) || h.err.Enabled(ctx, level)
}

func (h splitHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		return h.err.Handle(ctx, r)
	}
	return h.out.Handle(ctx, r)
}

func (h splitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return splitHandler{out: h.out.WithAttrs(attrs), err: h.err.WithAttrs(attrs)}
}

func (h splitHandler) WithGroup(name string) slog.Handler {
	return splitHandler{out: h.out.WithGroup(name), err: h.err.WithGroup(name)}
}

// WithComponent tags a logger with the subsystem that emits a log line
// (e.g. "auth", "httpapi", "repository").
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With(slog.String("component", component))
}

type ctxKey struct{}

// IntoContext stores logger in ctx for retrieval via FromContext.
func IntoContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext returns the logger stored in ctx, or fallback if none is set.
func FromContext(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return fallback
}
