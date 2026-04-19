// Package metrics defines the application's Prometheus metrics and the Gin
// middleware that records HTTP request observations against them.
//
// All HTTP metrics use Gin's matched route template (c.FullPath()) as the
// `path` label — never the raw URL path — to keep label cardinality bounded.
// Unmatched routes (404s) are reported as path="unknown".
//
// The metrics are registered on the default prometheus registerer, which also
// auto-registers GoCollector and ProcessCollector, so scraping /metrics gives
// Go runtime and process stats for free.
package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests handled, partitioned by method, route template and response status code.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds, partitioned by method, route template and response status code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpResponseSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP response bodies in bytes, partitioned by method, route template and response status code.",
			Buckets: []float64{64, 256, 1024, 4 * 1024, 16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024},
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being served.",
		},
	)
)

// Middleware returns a Gin middleware that records HTTP request metrics.
//
// excludePrefixes is a list of route-template prefixes that should NOT be
// observed (e.g. "/healthz"). Excluded requests are still served normally.
//
// It should be installed as one of the outermost middlewares so that the
// timing covers the full handling of the request (in practice: right after
// RequestID/SlogRequest, before CORS/Gzip).
func Middleware(excludePrefixes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		if pathExcluded(path, excludePrefixes) {
			return
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		labels := prometheus.Labels{
			"method": method,
			"path":   path,
			"status": status,
		}

		httpRequestsTotal.With(labels).Inc()
		httpRequestDurationSeconds.With(labels).Observe(time.Since(start).Seconds())

		bytesOut := c.Writer.Size()
		if bytesOut < 0 {
			bytesOut = 0
		}
		httpResponseSizeBytes.With(labels).Observe(float64(bytesOut))
	}
}

// Handler returns an http.Handler that serves metrics in the Prometheus text
// exposition format. Callers should mount this on a dedicated HTTP server
// (separate from the main API server) so metrics are not exposed on the
// user-facing port.
func Handler() http.Handler {
	return promhttp.Handler()
}

// pathExcluded reports whether path starts with any of the given prefixes.
// Empty prefix entries are ignored.
func pathExcluded(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
