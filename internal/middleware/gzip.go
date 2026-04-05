package middleware

import (
	"net/http"

	gzipgin "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// GzipResponseMiddleware returns a middleware that compresses HTTP responses
// using gzip when the client indicates support via the Accept-Encoding header.
func GzipResponseMiddleware() gin.HandlerFunc {
	return gzipgin.Gzip(gzipgin.DefaultCompression)
}

// GzipRequestMiddleware returns a middleware that transparently decompresses
// gzip-encoded request bodies (Content-Encoding: gzip).
func GzipRequestMiddleware() gin.HandlerFunc {
	return gzipgin.DefaultDecompressHandle
}

// MaxRequestBodyMiddleware limits the size of request bodies to maxBytes.
// When placed after GzipRequestMiddleware it caps the decompressed payload,
// guarding against gzip bombs (a tiny compressed body that expands to gigabytes).
func MaxRequestBodyMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
