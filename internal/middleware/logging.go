package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"paradigm-reboot-prober-go/internal/logging"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

// RequestIDMiddleware generates a unique request ID for each request (or reuses
// an incoming X-Request-ID header from a reverse proxy). The ID is injected
// into the slog context and set as a response header.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			requestID = hex.EncodeToString(b)
		}

		ctx := logging.AppendCtx(c.Request.Context(), slog.String("request_id", requestID))
		c.Request = c.Request.WithContext(ctx)
		c.Header(requestIDHeader, requestID)

		c.Next()
	}
}

// SlogRequestMiddleware logs each HTTP request with method, path, client IP,
// status code, latency and response size. It must be placed after
// RequestIDMiddleware so that request_id is already in the context.
//
// Log levels: INFO for 1xx–3xx, WARN for 4xx, ERROR for 5xx.
func SlogRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		ctx := logging.AppendCtx(c.Request.Context(),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("client_ip", c.ClientIP()),
		)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		bytesOut := c.Writer.Size()
		if bytesOut < 0 {
			bytesOut = 0
		}

		attrs := []any{
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"bytes_out", bytesOut,
		}

		switch {
		case status >= 500:
			slog.ErrorContext(ctx, "request completed", attrs...)
		case status >= 400:
			slog.WarnContext(ctx, "request completed", attrs...)
		default:
			slog.InfoContext(ctx, "request completed", attrs...)
		}
	}
}
