package service

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// compareVersion compares two dot-separated version strings numerically.
// Returns -1, 0, or +1 (like cmp.Compare). Non-numeric segments fall back to
// lexicographic comparison. Missing trailing segments are treated as 0.
func compareVersion(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(pa), len(pb)) {
		sa, sb := "0", "0"
		if i < len(pa) {
			sa = pa[i]
		}
		if i < len(pb) {
			sb = pb[i]
		}
		na, errA := strconv.Atoi(sa)
		nb, errB := strconv.Atoi(sb)
		if errA == nil && errB == nil {
			if c := cmp.Compare(na, nb); c != 0 {
				return c
			}
		} else if c := cmp.Compare(sa, sb); c != 0 {
			return c
		}
	}
	return 0
}

type SongService struct {
	songRepo *repository.SongRepository
}

func NewSongService(songRepo *repository.SongRepository) *SongService {
	return &SongService{songRepo: songRepo}
}

func buildChartInfos(songs []model.Song) []model.ChartInfo {
	var charts []model.ChartInfo
	for _, song := range songs {
		for _, chart := range song.Charts {
			charts = append(charts, model.ChartInfo{
				SongBase:     song.WithOverride(chart.SongBaseOverride),
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
	// Default sort: Version DESC (newest first), SongID ASC (as order in album), then Difficulty DESC (hardest first)
	slices.SortFunc(charts, func(a, b model.ChartInfo) int {
		if c := compareVersion(a.Version, b.Version); c != 0 {
			return -c // descending
		}
		if c := cmp.Compare(a.SongID, b.SongID); c != 0 {
			return c // ascending
		}
		return cmp.Compare(b.Difficulty.Order(), a.Difficulty.Order())
	})
	return charts
}

func computeSongsETag(songs []model.Song) string {
	h := sha256.New()
	_ = binary.Write(h, binary.BigEndian, uint64(len(songs)))
	for _, s := range songs {
		_ = binary.Write(h, binary.BigEndian, uint64(s.ID))
		_ = binary.Write(h, binary.BigEndian, s.UpdatedAt.UnixNano())
		_ = binary.Write(h, binary.BigEndian, uint64(len(s.Charts)))
		for _, c := range s.Charts {
			_ = binary.Write(h, binary.BigEndian, uint64(c.ID))
			_ = binary.Write(h, binary.BigEndian, c.UpdatedAt.UnixNano())
		}
	}
	return fmt.Sprintf(`"%x"`, h.Sum(nil)[:8])
}

func (s *SongService) GetAllCharts(ctx context.Context) ([]model.ChartInfo, error) {
	songs, err := s.songRepo.GetAllSongs()
	if err != nil {
		return nil, err
	}
	return buildChartInfos(songs), nil
}

func (s *SongService) GetAllChartsWithETag(ctx context.Context) ([]model.ChartInfo, string, error) {
	songs, err := s.songRepo.GetAllSongs()
	if err != nil {
		return nil, "", err
	}
	charts := buildChartInfos(songs)
	etag := computeSongsETag(songs)
	return charts, etag, nil
}

// ResolveSongID parses a song_addr (numeric ID or wiki_id) and returns the song_id.
// Returns an error if the song doesn't exist.
func (s *SongService) ResolveSongID(ctx context.Context, songAddr string) (int, error) {
	if id, err := strconv.Atoi(songAddr); err == nil {
		song, err := s.songRepo.GetSongByID(id)
		if err != nil {
			return 0, err
		}
		if song == nil {
			return 0, fmt.Errorf("song %w", ErrNotFound)
		}
		return song.ID, nil
	}

	song, err := s.songRepo.GetSongByWikiID(songAddr)
	if err != nil {
		return 0, err
	}
	if song == nil {
		return 0, fmt.Errorf("song %w", ErrNotFound)
	}
	return song.ID, nil
}

// ResolveChartID parses a chart_addr (numeric ID or "wiki_id:difficulty") and returns the chart_id.
// Returns an error if the chart doesn't exist or the difficulty is invalid.
func (s *SongService) ResolveChartID(ctx context.Context, chartAddr string) (int, error) {
	if id, err := strconv.Atoi(chartAddr); err == nil {
		chart, err := s.songRepo.GetChartByID(id)
		if err != nil {
			return 0, err
		}
		if chart == nil {
			return 0, fmt.Errorf("chart %w", ErrNotFound)
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
		return 0, fmt.Errorf("chart %w", ErrNotFound)
	}
	return chart.ID, nil
}

func (s *SongService) GetSingleSong(ctx context.Context, songID int, src string) (*model.Song, error) {
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
		return nil, fmt.Errorf("song doesn't exist: %w", ErrNotFound)
	}

	return song, nil
}

func (s *SongService) GetSingleSongByWikiID(ctx context.Context, wikiID string) (*model.Song, error) {
	var song *model.Song
	var err error

	song, err = s.songRepo.GetSongByWikiID(wikiID)

	if err != nil {
		return nil, err
	}
	if song == nil {
		return nil, fmt.Errorf("song doesn't exist: %w", ErrNotFound)
	}

	return song, nil
}

func (s *SongService) CreateSong(ctx context.Context, req *request.CreateSongRequest) ([]model.ChartInfo, error) {
	// Check for duplicate difficulties
	seenDifficulties := make(map[model.Difficulty]bool)
	for _, chartInput := range req.Charts {
		if seenDifficulties[chartInput.Difficulty] {
			return nil, fmt.Errorf("duplicate chart difficulty: %s", chartInput.Difficulty)
		}
		seenDifficulties[chartInput.Difficulty] = true
	}

	// Map request to model.Song
	song := &model.Song{
		SongBase: req.SongBase,
	}

	// Map charts
	for _, chartInput := range req.Charts {
		chart := model.Chart{
			Difficulty:       chartInput.Difficulty,
			Level:            chartInput.Level,
			LevelDesign:      &chartInput.LevelDesign,
			Notes:            chartInput.Notes,
			SongBaseOverride: chartInput.SongBaseOverride,
		}
		song.Charts = append(song.Charts, chart)
	}

	createdSong, err := s.songRepo.CreateSong(song)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create song", "error", err, "title", req.Title)
		return nil, err
	}
	slog.InfoContext(ctx, "song created", "song_id", createdSong.ID, "title", createdSong.Title)

	// Convert to response format
	var charts []model.ChartInfo
	for _, chart := range createdSong.Charts {
		info := model.ChartInfo{
			SongBase:     createdSong.WithOverride(chart.SongBaseOverride),
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

func (s *SongService) UpdateSong(ctx context.Context, req *request.UpdateSongRequest) ([]model.ChartInfo, error) {
	// Check for duplicate difficulties
	seenDifficulties := make(map[model.Difficulty]bool)
	for _, chartInput := range req.Charts {
		if seenDifficulties[chartInput.Difficulty] {
			return nil, fmt.Errorf("duplicate chart difficulty: %s", chartInput.Difficulty)
		}
		seenDifficulties[chartInput.Difficulty] = true
	}

	// Map request to model.Song
	song := &model.Song{
		SongBase: req.SongBase,
	}

	// Map charts
	for _, chartInput := range req.Charts {
		chart := model.Chart{
			Difficulty:       chartInput.Difficulty,
			Level:            chartInput.Level,
			LevelDesign:      &chartInput.LevelDesign,
			Notes:            chartInput.Notes,
			SongBaseOverride: chartInput.SongBaseOverride,
		}
		song.Charts = append(song.Charts, chart)
	}

	updatedSong, err := s.songRepo.UpdateSong(req.ID, song)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("song %w", ErrNotFound)
		}
		slog.ErrorContext(ctx, "failed to update song", "error", err, "song_id", req.ID)
		return nil, err
	}
	slog.InfoContext(ctx, "song updated", "song_id", updatedSong.ID, "title", updatedSong.Title)

	// Convert to response format
	var charts []model.ChartInfo
	for _, chart := range updatedSong.Charts {
		info := model.ChartInfo{
			SongBase:     updatedSong.WithOverride(chart.SongBaseOverride),
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
