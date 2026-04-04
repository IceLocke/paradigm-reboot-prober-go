package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"paradigm-reboot-prober-go/internal/model"

	gzipgin "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// GzipResponseMiddleware returns a middleware that compresses HTTP responses
// using gzip when the client indicates support via the Accept-Encoding header.
func GzipResponseMiddleware() gin.HandlerFunc {
	return gzipgin.Gzip(gzipgin.DefaultCompression)
}

// GzipRequestMiddleware returns a middleware that transparently decompresses
// gzip-encoded request bodies. When a request carries "Content-Encoding: gzip",
// the body is wrapped in a gzip reader so downstream handlers can read the
// original payload without any extra work.
func GzipRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.EqualFold(c.GetHeader("Content-Encoding"), "gzip") {
			c.Next()
			return
		}

		reader, err := gzip.NewReader(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.Response{
				Error: fmt.Sprintf("Failed to decompress gzip request body: %v", err),
			})
			c.Abort()
			return
		}

		// Replace the request body with the decompressed reader.
		c.Request.Body = io.NopCloser(reader)
		// Content-Length is no longer accurate after decompression.
		c.Request.ContentLength = -1
		// Remove Content-Encoding so downstream handlers do not attempt
		// to decompress again.
		c.Request.Header.Del("Content-Encoding")

		defer func() { _ = reader.Close() }()

		c.Next()
	}
}
