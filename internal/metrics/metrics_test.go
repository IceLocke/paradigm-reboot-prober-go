package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestEngine(excludePrefixes []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware(excludePrefixes))
	return r
}

func doRequest(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func labels(method, path, status string) map[string]string {
	return map[string]string{"method": method, "path": path, "status": status}
}

func TestMiddleware_Counts2xx(t *testing.T) {
	r := newTestEngine(nil)
	r.GET("/api/v2/songs", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	before := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/api/v2/songs", "200")))
	beforeDur := testutil.CollectAndCount(httpRequestDurationSeconds, "http_request_duration_seconds")
	beforeSize := testutil.CollectAndCount(httpResponseSizeBytes, "http_response_size_bytes")

	w := doRequest(r, "GET", "/api/v2/songs")
	assert.Equal(t, http.StatusOK, w.Code)

	after := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/api/v2/songs", "200")))
	afterDur := testutil.CollectAndCount(httpRequestDurationSeconds, "http_request_duration_seconds")
	afterSize := testutil.CollectAndCount(httpResponseSizeBytes, "http_response_size_bytes")

	assert.Equal(t, before+1, after, "counter should increment by 1")
	assert.GreaterOrEqual(t, afterDur, beforeDur, "duration histogram series count must not shrink")
	assert.GreaterOrEqual(t, afterSize, beforeSize, "response-size histogram series count must not shrink")
}

func TestMiddleware_Counts5xx(t *testing.T) {
	r := newTestEngine(nil)
	r.GET("/boom", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "boom")
	})

	before := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/boom", "500")))
	doRequest(r, "GET", "/boom")
	after := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/boom", "500")))

	assert.Equal(t, before+1, after)
}

func TestMiddleware_UnmatchedRouteIsUnknown(t *testing.T) {
	r := newTestEngine(nil)
	// No routes registered → 404, FullPath() is empty, we label as "unknown".

	before := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "unknown", "404")))
	w := doRequest(r, "GET", "/totally/made/up")
	assert.Equal(t, http.StatusNotFound, w.Code)
	after := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "unknown", "404")))

	assert.Equal(t, before+1, after)
}

func TestMiddleware_InFlightReturnsToZero(t *testing.T) {
	r := newTestEngine(nil)
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	baseline := testutil.ToFloat64(httpRequestsInFlight)
	for i := 0; i < 5; i++ {
		doRequest(r, "GET", "/ping")
	}
	// After all requests complete synchronously, the gauge must return to the
	// baseline (0 if no other test runs concurrently, or whatever it was).
	assert.Equal(t, baseline, testutil.ToFloat64(httpRequestsInFlight))
}

func TestMiddleware_ExcludedPathNotCounted(t *testing.T) {
	r := newTestEngine([]string{"/healthz"})
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	before := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/healthz", "200")))
	w := doRequest(r, "GET", "/healthz")
	assert.Equal(t, http.StatusOK, w.Code)
	after := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/healthz", "200")))

	assert.Equal(t, before, after, "excluded path must not increment the counter")
}

func TestMiddleware_UsesRouteTemplateNotRawPath(t *testing.T) {
	r := newTestEngine(nil)
	r.GET("/api/v2/records/:username", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	before := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/api/v2/records/:username", "200")))
	doRequest(r, "GET", "/api/v2/records/alice")
	doRequest(r, "GET", "/api/v2/records/bob")
	after := testutil.ToFloat64(httpRequestsTotal.With(labels("GET", "/api/v2/records/:username", "200")))

	assert.Equal(t, before+2, after, "both requests must collapse into the :username template label")
}

func TestHandler_ExposesMetrics(t *testing.T) {
	// Drive one real observation so the exposition output is non-trivial.
	r := newTestEngine(nil)
	r.GET("/hello", func(c *gin.Context) { c.String(http.StatusOK, "hi") })
	doRequest(r, "GET", "/hello")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body, _ := io.ReadAll(w.Body)
	text := string(body)
	// HTTP metrics.
	assert.Contains(t, text, "http_requests_total")
	assert.Contains(t, text, "http_request_duration_seconds")
	assert.Contains(t, text, "http_requests_in_flight")
	assert.Contains(t, text, "http_response_size_bytes")
	// Free runtime/process metrics from default registry.
	assert.Contains(t, text, "go_goroutines")
	assert.True(t,
		strings.Contains(text, "process_resident_memory_bytes") ||
			strings.Contains(text, "process_cpu_seconds_total"),
		"expected process collector metrics in the exposition output",
	)
}

func TestPathExcluded(t *testing.T) {
	t.Run("matches prefix", func(t *testing.T) {
		assert.True(t, pathExcluded("/healthz", []string{"/healthz"}))
		assert.True(t, pathExcluded("/healthz/ready", []string{"/healthz"}))
	})
	t.Run("no match", func(t *testing.T) {
		assert.False(t, pathExcluded("/api/v2/songs", []string{"/healthz"}))
	})
	t.Run("nil and empty list", func(t *testing.T) {
		assert.False(t, pathExcluded("/anything", nil))
		assert.False(t, pathExcluded("/anything", []string{}))
	})
	t.Run("empty prefix ignored", func(t *testing.T) {
		assert.False(t, pathExcluded("/anything", []string{""}))
	})
}
