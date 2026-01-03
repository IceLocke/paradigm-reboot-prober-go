package controller

import (
	"net/http"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/service"
	"paradigm-reboot-prober-go/internal/util"
	"path/filepath"
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

// GetPlayRecords godoc
// @Summary Get play records
// @Description Retrieve play records for a user based on scope (b50, best, all)
// @Tags record
// @Produce json
// @Param username path string true "Username"
// @Param scope query string false "Scope (b50, best, all)" default(b50)
// @Param underflow query int false "Underflow for b50" default(0)
// @Param page_size query int false "Page size" default(50)
// @Param page_index query int false "Page index" default(1)
// @Param sort_by query string false "Sort by (rating, score, record_time, etc.)" default(rating)
// @Param order query string false "Order (desc or asce)" default(desc)
// @Success 200 {object} model.PlayRecordResponse
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /records/{username} [get]
func (ctrl *RecordController) GetPlayRecords(c *gin.Context) {
	username := c.Param("username")
	scope := c.DefaultQuery("scope", "b50")
	underflow, _ := strconv.Atoi(c.DefaultQuery("underflow", "0"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	pageIndex, _ := strconv.Atoi(c.DefaultQuery("page_index", "1"))
	sortBy := c.DefaultQuery("sort_by", "rating")
	order := c.DefaultQuery("order", "desc")

	// Get current user from context (if authenticated)
	var currentUser *model.User
	currentUsername, exists := c.Get("username")
	if exists {
		currentUser, _ = ctrl.userService.GetUser(currentUsername.(string))
	}

	// Check authority
	if err := ctrl.userService.CheckProbeAuthority(username, currentUser); err != nil {
		c.JSON(http.StatusForbidden, model.Response{Error: err.Error()})
		return
	}

	var records interface{}
	var total int64
	var err error

	switch scope {
	case "b50":
		records, err = ctrl.recordService.GetBest50Records(username, underflow)
	case "best":
		records, err = ctrl.recordService.GetBestRecords(username, pageSize, pageIndex-1, sortBy, order)
		total, _ = ctrl.recordService.CountBestRecords(username)
	case "all":
		records, err = ctrl.recordService.GetAllRecords(username, pageSize, pageIndex-1, sortBy, order)
		total, _ = ctrl.recordService.CountAllRecords(username)
	default:
		c.JSON(http.StatusBadRequest, model.Response{Error: "invalid scope parameter"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": username,
		"records":  records,
		"total":    total,
	})
}

// UploadRecords godoc
// @Summary Upload play records
// @Description Batch upload play records for a user
// @Tags record
// @Accept json
// @Produce json
// @Param username path string true "Username"
// @Param record body request.BatchCreatePlayRecordRequest true "Play records upload info"
// @Success 201 {array} model.PlayRecord
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /records/{username} [post]
func (ctrl *RecordController) UploadRecords(c *gin.Context) {
	username := c.Param("username")
	var req request.BatchCreatePlayRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	// Ambiguous data check
	if (len(req.PlayRecords) > 0) == (req.CSVFilename != "") {
		c.JSON(http.StatusBadRequest, model.Response{Error: "ambiguous data: provide either play_records or csv_filename"})
		return
	}

	currentUser := c.GetString("username")
	authorized := false

	if currentUser == username {
		authorized = true
	} else {
		user, err := ctrl.userService.GetUser(username)
		if err == nil && user != nil && req.UploadToken != "" && req.UploadToken == user.UploadToken {
			authorized = true
		}
	}

	if !authorized {
		c.JSON(http.StatusUnauthorized, model.Response{Error: "unauthorized"})
		return
	}

	var playRecords []model.PlayRecordBase
	isReplace := req.IsReplace

	if req.CSVFilename != "" {
		csvPath := filepath.Join(config.GlobalConfig.Upload.CSVPath, req.CSVFilename)
		var err error
		playRecords, err = util.GetRecordsFromCSV(csvPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.Response{Error: "failed to parse csv: " + err.Error()})
			return
		}
		isReplace = true // CSV upload usually implies replacement in original code
	} else {
		playRecords = req.PlayRecords
	}

	records, err := ctrl.recordService.CreateRecords(username, playRecords, isReplace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, records)
}

// ExportCSV godoc
// @Summary Export records to CSV
// @Description Export all best records for a user to CSV
// @Tags record
// @Produce text/csv
// @Param username path string true "Username"
// @Success 200 {string} string "CSV content"
// @Failure 401 {object} model.Response
// @Router /records/{username}/export/csv [get]
func (ctrl *RecordController) ExportCSV(c *gin.Context) {
	username := c.Param("username")

	// Get current user from context (if authenticated)
	var currentUser *model.User
	currentUsername, exists := c.Get("username")
	if exists {
		currentUser, _ = ctrl.userService.GetUser(currentUsername.(string))
	}

	// Check authority
	if err := ctrl.userService.CheckProbeAuthority(username, currentUser); err != nil {
		c.JSON(http.StatusForbidden, model.Response{Error: err.Error()})
		return
	}

	records, err := ctrl.recordService.GetAllLevelsWithBestScores(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	csvData, err := util.GenerateCSV(records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: "failed to generate CSV"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=records.csv")
	c.Data(http.StatusOK, "text/csv", []byte(csvData))
}

// GetB50Img godoc
// @Summary Get B50 image
// @Description Generate and return B50 image for a user
// @Tags record
// @Produce image/png
// @Param username path string true "Username"
// @Success 200 {file} binary
// @Failure 403 {object} model.Response
// @Router /records/{username}/export/b50 [get]
func (ctrl *RecordController) GetB50Img(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, model.Response{Error: "not implemented yet"})
}

// GetB50Trends godoc
// @Summary Get B50 trends
// @Description Get B50 rating trends for a user
// @Tags record
// @Produce json
// @Param username path string true "Username"
// @Success 200 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /records/{username}/trends [get]
func (ctrl *RecordController) GetB50Trends(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, model.Response{Error: "not implemented yet"})
}
