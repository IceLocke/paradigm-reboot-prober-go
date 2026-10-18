package repository

import (
	"fmt"
	"math"
	"paradigm-reboot-prober-go/internal/model"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordIDs(records []model.PlayRecord) []int {
	ids := make([]int, len(records))
	for i := range records {
		ids[i] = records[i].ID
	}
	return ids
}

func TestRecordRepository_FittingRanking(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	t.Cleanup(repo.cache.Stop)
	songRepo := NewSongRepository(db)
	t.Cleanup(songRepo.cache.Stop)
	var promotedIDs, promotedChartIDs []int
	var allIDs []int
	for season, count := range []int{42, 22} {
		level := 16.0 + float64(season)
		for i := 0; i < count; i++ {
			chart := model.Chart{Difficulty: model.DifficultyMassive, Level: level, FittingLevel: float64Ptr(level - 0.2)}
			switch i {
			case 0:
				chart.FittingLevel = nil
			case 1:
				chart.FittingLevel = float64Ptr(0)
			case count - 1:
				chart.Level = level - 0.1
				chart.FittingLevel = float64Ptr(level + 0.2)
			}
			song, err := songRepo.CreateSong(&model.Song{
				SongBase: model.SongBase{WikiID: fmt.Sprintf("fitted_%d_%d", season, i), B15: season == 1},
				Charts:   []model.Chart{chart},
			})
			require.NoError(t, err)
			record, err := repo.CreateRecord(&model.PlayRecord{
				Username: "fitted_user", PlayRecordBase: model.PlayRecordBase{ChartID: song.Charts[0].ID, Score: intPtr(1000000)},
			}, false)
			require.NoError(t, err)
			allIDs = append(allIDs, record.ID)
			if i == count-1 {
				promotedIDs = append(promotedIDs, record.ID)
				promotedChartIDs = append(promotedChartIDs, song.Charts[0].ID)
				// A lower historical score and another user's score cannot enter this user's pool.
				for _, username := range []string{"fitted_user", "another_user"} {
					_, err := repo.CreateRecord(&model.PlayRecord{
						Username: username, PlayRecordBase: model.PlayRecordBase{ChartID: record.ChartID, Score: intPtr(900000)},
					}, false)
					require.NoError(t, err)
				}
			}
		}
	}

	officialOld, officialNew, err := repo.GetBest50Records("fitted_user", 0, "rating", true, model.RecordFilter{})
	require.NoError(t, err)
	assert.NotContains(t, recordIDs(officialOld), promotedIDs[0])
	assert.NotContains(t, recordIDs(officialNew), promotedIDs[1])

	fittedOld, fittedNew, err := repo.GetBest50Records("fitted_user", 0, "fitting_rating", true, model.RecordFilter{})
	require.NoError(t, err)
	require.Len(t, fittedOld, 35)
	require.Len(t, fittedNew, 15)
	assert.Equal(t, promotedIDs[0], fittedOld[0].ID)
	assert.Equal(t, promotedIDs[1], fittedNew[0].ID)
	assert.Equal(t, 15900, fittedOld[0].Rating)
	assert.Equal(t, 16200, fittedOld[0].FittingRating())
	assert.Equal(t, 17200, fittedNew[0].FittingRating())
	assert.Equal(t, 16000, fittedOld[1].FittingRating(), "nil fitting levels fall back to official levels")

	t.Run("overflow and ascending presentation retain the highest ratings", func(t *testing.T) {
		old, new, err := repo.GetBest50Records("fitted_user", 5, "fitting_rating", true, model.RecordFilter{})
		require.NoError(t, err)
		require.Len(t, old, 40)
		require.Len(t, new, 20)
		assert.Equal(t, recordIDs(fittedOld), recordIDs(old[:35]))
		assert.Equal(t, recordIDs(fittedNew), recordIDs(new[:15]))
		for _, sortBy := range []string{"rating", "fitting_rating"} {
			descOld, descNew, err := repo.GetBest50Records("fitted_user", 0, sortBy, true, model.RecordFilter{})
			require.NoError(t, err)
			ascOld, ascNew, err := repo.GetBest50Records("fitted_user", 0, sortBy, false, model.RecordFilter{})
			require.NoError(t, err)
			slices.Reverse(descOld)
			slices.Reverse(descNew)
			assert.Equal(t, recordIDs(descOld), recordIDs(ascOld))
			assert.Equal(t, recordIDs(descNew), recordIDs(ascNew))
			// Read both cache entries again to catch order-dependent cache contamination.
			again, _, err := repo.GetBest50Records("fitted_user", 0, sortBy, true, model.RecordFilter{})
			require.NoError(t, err)
			slices.Reverse(descOld)
			assert.Equal(t, recordIDs(descOld), recordIDs(again))
		}
	})

	t.Run("global ranking and stable pagination happen after fitting calculation", func(t *testing.T) {
		global, err := repo.GetBestRecords("fitted_user", 50, 0, "fitting_rating", true, model.RecordFilter{})
		require.NoError(t, err)
		require.Len(t, global, 50)
		assert.Equal(t, promotedIDs[1], global[0].ID)
		newCount := 0
		for _, record := range global {
			if record.Chart.Song.B15 {
				newCount++
			}
		}
		assert.Greater(t, newCount, 15, "global top 50 cannot be derived from seasonal B50")
		for _, descending := range []bool{true, false} {
			var records []model.PlayRecord
			for page := 0; page < 10; page++ {
				batch, err := repo.GetBestRecords("fitted_user", 7, page, "fitting_rating", descending, model.RecordFilter{})
				require.NoError(t, err)
				records = append(records, batch...)
			}
			require.Len(t, records, len(allIDs))
			assert.ElementsMatch(t, allIDs, recordIDs(records))
			for i := 1; i < len(records); i++ {
				a, b := &records[i-1], &records[i]
				if a.FittingRating() == b.FittingRating() {
					assert.Equal(t, descending, a.ID > b.ID)
				} else {
					assert.Equal(t, descending, a.FittingRating() > b.FittingRating())
				}
			}
			assert.Equal(t, 0, records[map[bool]int{true: len(records) - 1, false: 0}[descending]].FittingRating(), "zero is a valid fitting level")
		}
		for _, page := range []int{10, math.MaxInt} {
			records, err := repo.GetBestRecords("fitted_user", 7, page, "fitting_rating", true, model.RecordFilter{})
			require.NoError(t, err)
			assert.Empty(t, records)
		}
	})

	t.Run("filters use official levels before fitting ranking", func(t *testing.T) {
		filter := model.RecordFilter{B15: func() *bool { v := true; return &v }(), MinLevel: float64Ptr(16.95), Difficulties: []model.Difficulty{model.DifficultyMassive}}
		old, new, err := repo.GetBest50Records("fitted_user", 0, "fitting_rating", true, filter)
		require.NoError(t, err)
		assert.Empty(t, old)
		require.Len(t, new, 15)
		assert.NotContains(t, recordIDs(new), promotedIDs[1])
		assert.Equal(t, 17000, new[0].FittingRating())
	})

	t.Run("published fitting updates take effect without record writes", func(t *testing.T) {
		require.NoError(t, db.Model(&model.Chart{}).Where("id IN ?", promotedChartIDs).Update("fitting_level", 15).Error)
		old, new, err := repo.GetBest50Records("fitted_user", 0, "fitting_rating", true, model.RecordFilter{})
		require.NoError(t, err)
		assert.NotContains(t, recordIDs(old), promotedIDs[0])
		assert.NotContains(t, recordIDs(new), promotedIDs[1])
		var stored model.PlayRecord
		require.NoError(t, db.First(&stored, promotedIDs[0]).Error)
		assert.Equal(t, 15900, stored.Rating)
		best, err := repo.GetBestRecords("fitted_user", 100, 0, "fitting_rating", true, model.RecordFilter{})
		require.NoError(t, err)
		for _, record := range best {
			if slices.Contains(promotedIDs, record.ID) {
				assert.Equal(t, 15000, record.FittingRating())
			}
		}
	})

	t.Run("empty user", func(t *testing.T) {
		old, new, err := repo.GetBest50Records("no_records", 0, "fitting_rating", true, model.RecordFilter{})
		require.NoError(t, err)
		assert.Empty(t, old)
		assert.Empty(t, new)
	})

	t.Run("rankings and counts exclude soft-deleted candidates", func(t *testing.T) {
		require.NoError(t, db.Where("play_record_id = ?", promotedIDs[0]).Delete(&model.BestPlayRecord{}).Error)
		var chartRecord, songRecord model.PlayRecord
		require.NoError(t, db.First(&chartRecord, allIDs[0]).Error)
		require.NoError(t, db.First(&songRecord, allIDs[42]).Error)
		var chart model.Chart
		require.NoError(t, db.First(&chart, songRecord.ChartID).Error)
		require.NoError(t, db.Delete(&model.Chart{}, chartRecord.ChartID).Error)
		require.NoError(t, db.Delete(&model.Song{}, chart.SongID).Error)
		require.NoError(t, db.Delete(&model.PlayRecord{}, allIDs[2]).Error)
		repo.invalidateUserRecords("fitted_user")
		for _, sortBy := range []string{"rating", "fitting_rating"} {
			records, err := repo.GetBestRecords("fitted_user", 100, 0, sortBy, true, model.RecordFilter{})
			require.NoError(t, err)
			assert.Len(t, records, len(allIDs)-4)
			for _, id := range []int{promotedIDs[0], allIDs[0], allIDs[42], allIDs[2]} {
				assert.NotContains(t, recordIDs(records), id)
			}
		}
		count, err := repo.CountBestRecords("fitted_user", model.RecordFilter{})
		require.NoError(t, err)
		assert.EqualValues(t, len(allIDs)-4, count)
	})
}
