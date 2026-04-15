package controller

import (
	"bytes"
	"context"
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
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
		}
		body, _ := json.Marshal(reqBody)
		w := performRequest(r, "POST", "/register", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var user model.User
		err := json.Unmarshal(w.Body.Bytes(), &user)
		assert.NoError(t, err)
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
		err := json.Unmarshal(w.Body.Bytes(), &token)
		assert.NoError(t, err)
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
		err := json.Unmarshal(w.Body.Bytes(), &user)
		assert.NoError(t, err)
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

func TestUserController_UpdateMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := setupEnv(t)

	// Seed a user via service
	ctx := context.Background()
	_, err := env.userService.CreateUser(ctx, &request.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	assert.NoError(t, err)

	r := gin.Default()
	r.PUT("/user/me", func(c *gin.Context) {
		c.Set("username", "testuser")
		env.userCtrl.UpdateMe(c)
	})

	nickname := "NewNickname"
	reqBody := request.UpdateUserRequest{
		Nickname: &nickname,
	}
	body, _ := json.Marshal(reqBody)
	w := performRequest(r, "PUT", "/user/me", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var user model.User
	err = json.Unmarshal(w.Body.Bytes(), &user)
	assert.NoError(t, err)
	assert.Equal(t, "NewNickname", user.Nickname)
}

func TestUserController_ChangePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := setupEnv(t)

	// Seed a user via service
	ctx := context.Background()
	_, err := env.userService.CreateUser(ctx, &request.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	assert.NoError(t, err)

	r := gin.Default()
	r.PUT("/user/me/password", func(c *gin.Context) {
		c.Set("username", "testuser")
		env.userCtrl.ChangePassword(c)
	})

	t.Run("Success", func(t *testing.T) {
		reqBody := request.ChangePasswordRequest{
			OldPassword: "password123",
			NewPassword: "newpassword456",
		}
		body, _ := json.Marshal(reqBody)
		w := performRequest(r, "PUT", "/user/me/password", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp model.Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "password changed successfully", resp.Message)
	})

	t.Run("Wrong old password", func(t *testing.T) {
		reqBody := request.ChangePasswordRequest{
			OldPassword: "wrongpassword",
			NewPassword: "newpassword789",
		}
		body, _ := json.Marshal(reqBody)
		w := performRequest(r, "PUT", "/user/me/password", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestUserController_ResetPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := setupEnv(t)

	// Seed a user via service
	ctx := context.Background()
	_, err := env.userService.CreateUser(ctx, &request.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	assert.NoError(t, err)

	r := gin.Default()
	r.POST("/user/reset-password", func(c *gin.Context) {
		c.Set("username", "adminuser")
		env.userCtrl.ResetPassword(c)
	})

	reqBody := request.ResetPasswordRequest{
		Username:    "testuser",
		NewPassword: "resetpassword123",
	}
	body, _ := json.Marshal(reqBody)
	w := performRequest(r, "POST", "/user/reset-password", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp model.Response
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "password reset successfully", resp.Message)
}
