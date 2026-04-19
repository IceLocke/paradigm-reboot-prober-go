package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// noopCloser wraps an io.Writer (stdout/stderr) so callers can always defer Close
// without having to branch on the output type.
type noopCloser struct{}

func (noopCloser) Close() error { return nil }

// Setup configures the global slog default logger with the selected handler
// (text or json) wrapped by ContextHandler for automatic context field
// extraction.
//
// output must be one of "stdout" (default), "stderr", or "file". When output is
// "file", filePath is required and the file is opened in append mode (created
// if missing). The returned io.Closer must be closed at application shutdown.
//
// format must be "text" (default) or "json". An empty string is treated as
// "text" for backwards compatibility.
//
// Call this once at application startup before any logging occurs.
func Setup(output, filePath, format string) (io.Closer, error) {
	var (
		writer io.Writer
		closer io.Closer = noopCloser{}
	)

	switch output {
	case "", "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "file":
		if filePath == "" {
			return nil, fmt.Errorf("logging: output=file but file path is empty")
		}
		if dir := filepath.Dir(filePath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("logging: create log dir %q: %w", dir, err)
			}
		}
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("logging: open log file %q: %w", filePath, err)
		}
		writer = f
		closer = f
	default:
		return nil, fmt.Errorf("logging: invalid output %q (expected stdout, stderr, or file)", output)
	}

	opts := &slog.HandlerOptions{Level: slog.LevelInfo}

	var inner slog.Handler
	switch format {
	case "", "text":
		inner = slog.NewTextHandler(writer, opts)
	case "json":
		inner = slog.NewJSONHandler(writer, opts)
	default:
		// Close the file we may have just opened to avoid leaking a handle.
		_ = closer.Close()
		return nil, fmt.Errorf("logging: invalid format %q (expected text or json)", format)
	}

	handler := NewContextHandler(inner)
	slog.SetDefault(slog.New(handler))
	return closer, nil
}
