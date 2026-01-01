package middleware

import (
	"net/http"
	"net/http/httptest"
	"paradigm-reboot-prober-go/pkg/auth"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupAuth      func() string
		expectedStatus int
		expectedBody   string
		checkContext   bool
	}{
		{
			name: "No Authorization Header",
			setupAuth: func() string {
				return ""
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header is required"}`,
		},
		{
			name: "Invalid Header Format - Missing Bearer",
			setupAuth: func() string {
				return "InvalidToken"
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid authorization header format"}`,
		},
		{
			name: "Invalid Header Format - Wrong Scheme",
			setupAuth: func() string {
				return "Basic somebase64"
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid authorization header format"}`,
		},
		{
			name: "Invalid Token",
			setupAuth: func() string {
				return "Bearer invalid.token.string"
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid or expired token"}`,
		},
		{
			name: "Expired Token",
			setupAuth: func() string {
				duration := -1 * time.Minute
				token, _ := auth.GenerateAccessJWT("testuser", &duration)
				return "Bearer " + token
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid or expired token"}`,
		},
		{
			name: "Valid Token",
			setupAuth: func() string {
				token, _ := auth.GenerateAccessJWT("testuser", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
			checkContext:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router
			r := gin.New()
			r.Use(AuthMiddleware())
			r.GET("/", func(c *gin.Context) {
				if tt.checkContext {
					username, exists := c.Get("username")
					assert.True(t, exists)
					assert.Equal(t, "testuser", username)
				}
				c.String(http.StatusOK, "OK")
			})

			// Create request
			req, _ := http.NewRequest("GET", "/", nil)
			authHeader := tt.setupAuth()
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			// Perform request
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "OK" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}
