package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupGzipRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GzipRequestMiddleware())
	r.Use(GzipResponseMiddleware())

	r.POST("/echo", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, string(body))
	})

	r.GET("/hello", func(c *gin.Context) {
		// Return a long-enough body so gzip actually compresses it.
		c.String(http.StatusOK, strings.Repeat("hello world! ", 100))
	})

	return r
}

func gzipCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

func gzipDecompress(t *testing.T, data []byte) []byte {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(data))
	assert.NoError(t, err)
	defer func() { _ = r.Close() }()
	out, err := io.ReadAll(r)
	assert.NoError(t, err)
	return out
}

// ---------------------------------------------------------------------------
// Tests – Response Compression
// ---------------------------------------------------------------------------

func TestGzipResponse_CompressesWhenAccepted(t *testing.T) {
	router := setupGzipRouter()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

	body := gzipDecompress(t, w.Body.Bytes())
	assert.Equal(t, strings.Repeat("hello world! ", 100), string(body))
}

func TestGzipResponse_NoCompressWithoutAcceptEncoding(t *testing.T) {
	router := setupGzipRouter()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	// Explicitly remove Accept-Encoding
	req.Header.Del("Accept-Encoding")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEqual(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Equal(t, strings.Repeat("hello world! ", 100), w.Body.String())
}

// ---------------------------------------------------------------------------
// Tests – Request Decompression
// ---------------------------------------------------------------------------

func TestGzipRequest_DecompressesBody(t *testing.T) {
	router := setupGzipRouter()
	payload := `{"username":"tester","score":1000000}`
	compressed := gzipCompress([]byte(payload))

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, payload, w.Body.String())
}

func TestGzipRequest_PassesThroughUncompressed(t *testing.T) {
	router := setupGzipRouter()
	payload := `{"plain":"body"}`

	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, payload, w.Body.String())
}

func TestGzipRequest_InvalidGzipReturns400(t *testing.T) {
	router := setupGzipRouter()

	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("not gzip data"))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
