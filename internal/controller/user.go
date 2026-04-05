package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"paradigm-reboot-prober-go/internal/logging"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags user
// @Accept json
// @Produce json
// @Param user body request.CreateUserRequest true "User registration info"
// @Success 201 {object} model.UserPublic
// @Failure 400 {object} model.Response
// @Router /user/register [post]
func (ctrl *UserController) Register(c *gin.Context) {
	var req request.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	req.Username = strings.ToLower(req.Username)

	ctx := logging.AppendCtx(c.Request.Context(), slog.String("register_user", req.Username))
	user, err := ctrl.userService.CreateUser(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user.ToPublic())
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags user
// @Accept x-www-form-urlencoded
// @Produce json
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Success 200 {object} model.Token
// @Failure 401 {object} model.Response
// @Router /user/login [post]
func (ctrl *UserController) Login(c *gin.Context) {
	username := c.PostForm("username")
	username = strings.ToLower(username)
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, model.Response{Error: "username and password are required"})
		return
	}

	ctx := logging.AppendCtx(c.Request.Context(), slog.String("login_user", username))
	token, err := ctrl.userService.Login(ctx, username, password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Token{
		AccessToken: token,
		TokenType:   "Bearer",
	})
}

// GetMe godoc
// @Summary Get current user info
// @Description Get the profile of the currently authenticated user
// @Tags user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.User
// @Failure 401 {object} model.Response
// @Router /user/me [get]
func (ctrl *UserController) GetMe(c *gin.Context) {
	username := c.GetString("username")
	user, err := ctrl.userService.GetUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, model.Response{Error: "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// RefreshUploadToken godoc
// @Summary Refresh upload token
// @Description Generate a new upload token for the current user
// @Tags user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.UploadToken
// @Failure 401 {object} model.Response
// @Router /user/me/upload-token [post]
func (ctrl *UserController) RefreshUploadToken(c *gin.Context) {
	username := c.GetString("username")
	token, err := ctrl.userService.RefreshUploadToken(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.UploadToken{UploadToken: token})
}

// UpdateMe godoc
// @Summary Update current user info
// @Description Update the profile of the currently authenticated user
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body request.UpdateUserRequest true "User update info"
// @Success 200 {object} model.User
// @Failure 400 {object} model.Response
// @Router /user/me [put]
func (ctrl *UserController) UpdateMe(c *gin.Context) {
	username := c.GetString("username")
	var req request.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	user, err := ctrl.userService.UpdateUser(c.Request.Context(), username, &req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}

// ChangePassword godoc
// @Summary Change password
// @Description Change the current user's password
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.ChangePasswordRequest true "Password change info"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /user/me/password [put]
func (ctrl *UserController) ChangePassword(c *gin.Context) {
	username := c.GetString("username")
	var req request.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}
	if err := ctrl.userService.ChangePassword(c.Request.Context(), username, &req); err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, model.Response{Error: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, model.Response{Message: "password changed successfully"})
}

// ResetPassword godoc
// @Summary Reset user password (Admin only)
// @Description Reset a user's password by admin
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.ResetPasswordRequest true "Password reset info"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /user/reset-password [post]
func (ctrl *UserController) ResetPassword(c *gin.Context) {
	var req request.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}
	if err := ctrl.userService.ResetPassword(c.Request.Context(), &req); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, model.Response{Message: "password reset successfully"})
}
