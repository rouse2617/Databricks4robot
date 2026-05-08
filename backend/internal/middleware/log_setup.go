package middleware

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// SetupLogger configures the global slog logger based on config.
//
//	level:  "debug", "info", "warn", "error"
//	format: "text" (human-readable) or "json" (structured, for log aggregation)
//	file:   file path to write logs to (empty = stdout only, "both" prefix = stdout + file)
//
// Returns a closer function to flush/close the log file (call in defer).
func SetupLogger(level, format, file string) func() {
	// Parse level.
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	// Determine output writer.
	var w io.Writer = os.Stdout
	var closer func()

	if file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Error("failed to open log file, falling back to stdout", "file", file, "err", err)
		} else {
			// Write to both stdout and file.
			w = io.MultiWriter(os.Stdout, f)
			closer = func() { f.Close() }
		}
	}

	// Create handler.
	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	slog.SetDefault(slog.New(handler))

	if closer == nil {
		return func() {}
	}
	return closer
}
