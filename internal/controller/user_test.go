package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUserController(t *testing.T) {
	env := setupEnv(t)
	r := gin.Default()

	r.POST("/register", env.userCtrl.Register)
	r.POST("/login", env.userCtrl.Login)

	t.Run("Register Success", func(t *testing.T) {
		reqBody := request.CreateUserRequest{
			UserBase: model.UserBase{
				Username: "testuser",
				Email:    "test@example.com",
			},
			Password: "password123",
		}
		body, _ := json.Marshal(reqBody)
		w := performRequest(r, "POST", "/register", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var user model.User
		json.Unmarshal(w.Body.Bytes(), &user)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("Login Success", func(t *testing.T) {
		// Gin's PostForm requires a specific way to test
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString("username=testuser&password=password123"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var token model.Token
		json.Unmarshal(w.Body.Bytes(), &token)
		assert.NotEmpty(t, token.AccessToken)
	})

	t.Run("GetMe", func(t *testing.T) {
		r.GET("/me", func(c *gin.Context) {
			c.Set("username", "testuser")
			env.userCtrl.GetMe(c)
		})

		w := performRequest(r, "GET", "/me", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var user model.User
		json.Unmarshal(w.Body.Bytes(), &user)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("RefreshUploadToken", func(t *testing.T) {
		r.POST("/refresh", func(c *gin.Context) {
			c.Set("username", "testuser")
			env.userCtrl.RefreshUploadToken(c)
		})

		w := performRequest(r, "POST", "/refresh", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "upload_token")
	})
}
