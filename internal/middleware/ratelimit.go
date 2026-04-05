package middleware

import (
	"net/http"
	"paradigm-reboot-prober-go/internal/model"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware returns a per-IP rate limiter backed by golang.org/x/time/rate.
// maxAttempts requests are allowed within each window per IP address.
func RateLimitMiddleware(maxAttempts int, window time.Duration) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var mu sync.Mutex
	clients := make(map[string]*client)

	r := rate.Every(window / time.Duration(maxAttempts))

	// Background cleanup of stale entries
	go func() {
		for {
			time.Sleep(window)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > window*2 {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists {
			cl = &client{limiter: rate.NewLimiter(r, maxAttempts)}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		mu.Unlock()

		if !cl.limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, model.Response{Error: "too many requests, please try again later"})
			c.Abort()
			return
		}
		c.Next()
	}
}
