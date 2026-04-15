package middleware

import (
	"net/http"
	"net/http/httptest"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/repository"
	"paradigm-reboot-prober-go/internal/service"
	"paradigm-reboot-prober-go/pkg/auth"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupAuthTest(t *testing.T) *service.UserService {
	t.Helper()
	config.InitDefaults()
	config.GlobalConfig.Auth.SecretKey = "testsecret"

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	encoded, _ := auth.EncodePassword("password")

	// Active user
	db.Create(&model.User{
		UserBase: model.UserBase{
			Username:    "testuser",
			Email:       "test@test.com",
			IsActive:    true,
			UploadToken: "tok",
		},
		EncodedPassword: encoded,
	})

	// Inactive user — IsActive has gorm default:true, so Create with false
	// would be skipped as a zero-value. Create first, then update explicitly.
	inactiveUser := &model.User{
		UserBase: model.UserBase{
			Username:    "inactiveuser",
			Email:       "inactive@test.com",
			IsActive:    true,
			UploadToken: "tok2",
		},
		EncodedPassword: encoded,
	}
	db.Create(inactiveUser)
	db.Model(inactiveUser).Update("is_active", false)

	// Admin user
	db.Create(&model.User{
		UserBase: model.UserBase{
			Username:    "adminuser",
			Email:       "admin@test.com",
			IsActive:    true,
			IsAdmin:     true,
			UploadToken: "tok3",
		},
		EncodedPassword: encoded,
	})

	userRepo := repository.NewUserRepository(db)
	return service.NewUserService(userRepo)
}

func TestOptionalAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService := setupAuthTest(t)

	tests := []struct {
		name           string
		setupAuth      func() string
		expectedStatus int
		expectUsername bool
	}{
		{
			name: "No Authorization Header",
			setupAuth: func() string {
				return ""
			},
			expectedStatus: http.StatusOK,
			expectUsername: false,
		},
		{
			name: "Invalid Token",
			setupAuth: func() string {
				return "Bearer invalid.token.string"
			},
			expectedStatus: http.StatusOK,
			expectUsername: false,
		},
		{
			name: "Valid Token - Active User",
			setupAuth: func() string {
				token, _ := auth.GenerateAccessJWT("testuser", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusOK,
			expectUsername: true,
		},
		{
			name: "Valid Token - Inactive User",
			setupAuth: func() string {
				token, _ := auth.GenerateAccessJWT("inactiveuser", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusOK,
			expectUsername: false,
		},
		{
			name: "Valid Token - Non-existent User",
			setupAuth: func() string {
				token, _ := auth.GenerateAccessJWT("nonexistent", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusOK,
			expectUsername: false,
		},
		{
			name: "Refresh Token - Rejected",
			setupAuth: func() string {
				token, _ := auth.GenerateRefreshJWT("testuser", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusOK,
			expectUsername: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(OptionalAuthMiddleware(userService))
			r.GET("/", func(c *gin.Context) {
				_, exists := c.Get("username")
				assert.Equal(t, tt.expectUsername, exists)
				c.String(http.StatusOK, "OK")
			})

			req, _ := http.NewRequest("GET", "/", nil)
			authHeader := tt.setupAuth()
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAdminMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService := setupAuthTest(t)

	tests := []struct {
		name           string
		setUsername    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "No username in context",
			setUsername:    "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authentication required"}`,
		},
		{
			name:           "Valid admin user",
			setUsername:    "adminuser",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid non-admin user",
			setUsername:    "testuser",
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Admin access required"}`,
		},
		{
			name:           "Non-existent user",
			setUsername:    "nonexistent",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"User not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			if tt.setUsername != "" {
				r.Use(func(c *gin.Context) {
					c.Set("username", tt.setUsername)
					c.Next()
				})
			}
			r.Use(AdminMiddleware(userService))
			r.GET("/", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req, _ := http.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userService := setupAuthTest(t)

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
		{
			name: "Valid Token - User Not Found",
			setupAuth: func() string {
				token, _ := auth.GenerateAccessJWT("nonexistent", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"user not found"}`,
		},
		{
			name: "Valid Token - Inactive User",
			setupAuth: func() string {
				token, _ := auth.GenerateAccessJWT("inactiveuser", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"user account is deactivated"}`,
		},
		{
			name: "Refresh Token - Rejected",
			setupAuth: func() string {
				token, _ := auth.GenerateRefreshJWT("testuser", nil)
				return "Bearer " + token
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid token type"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router
			r := gin.New()
			r.Use(AuthMiddleware(userService))
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
