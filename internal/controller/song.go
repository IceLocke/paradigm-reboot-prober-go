package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"paradigm-reboot-prober-go/internal/logging"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SongController struct {
	songService *service.SongService
}

func NewSongController(songService *service.SongService) *SongController {
	return &SongController{songService: songService}
}

// GetAllCharts godoc
// @Summary Get all charts
// @Description Retrieve a list of all charts with their details
// @Tags song
// @Produce json
// @Success 200 {array} model.ChartInfo
// @Failure 500 {object} model.Response
// @Router /songs [get]
func (ctrl *SongController) GetAllCharts(c *gin.Context) {
	charts, err := ctrl.songService.GetAllCharts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, charts)
}

// GetSingleSongInfo godoc
// @Summary Get single song info
// @Description Retrieve detailed information about a single song by ID
// @Tags song
// @Produce json
// @Param song_id path string true "Song ID"
// @Param src query string false "Source (prp or wiki)" default(prp)
// @Success 200 {object} model.Song
// @Failure 400 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /songs/{song_id} [get]
func (ctrl *SongController) GetSingleSongInfo(c *gin.Context) {
	songIDStr := c.Param("song_id")
	src := c.DefaultQuery("src", "prp")
	ctx := logging.AppendCtx(c.Request.Context(), slog.String("song_addr", songIDStr))

	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		// If not an integer, try searching by WikiID
		song, err := ctrl.songService.GetSingleSongByWikiID(ctx, songIDStr)
		if err != nil {
			c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, song)
		return
	}

	song, err := ctrl.songService.GetSingleSong(ctx, songID, src)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, song)
}

// CreateSong godoc
// @Summary Create a new song
// @Description Create a new song with its charts (Admin only)
// @Tags song
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param song body request.CreateSongRequest true "Song creation info"
// @Success 201 {array} model.ChartInfo
func (ctrl *SongController) CreateSong(c *gin.Context) {
	var req request.CreateSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	ctx := logging.AppendCtx(c.Request.Context(), slog.String("song_title", req.Title))
	charts, err := ctrl.songService.CreateSong(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, charts)
}

// UpdateSong godoc
// @Summary Update an existing song
// @Description Update song details and its charts (Admin only)
// @Tags song
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param song body request.UpdateSongRequest true "Song update info"
// @Success 200 {array} model.ChartInfo
// @Failure 400 {object} model.Response
// @Router /songs [put]
func (ctrl *SongController) UpdateSong(c *gin.Context) {
	var req request.UpdateSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	ctx := logging.AppendCtx(c.Request.Context(),
		slog.Int("song_id", req.ID),
		slog.String("song_title", req.Title),
	)
	charts, err := ctrl.songService.UpdateSong(ctx, &req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, model.Response{Error: "song not found"})
		} else {
			c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, charts)
}
