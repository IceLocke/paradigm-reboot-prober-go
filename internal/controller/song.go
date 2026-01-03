package controller

import (
	"net/http"
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

// GetAllSongLevels godoc
// @Summary Get all song levels
// @Description Retrieve a list of all song levels with their details
// @Tags song
// @Produce json
// @Success 200 {array} model.SongLevelInfo
// @Failure 500 {object} model.Response
// @Router /songs [get]
func (ctrl *SongController) GetAllSongLevels(c *gin.Context) {
	levels, err := ctrl.songService.GetAllSongLevels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, levels)
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

	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		// If not an integer, try searching by WikiID
		song, err := ctrl.songService.GetSingleSongByWikiID(songIDStr)
		if err != nil {
			c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, song)
		return
	}

	song, err := ctrl.songService.GetSingleSong(songID, src)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, song)
}

// CreateSong godoc
// @Summary Create a new song
// @Description Create a new song with its levels (Admin only)
// @Tags song
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param song body request.CreateSongRequest true "Song creation info"
// @Success 200 {array} model.SongLevelInfo
// @Failure 400 {object} model.Response
// @Router /songs [post]
func (ctrl *SongController) CreateSong(c *gin.Context) {
	var req request.CreateSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	levels, err := ctrl.songService.CreateSong(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, levels)
}

// UpdateSong godoc
// @Summary Update an existing song
// @Description Update song details and its levels (Admin only)
// @Tags song
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param song body request.UpdateSongRequest true "Song update info"
// @Success 200 {array} model.SongLevelInfo
// @Failure 400 {object} model.Response
// @Router /songs [put]
func (ctrl *SongController) UpdateSong(c *gin.Context) {
	var req request.UpdateSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}

	levels, err := ctrl.songService.UpdateSong(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, levels)
}
