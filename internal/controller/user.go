package controller

import (
	"net/http"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/service"

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
// @Success 200 {object} model.User
// @Failure 400 {object} model.Response
// @Router /user/register [post]
func (ctrl *UserController) Register(c *gin.Context) {
	var req request.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	user, err := ctrl.userService.CreateUser(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
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
	password := c.PostForm("password")

	token, err := ctrl.userService.Login(username, password)
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
	token, err := ctrl.userService.RefreshUploadToken(username)
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

	user, err := ctrl.userService.UpdateUser(username, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}
