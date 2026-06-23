package logger

import (
	"log/slog"
	"os"
)

func getHandler(logFormat LogFormat, opts *slog.HandlerOptions) slog.Handler {
	switch logFormat {
	case LogFormatJSON:
		return slog.NewJSONHandler(os.Stdout, opts)
	case LogFormatText:
		return slog.NewTextHandler(os.Stdout, opts)
	default:
		return slog.NewTextHandler(os.Stdout, opts)
	}
}

func SetupLogger(logFormat LogFormat, level slog.Level) {
	hostname, err := os.Hostname()

	if err != nil {
		hostname = "unknown"
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler = getHandler(logFormat, opts)

	logger := slog.New(handler).With(
		slog.String("service", "litebans-api"),
		slog.String("hostname", hostname),
	)

	slog.SetDefault(logger)

	slog.Info("Logger initialized",
		slog.String("level", level.String()),
		slog.Any("format", logFormat),
	)

	if err != nil {
		slog.Warn("Failed to get hostname, using 'unknown'", slog.String("error", err.Error()))
	}

}
