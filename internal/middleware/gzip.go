package middleware

import (
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
