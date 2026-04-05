package middleware

import (
	"log/slog"
	"net/http"
	"paradigm-reboot-prober-go/internal/logging"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/service"
	"paradigm-reboot-prober-go/pkg/auth"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks for a valid JWT token and verifies the user still
// exists and is active.
func AuthMiddleware(userService *service.UserService) gin.HandlerFunc {
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

		// Verify the user still exists and is active
		user, err := userService.GetUser(username)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, model.Response{Error: "user not found"})
			c.Abort()
			return
		}
		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, model.Response{Error: "user account is deactivated"})
			c.Abort()
			return
		}

		// Inject username into slog context for automatic inclusion in all downstream logs
		ctx := logging.AppendCtx(c.Request.Context(), slog.String("username", username))
		c.Request = c.Request.WithContext(ctx)
		c.Set("username", username)
		c.Next()
	}
}

// OptionalAuthMiddleware extracts the username from a JWT token if present.
// It does not abort when the header is missing or invalid, but skips
// inactive or nonexistent users.
func OptionalAuthMiddleware(userService *service.UserService) gin.HandlerFunc {
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
			user, err := userService.GetUser(username)
			if err == nil && user != nil && user.IsActive {
				ctx := logging.AppendCtx(c.Request.Context(), slog.String("username", username))
				c.Request = c.Request.WithContext(ctx)
				c.Set("username", username)
			}
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
