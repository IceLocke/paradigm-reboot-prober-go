package controller

import (
	"crypto/subtle"
	"net/http"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type RecordController struct {
	recordService *service.RecordService
	userService   *service.UserService
	songService   *service.SongService
}

func NewRecordController(recordService *service.RecordService, userService *service.UserService, songService *service.SongService) *RecordController {
	return &RecordController{
		recordService: recordService,
		userService:   userService,
		songService:   songService,
	}
}

// paginationParams holds parsed pagination and sorting parameters
type paginationParams struct {
	pageSize  int
	pageIndex int
	sortBy    string
	order     string
}

// parsePaginationParams extracts and validates pagination parameters from the request
func parsePaginationParams(c *gin.Context) paginationParams {
	defaultPageSize := strconv.Itoa(config.GlobalConfig.Pagination.DefaultPageSize)
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", defaultPageSize))
	pageIndex, _ := strconv.Atoi(c.DefaultQuery("page_index", "1"))

	if pageSize <= 0 {
		pageSize = config.GlobalConfig.Pagination.DefaultPageSize
	}
	if pageSize > config.GlobalConfig.Pagination.MaxPageSize {
		pageSize = config.GlobalConfig.Pagination.MaxPageSize
	}
	if pageIndex < 1 {
		pageIndex = 1
	}

	return paginationParams{
		pageSize:  pageSize,
		pageIndex: pageIndex,
		sortBy:    c.DefaultQuery("sort_by", "rating"),
		order:     c.DefaultQuery("order", "desc"),
	}
}

// GetPlayRecords godoc
// @Summary Get play records
// @Description Retrieve play records for a user based on scope (b50, best, all, all-charts)
// @Tags record
// @Produce json
// @Param username path string true "Username"
// @Param scope query string false "Scope (b50, best, all, all-charts)" default(b50)
// @Param underflow query int false "Underflow for b50" default(0)
// @Param page_size query int false "Page size" default(50)
// @Param page_index query int false "Page index" default(1)
// @Param sort_by query string false "Sort by (rating, score, record_time, etc.)" default(rating)
// @Param order query string false "Order (desc or asc)" default(desc)
// @Success 200 {object} model.PlayRecordResponse
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /records/{username} [get]
func (ctrl *RecordController) GetPlayRecords(c *gin.Context) {
	username := c.Param("username")
	username = strings.ToLower(username)
	scope := c.DefaultQuery("scope", "b50")
	underflow, _ := strconv.Atoi(c.DefaultQuery("underflow", "0"))
	p := parsePaginationParams(c)

	// Validate underflow
	if underflow < 0 {
		underflow = 0
	}
	if underflow > config.GlobalConfig.Game.B35Limit {
		underflow = config.GlobalConfig.Game.B35Limit
	}

	// Get current user from context (if authenticated)
	var currentUser *model.User
	currentUsername, exists := c.Get("username")
	if exists {
		currentUser, _ = ctrl.userService.GetUser(currentUsername.(string))
	}

	// Check authority
	if err := ctrl.userService.CheckProbeAuthority(username, currentUser); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
		} else {
			c.JSON(http.StatusForbidden, model.Response{Error: err.Error()})
		}
		return
	}

	// Fetch target user for nickname
	targetUser, err := ctrl.userService.GetUser(username)
	if targetUser == nil {
		c.JSON(http.StatusNotFound, model.Response{Error: "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	switch scope {
	case "b50":
		records, err := ctrl.recordService.GetBest50Records(username, underflow)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		recordInfos := make([]model.PlayRecordInfo, 0, len(records))
		for _, r := range records {
			recordInfos = append(recordInfos, model.ToPlayRecordInfo(r))
		}
		c.JSON(http.StatusOK, model.PlayRecordResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Total:    len(recordInfos),
			Records:  recordInfos,
		})

	case "best":
		records, err := ctrl.recordService.GetBestRecords(username, p.pageSize, p.pageIndex-1, p.sortBy, p.order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		total, err := ctrl.recordService.CountBestRecords(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		recordInfos := make([]model.PlayRecordInfo, 0, len(records))
		for i := range records {
			recordInfos = append(recordInfos, model.ToPlayRecordInfo(&records[i]))
		}
		c.JSON(http.StatusOK, model.PlayRecordResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Total:    int(total),
			Records:  recordInfos,
		})

	case "all":
		records, err := ctrl.recordService.GetAllRecords(username, p.pageSize, p.pageIndex-1, p.sortBy, p.order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		total, err := ctrl.recordService.CountAllRecords(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		recordInfos := make([]model.PlayRecordInfo, 0, len(records))
		for i := range records {
			recordInfos = append(recordInfos, model.ToPlayRecordInfo(&records[i]))
		}
		c.JSON(http.StatusOK, model.PlayRecordResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Total:    int(total),
			Records:  recordInfos,
		})

	case "all-charts":
		charts, err := ctrl.recordService.GetAllChartsWithBestScores(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, model.AllChartsResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Charts:   charts,
		})

	default:
		c.JSON(http.StatusBadRequest, model.Response{Error: "invalid scope parameter"})
		return
	}
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
	username = strings.ToLower(username)
	var req request.BatchCreatePlayRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	if len(req.PlayRecords) == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Error: "play_records is required"})
		return
	}

	currentUser := c.GetString("username")
	authorized := false

	if currentUser == username {
		authorized = true
	} else {
		user, err := ctrl.userService.GetUser(username)
		if err == nil && user != nil && req.UploadToken != "" &&
			subtle.ConstantTimeCompare([]byte(req.UploadToken), []byte(user.UploadToken)) == 1 {
			authorized = true
		}
	}

	if !authorized {
		c.JSON(http.StatusUnauthorized, model.Response{Error: "unauthorized"})
		return
	}

	records, err := ctrl.recordService.CreateRecords(username, req.PlayRecords, req.IsReplace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, records)
}

// checkProbeAuthority is a helper that resolves the current user and checks probe authority
func (ctrl *RecordController) checkProbeAuthority(c *gin.Context, username string) bool {
	var currentUser *model.User
	currentUsername, exists := c.Get("username")
	if exists {
		currentUser, _ = ctrl.userService.GetUser(currentUsername.(string))
	}
	if err := ctrl.userService.CheckProbeAuthority(username, currentUser); err != nil {
		c.JSON(http.StatusForbidden, model.Response{Error: err.Error()})
		return false
	}
	return true
}

// GetSongRecords godoc
// @Summary Get play records for a specific song
// @Description Retrieve play records for a user scoped to a specific song. song_addr can be numeric song_id or wiki_id.
// @Tags record
// @Produce json
// @Param username path string true "Username"
// @Param song_addr path string true "Song address (numeric song_id or wiki_id)"
// @Param scope query string false "Scope (best, all)" default(best)
// @Param page_size query int false "Page size (scope=all only)" default(50)
// @Param page_index query int false "Page index (scope=all only)" default(1)
// @Param sort_by query string false "Sort by (rating, score, record_time)" default(rating)
// @Param order query string false "Order (desc or asc)" default(desc)
// @Success 200 {object} model.PlayRecordResponse
// @Failure 400 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /records/{username}/song/{song_addr} [get]
func (ctrl *RecordController) GetSongRecords(c *gin.Context) {
	username := strings.ToLower(c.Param("username"))
	songAddr := c.Param("song_addr")
	scope := c.DefaultQuery("scope", "best")

	songID, err := ctrl.songService.ResolveSongID(songAddr)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
		return
	}

	if !ctrl.checkProbeAuthority(c, username) {
		return
	}

	// Fetch target user for nickname
	targetUser, err := ctrl.userService.GetUser(username)
	if targetUser == nil {
		c.JSON(http.StatusNotFound, model.Response{Error: "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	switch scope {
	case "best":
		records, err := ctrl.recordService.GetBestRecordsBySong(username, songID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		var recordInfos []model.PlayRecordInfo
		for i := range records {
			recordInfos = append(recordInfos, model.ToPlayRecordInfo(&records[i]))
		}
		c.JSON(http.StatusOK, model.PlayRecordResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Total:    len(recordInfos),
			Records:  recordInfos,
		})

	case "all":
		p := parsePaginationParams(c)
		records, err := ctrl.recordService.GetAllRecordsBySong(username, songID, p.pageSize, p.pageIndex-1, p.sortBy, p.order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		total, _ := ctrl.recordService.CountAllRecordsBySong(username, songID)
		var recordInfos []model.PlayRecordInfo
		for i := range records {
			recordInfos = append(recordInfos, model.ToPlayRecordInfo(&records[i]))
		}
		c.JSON(http.StatusOK, model.PlayRecordResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Total:    int(total),
			Records:  recordInfos,
		})

	default:
		c.JSON(http.StatusBadRequest, model.Response{Error: "invalid scope parameter, expected 'best' or 'all'"})
	}
}

// GetChartRecords godoc
// @Summary Get play records for a specific chart
// @Description Retrieve play records for a user scoped to a specific chart. chart_addr can be numeric chart_id or wiki_id:difficulty (e.g. felys:massive).
// @Tags record
// @Produce json
// @Param username path string true "Username"
// @Param chart_addr path string true "Chart address (numeric chart_id or wiki_id:difficulty)"
// @Param scope query string false "Scope (best, all)" default(best)
// @Param page_size query int false "Page size (scope=all only)" default(50)
// @Param page_index query int false "Page index (scope=all only)" default(1)
// @Param sort_by query string false "Sort by (rating, score, record_time)" default(rating)
// @Param order query string false "Order (desc or asc)" default(desc)
// @Success 200 {object} model.PlayRecordResponse
// @Failure 400 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /records/{username}/chart/{chart_addr} [get]
func (ctrl *RecordController) GetChartRecords(c *gin.Context) {
	username := strings.ToLower(c.Param("username"))
	chartAddr := c.Param("chart_addr")
	scope := c.DefaultQuery("scope", "best")

	chartID, err := ctrl.songService.ResolveChartID(chartAddr)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
		return
	}

	if !ctrl.checkProbeAuthority(c, username) {
		return
	}

	// Fetch target user for nickname
	targetUser, err := ctrl.userService.GetUser(username)
	if targetUser == nil {
		c.JSON(http.StatusNotFound, model.Response{Error: "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}

	switch scope {
	case "best":
		record, err := ctrl.recordService.GetBestRecordByChart(username, chartID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		var recordInfos []model.PlayRecordInfo
		if record != nil {
			recordInfos = append(recordInfos, model.ToPlayRecordInfo(record))
		}
		c.JSON(http.StatusOK, model.PlayRecordResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Total:    len(recordInfos),
			Records:  recordInfos,
		})

	case "all":
		p := parsePaginationParams(c)
		records, err := ctrl.recordService.GetAllRecordsByChart(username, chartID, p.pageSize, p.pageIndex-1, p.sortBy, p.order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
			return
		}
		total, _ := ctrl.recordService.CountAllRecordsByChart(username, chartID)
		var recordInfos []model.PlayRecordInfo
		for i := range records {
			recordInfos = append(recordInfos, model.ToPlayRecordInfo(&records[i]))
		}
		c.JSON(http.StatusOK, model.PlayRecordResponse{
			Username: username,
			Nickname: targetUser.Nickname,
			Total:    int(total),
			Records:  recordInfos,
		})

	default:
		c.JSON(http.StatusBadRequest, model.Response{Error: "invalid scope parameter, expected 'best' or 'all'"})
	}
}
