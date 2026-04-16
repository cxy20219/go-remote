package agent

import (
	"log/slog"
	"os"
)

// InitLogger initializes the slog logger
func InitLogger(level, file string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	if file == "" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Warn("failed to open log file, using stdout", "error", err)
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			handler = slog.NewJSONHandler(f, opts)
		}
	}

	slog.SetDefault(slog.New(handler))
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
