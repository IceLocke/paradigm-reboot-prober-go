package service

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
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
				SongID:       song.SongID,
				ChartID:      chart.ChartID,
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
			SongID:       createdSong.SongID,
			ChartID:      chart.ChartID,
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

	updatedSong, err := s.songRepo.UpdateSong(req.SongID, song)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	var charts []model.ChartInfo
	for _, chart := range updatedSong.Charts {
		info := model.ChartInfo{
			SongBase:     updatedSong.SongBase,
			SongID:       updatedSong.SongID,
			ChartID:      chart.ChartID,
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
