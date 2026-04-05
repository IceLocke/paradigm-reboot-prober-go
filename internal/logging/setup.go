package logging

import (
	"log/slog"
	"os"
)

// Setup configures the global slog default logger with a TextHandler wrapped
// by ContextHandler for automatic context field extraction.
// Call this once at application startup before any logging occurs.
func Setup() {
	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	handler := NewContextHandler(textHandler)
	slog.SetDefault(slog.New(handler))
}
