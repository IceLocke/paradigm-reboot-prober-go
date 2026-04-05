package logging

import (
	"context"
	"log/slog"
)

type ctxKey string

const slogFields ctxKey = "slog_fields"

// AppendCtx adds slog attributes to the provided context so that they will be
// automatically included in any log record created with such context.
func AppendCtx(parent context.Context, attrs ...slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if existing, ok := parent.Value(slogFields).([]slog.Attr); ok {
		v := make([]slog.Attr, len(existing), len(existing)+len(attrs))
		copy(v, existing)
		v = append(v, attrs...)
		return context.WithValue(parent, slogFields, v)
	}

	v := make([]slog.Attr, len(attrs))
	copy(v, attrs)
	return context.WithValue(parent, slogFields, v)
}

// attrsFromCtx extracts slog attributes stored in the context.
func attrsFromCtx(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		return attrs
	}
	return nil
}
