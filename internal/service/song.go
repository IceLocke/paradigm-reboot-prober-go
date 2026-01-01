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

func (s *SongService) GetAllSongLevels() ([]model.SongLevelInfo, error) {
	songs, err := s.songRepo.GetAllSongs()
	if err != nil {
		return nil, err
	}

	var songLevels []model.SongLevelInfo
	for _, song := range songs {
		for _, level := range song.SongLevels {
			songLevels = append(songLevels, model.SongLevelInfo{
				SongBase:    song.SongBase,
				SongID:      song.SongID,
				SongLevelID: level.SongLevelID,
				Difficulty:  level.Difficulty,
				Level:       level.Level,
				Notes:       level.Notes,
				// Add other fields as needed
			})
		}
	}
	return songLevels, nil
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
		return nil, nil
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

func (s *SongService) CreateSong(req *request.CreateSongRequest) ([]model.SongLevelInfo, error) {
	// Map request to model.Song
	song := &model.Song{
		SongBase: req.SongBase,
	}

	// Map levels
	for _, levelInfo := range req.Levels {
		songLevel := model.SongLevel{
			Difficulty:  levelInfo.Difficulty,
			Level:       levelInfo.Level,
			LevelDesign: &levelInfo.LevelDesign,
			Notes:       levelInfo.Notes,
		}
		song.SongLevels = append(song.SongLevels, songLevel)
	}

	createdSong, err := s.songRepo.CreateSong(song)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	var songLevels []model.SongLevelInfo
	for _, level := range createdSong.SongLevels {
		info := model.SongLevelInfo{
			SongBase:    createdSong.SongBase,
			SongID:      createdSong.SongID,
			SongLevelID: level.SongLevelID,
			Difficulty:  level.Difficulty,
			Level:       level.Level,
			Notes:       level.Notes,
		}
		if level.LevelDesign != nil {
			info.LevelDesign = *level.LevelDesign
		}
		songLevels = append(songLevels, info)
	}

	return songLevels, nil
}

func (s *SongService) UpdateSong(req *request.UpdateSongRequest) ([]model.SongLevelInfo, error) {
	// Map request to model.Song
	song := &model.Song{
		SongBase: req.SongBase,
	}

	// Map levels
	for _, levelInfo := range req.Levels {
		songLevel := model.SongLevel{
			Difficulty:  levelInfo.Difficulty,
			Level:       levelInfo.Level,
			LevelDesign: &levelInfo.LevelDesign,
			Notes:       levelInfo.Notes,
		}
		song.SongLevels = append(song.SongLevels, songLevel)
	}

	updatedSong, err := s.songRepo.UpdateSong(req.SongID, song)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	var songLevels []model.SongLevelInfo
	for _, level := range updatedSong.SongLevels {
		info := model.SongLevelInfo{
			SongBase:    updatedSong.SongBase,
			SongID:      updatedSong.SongID,
			SongLevelID: level.SongLevelID,
			Difficulty:  level.Difficulty,
			Level:       level.Level,
			Notes:       level.Notes,
		}
		if level.LevelDesign != nil {
			info.LevelDesign = *level.LevelDesign
		}
		songLevels = append(songLevels, info)
	}

	return songLevels, nil
}
