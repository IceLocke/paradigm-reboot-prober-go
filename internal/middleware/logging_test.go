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
		r.Use(SlogRequestMiddleware(nil))
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
		r.Use(SlogRequestMiddleware(nil))
		// No routes registered → 404

		req, _ := http.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Handles 500 status", func(t *testing.T) {
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.Use(SlogRequestMiddleware(nil))
		r.GET("/error", func(c *gin.Context) {
			c.String(http.StatusInternalServerError, "error")
		})

		req, _ := http.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestPathExcluded(t *testing.T) {
	t.Run("Matches prefix", func(t *testing.T) {
		assert.True(t, pathExcluded("/healthz", []string{"/healthz"}))
		assert.True(t, pathExcluded("/healthz/ready", []string{"/healthz"}))
		assert.True(t, pathExcluded("/metrics/foo", []string{"/metrics"}))
	})

	t.Run("Does not match unrelated paths", func(t *testing.T) {
		assert.False(t, pathExcluded("/api/v2/songs", []string{"/healthz"}))
		assert.False(t, pathExcluded("/health", []string{"/healthz"}))
		assert.False(t, pathExcluded("/", []string{"/healthz"}))
	})

	t.Run("Empty prefix list never matches", func(t *testing.T) {
		assert.False(t, pathExcluded("/anything", nil))
		assert.False(t, pathExcluded("/anything", []string{}))
	})

	t.Run("Empty prefix entry is ignored", func(t *testing.T) {
		assert.False(t, pathExcluded("/anything", []string{""}))
	})

	t.Run("SlogRequestMiddleware still serves excluded paths", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(RequestIDMiddleware())
		r.Use(SlogRequestMiddleware([]string{"/healthz"}))
		r.GET("/healthz", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req, _ := http.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Request is still handled normally; we only suppress the access log line.
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})
}
