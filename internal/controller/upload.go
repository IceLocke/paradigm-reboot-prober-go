package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/service"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type UploadController struct {
	userService *service.UserService
}

func NewUploadController(userService *service.UserService) *UploadController {
	return &UploadController{userService: userService}
}

func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// UploadCSV godoc
// @Summary Upload CSV file
// @Description Upload a CSV file containing play records
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV file"
// @Success 200 {object} model.UploadFileResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /upload/csv [post]
func (ctrl *UploadController) UploadCSV(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	if filepath.Ext(file.Filename) != ".csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only CSV files are allowed"})
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(config.GlobalConfig.Upload.CSVPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
		return
	}

	filename := fmt.Sprintf("%s_b50_%s.csv", username, generateRandomHex(6))
	dst := filepath.Join(config.GlobalConfig.Upload.CSVPath, filename)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	c.JSON(http.StatusOK, model.UploadFileResponse{Filename: filename})
}

// UploadImg godoc
// @Summary Upload image file
// @Description Upload an image file (Admin only)
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Image file"
// @Success 200 {object} model.UploadFileResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Router /upload/img [post]
func (ctrl *UploadController) UploadImg(c *gin.Context) {
	username := c.GetString("username")
	user, err := ctrl.userService.GetUser(username)
	if err != nil || user == nil || !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(config.GlobalConfig.Upload.ImgPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
		return
	}

	filename := fmt.Sprintf("%s_%s%s", generateRandomHex(8), file.Filename, filepath.Ext(file.Filename))
	dst := filepath.Join(config.GlobalConfig.Upload.ImgPath, filename)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	c.JSON(http.StatusOK, model.UploadFileResponse{Filename: filename})
}
