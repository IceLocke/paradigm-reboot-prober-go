package service

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
	"strconv"
	"strings"
)

type SongService struct {
	songRepo *repository.SongRepository
}

func NewSongService(songRepo *repository.SongRepository) *SongService {
	return &SongService{songRepo: songRepo}
}

func (s *SongService) GetAllCharts() ([]model.ChartInfo, error) {
	songs, err := s.songRepo.GetAllSongs()
	if err != nil {
		return nil, err
	}

	var charts []model.ChartInfo
	for _, song := range songs {
		for _, chart := range song.Charts {
			charts = append(charts, model.ChartInfo{
				SongBase:     song.SongBase,
				SongID:       song.ID,
				ID:           chart.ID,
				Difficulty:   chart.Difficulty,
				Level:        chart.Level,
				FittingLevel: chart.FittingLevel,
				LevelDesign:  chart.LevelDesign,
				Notes:        chart.Notes,
			})
		}
	}
	return charts, nil
}

// ResolveSongID parses a song_addr (numeric ID or wiki_id) and returns the song_id.
// Returns an error if the song doesn't exist.
func (s *SongService) ResolveSongID(songAddr string) (int, error) {
	if id, err := strconv.Atoi(songAddr); err == nil {
		song, err := s.songRepo.GetSongByID(id)
		if err != nil {
			return 0, err
		}
		if song == nil {
			return 0, errors.New("song not found")
		}
		return song.ID, nil
	}

	song, err := s.songRepo.GetSongByWikiID(songAddr)
	if err != nil {
		return 0, err
	}
	if song == nil {
		return 0, errors.New("song not found")
	}
	return song.ID, nil
}

// ResolveChartID parses a chart_addr (numeric ID or "wiki_id:difficulty") and returns the chart_id.
// Returns an error if the chart doesn't exist or the difficulty is invalid.
func (s *SongService) ResolveChartID(chartAddr string) (int, error) {
	if id, err := strconv.Atoi(chartAddr); err == nil {
		chart, err := s.songRepo.GetChartByID(id)
		if err != nil {
			return 0, err
		}
		if chart == nil {
			return 0, errors.New("chart not found")
		}
		return chart.ID, nil
	}

	// Split on the last ':' to handle wiki_id:difficulty format
	lastColon := strings.LastIndex(chartAddr, ":")
	if lastColon < 0 {
		return 0, errors.New("invalid chart address format, expected wiki_id:difficulty")
	}
	wikiID := chartAddr[:lastColon]
	diffStr := chartAddr[lastColon+1:]

	if wikiID == "" {
		return 0, errors.New("invalid chart address: empty wiki_id")
	}
	if !model.ValidDifficulty(diffStr) {
		return 0, errors.New("invalid difficulty: " + diffStr)
	}

	chart, err := s.songRepo.GetChartByWikiIDAndDifficulty(wikiID, model.Difficulty(diffStr))
	if err != nil {
		return 0, err
	}
	if chart == nil {
		return 0, errors.New("chart not found")
	}
	return chart.ID, nil
}

func (s *SongService) GetSingleSong(songID int, src string) (*model.Song, error) {
	var song *model.Song
	var err error

	switch src {
	case "prp":
		song, err = s.songRepo.GetSongByID(songID)
	case "wiki":
		return nil, errors.New("wiki source not implemented yet")
	default:
		return nil, errors.New("unsupported source type")
	}

	if err != nil {
		return nil, err
	}
	if song == nil {
		return nil, errors.New("song doesn't exist")
	}

	return song, nil
}

func (s *SongService) GetSingleSongByWikiID(wikiID string) (*model.Song, error) {
	var song *model.Song
	var err error

	song, err = s.songRepo.GetSongByWikiID(wikiID)

	if err != nil {
		return nil, err
	}
	if song == nil {
		return nil, errors.New("song doesn't exist")
	}

	return song, nil
}

func (s *SongService) CreateSong(req *request.CreateSongRequest) ([]model.ChartInfo, error) {
	// Map request to model.Song
	song := &model.Song{
		SongBase: req.SongBase,
	}

	// Map charts
	for _, chartInput := range req.Charts {
		chart := model.Chart{
			Difficulty:  chartInput.Difficulty,
			Level:       chartInput.Level,
			LevelDesign: &chartInput.LevelDesign,
			Notes:       chartInput.Notes,
		}
		song.Charts = append(song.Charts, chart)
	}

	createdSong, err := s.songRepo.CreateSong(song)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	var charts []model.ChartInfo
	for _, chart := range createdSong.Charts {
		info := model.ChartInfo{
			SongBase:     createdSong.SongBase,
			SongID:       createdSong.ID,
			ID:           chart.ID,
			Difficulty:   chart.Difficulty,
			Level:        chart.Level,
			FittingLevel: chart.FittingLevel,
			LevelDesign:  chart.LevelDesign,
			Notes:        chart.Notes,
		}
		charts = append(charts, info)
	}

	return charts, nil
}

func (s *SongService) UpdateSong(req *request.UpdateSongRequest) ([]model.ChartInfo, error) {
	// Map request to model.Song
	song := &model.Song{
		SongBase: req.SongBase,
	}

	// Map charts
	for _, chartInput := range req.Charts {
		chart := model.Chart{
			Difficulty:  chartInput.Difficulty,
			Level:       chartInput.Level,
			LevelDesign: &chartInput.LevelDesign,
			Notes:       chartInput.Notes,
		}
		song.Charts = append(song.Charts, chart)
	}

	updatedSong, err := s.songRepo.UpdateSong(req.ID, song)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	var charts []model.ChartInfo
	for _, chart := range updatedSong.Charts {
		info := model.ChartInfo{
			SongBase:     updatedSong.SongBase,
			SongID:       updatedSong.ID,
			ID:           chart.ID,
			Difficulty:   chart.Difficulty,
			Level:        chart.Level,
			FittingLevel: chart.FittingLevel,
			LevelDesign:  chart.LevelDesign,
			Notes:        chart.Notes,
		}
		charts = append(charts, info)
	}

	return charts, nil
}
