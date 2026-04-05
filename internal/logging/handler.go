package logging

import (
	"context"
	"log/slog"
)

// ContextHandler wraps any slog.Handler and automatically extracts slog
// attributes stored in the context (via AppendCtx) before delegating to the
// inner handler.
type ContextHandler struct {
	inner slog.Handler
}

// NewContextHandler creates a ContextHandler wrapping the given handler.
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

// Enabled reports whether the handler handles records at the given level.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle adds contextual attributes from the context to the Record before
// calling the underlying handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs := attrsFromCtx(ctx); len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}
	return h.inner.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler whose inner handler has the given
// attributes pre-applied.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup returns a new ContextHandler whose inner handler uses the given
// group name.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}
