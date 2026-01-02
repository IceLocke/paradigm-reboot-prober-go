package controller

import (
	"net/http"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RecordController struct {
	recordService *service.RecordService
	userService   *service.UserService
}

func NewRecordController(recordService *service.RecordService, userService *service.UserService) *RecordController {
	return &RecordController{
		recordService: recordService,
		userService:   userService,
	}
}

// UploadRecords godoc
// @Summary Upload play records
// @Description Batch upload play records for a user
// @Tags record
// @Accept json
// @Produce json
// @Param record body request.BatchCreatePlayRecordRequest true "Play records upload info"
// @Success 200 {array} model.PlayRecord
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /records [post]
func (ctrl *RecordController) UploadRecords(c *gin.Context) {
	var req request.BatchCreatePlayRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In this implementation, we might use either JWT or UploadToken.
	// If username is in context (from JWT), use it.
	// Otherwise, we might need to find user by UploadToken.
	username := c.GetString("username")
	if username == "" {
		// Fallback to UploadToken if provided
		if req.UploadToken != "" {
			user, err := ctrl.userService.GetUserByUploadToken(req.UploadToken)
			if err != nil || user == nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid upload token"})
				return
			}
			username = user.Username
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
	}

	records, err := ctrl.recordService.CreateRecords(username, req.PlayRecords, req.IsReplace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, records)
}

// GetB50 godoc
// @Summary Get Best 50 records
// @Description Retrieve the best 50 records for a user
// @Tags record
// @Produce json
// @Param username query string true "Username"
// @Success 200 {array} model.PlayRecord
// @Failure 400 {object} gin.H
// @Router /records/b50 [get]
func (ctrl *RecordController) GetB50(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}

	records, err := ctrl.recordService.GetBest50Records(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, records)
}

// GetBestRecords godoc
// @Summary Get best records
// @Description Retrieve best records for each song level for a user
// @Tags record
// @Produce json
// @Param username query string true "Username"
// @Param page query int false "Page index" default(0)
// @Param size query int false "Page size" default(10)
// @Param sort query string false "Sort by" default(score)
// @Param order query string false "Order (asc or desc)" default(desc)
// @Success 200 {array} model.PlayRecord
// @Router /records/best [get]
func (ctrl *RecordController) GetBestRecords(c *gin.Context) {
	username := c.Query("username")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	sort := c.DefaultQuery("sort", "score")
	order := c.DefaultQuery("order", "desc")

	records, err := ctrl.recordService.GetBestRecords(username, size, page, sort, order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, records)
}

// GetAllRecords godoc
// @Summary Get all records
// @Description Retrieve all play records for a user
// @Tags record
// @Produce json
// @Param username query string true "Username"
// @Param page query int false "Page index" default(0)
// @Param size query int false "Page size" default(10)
// @Param sort query string false "Sort by" default(record_time)
// @Param order query string false "Order (asc or desc)" default(desc)
// @Success 200 {array} model.PlayRecord
// @Router /records/all [get]
func (ctrl *RecordController) GetAllRecords(c *gin.Context) {
	username := c.Query("username")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	sort := c.DefaultQuery("sort", "record_time")
	order := c.DefaultQuery("order", "desc")

	records, err := ctrl.recordService.GetAllRecords(username, size, page, sort, order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, records)
}
