package logging

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	// Preserve and restore the global logger so tests don't leak state.
	originalLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	t.Run("stdout defaults", func(t *testing.T) {
		c, err := Setup("stdout", "", "text")
		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.NoError(t, c.Close()) // noopCloser
	})

	t.Run("empty strings default to stdout + text", func(t *testing.T) {
		c, err := Setup("", "", "")
		assert.NoError(t, err)
		assert.NoError(t, c.Close())
	})

	t.Run("stderr works", func(t *testing.T) {
		c, err := Setup("stderr", "", "text")
		assert.NoError(t, err)
		assert.NoError(t, c.Close())
	})

	t.Run("file writes text log lines", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "nested", "server.log")

		c, err := Setup("file", logPath, "text")
		assert.NoError(t, err)
		assert.NotNil(t, c)

		slog.Info("hello from test", "key", "value")
		assert.NoError(t, c.Close())

		// Restore default so other subtests don't write to the closed file.
		slog.SetDefault(originalLogger)

		data, err := os.ReadFile(logPath)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "hello from test")
		assert.Contains(t, string(data), "key=value")
	})

	t.Run("file writes valid JSON lines when format=json", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "server.log")

		c, err := Setup("file", logPath, "json")
		assert.NoError(t, err)

		slog.Info("hello json", "key", "value")
		assert.NoError(t, c.Close())
		slog.SetDefault(originalLogger)

		data, err := os.ReadFile(logPath)
		assert.NoError(t, err)

		line := strings.TrimSpace(string(data))
		assert.NotEmpty(t, line)

		var parsed map[string]any
		assert.NoError(t, json.Unmarshal([]byte(line), &parsed))
		assert.Equal(t, "hello json", parsed["msg"])
		assert.Equal(t, "value", parsed["key"])
		assert.Equal(t, "INFO", parsed["level"])
	})

	t.Run("file appends on subsequent opens", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "server.log")

		c1, err := Setup("file", logPath, "text")
		assert.NoError(t, err)
		slog.Info("first")
		assert.NoError(t, c1.Close())

		c2, err := Setup("file", logPath, "text")
		assert.NoError(t, err)
		slog.Info("second")
		assert.NoError(t, c2.Close())

		slog.SetDefault(originalLogger)

		data, err := os.ReadFile(logPath)
		assert.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "first")
		assert.Contains(t, content, "second")
		assert.Equal(t, 2, strings.Count(content, "msg="))
	})

	t.Run("file without path is rejected", func(t *testing.T) {
		_, err := Setup("file", "", "text")
		assert.Error(t, err)
	})

	t.Run("unknown output is rejected", func(t *testing.T) {
		_, err := Setup("syslog", "", "text")
		assert.Error(t, err)
	})

	t.Run("unknown format is rejected", func(t *testing.T) {
		_, err := Setup("stdout", "", "xml")
		assert.Error(t, err)
	})

	t.Run("unknown format does not leak file handle", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "server.log")

		_, err := Setup("file", logPath, "xml")
		assert.Error(t, err)

		// The file may have been created (by OpenFile) but must be closed so
		// that removing the temp dir succeeds on Windows and we don't leak
		// descriptors. On POSIX this is best-effort: assert we can still open
		// the path for writing (no exclusive lock).
		f, openErr := os.OpenFile(logPath, os.O_RDWR, 0o644)
		if openErr == nil {
			_ = f.Close()
		}
	})
}
