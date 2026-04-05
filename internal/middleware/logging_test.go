package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Generates request ID when none provided", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
		assert.Len(t, requestID, 16) // 8 bytes = 16 hex chars
	})

	t.Run("Reuses incoming X-Request-ID", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", "custom-request-id-123")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "custom-request-id-123", w.Header().Get("X-Request-ID"))
	})
}

func TestSlogRequestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Logs request and returns correct status", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.Use(SlogRequestMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
		// Request ID should be set by RequestIDMiddleware
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})

	t.Run("Handles 404 status", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.Use(SlogRequestMiddleware())
		// No routes registered → 404

		req, _ := http.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Handles 500 status", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.Use(SlogRequestMiddleware())
		r.GET("/error", func(c *gin.Context) {
			c.String(http.StatusInternalServerError, "error")
		})

		req, _ := http.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
