package middleware

import (
	"net/http"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/service"
	"paradigm-reboot-prober-go/pkg/auth"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware is a middleware that checks for a valid JWT token.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.Response{Error: "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, model.Response{Error: "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		username, err := auth.ExtractUsername(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.Response{Error: "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set the username in the context so it can be used by subsequent handlers
		c.Set("username", username)
		c.Next()
	}
}

// OptionalAuthMiddleware is a middleware that extracts the username from a JWT token if present, but does not abort if it's missing or invalid.
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		username, err := auth.ExtractUsername(tokenString)
		if err == nil {
			c.Set("username", username)
		}
		c.Next()
	}
}

// AdminMiddleware returns a middleware that checks if the authenticated user is an admin.
func AdminMiddleware(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		usernameStr := c.GetString("username")
		if usernameStr == "" {
			c.JSON(http.StatusUnauthorized, model.Response{Error: "Authentication required"})
			c.Abort()
			return
		}

		user, err := userService.GetUser(usernameStr)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, model.Response{Error: "User not found"})
			c.Abort()
			return
		}

		if !user.IsAdmin {
			c.JSON(http.StatusForbidden, model.Response{Error: "Admin access required"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
