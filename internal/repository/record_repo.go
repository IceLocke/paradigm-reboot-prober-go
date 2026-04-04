package repository

import (
	"errors"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"
	"time"

	"gorm.io/gorm"
)

// allowedSortColumns defines the whitelist of columns that can be used for sorting
var allowedSortColumns = map[string]string{
	"rating":      "rating",
	"score":       "score",
	"record_time": "record_time",
}

// validateSortBy returns a safe column name for ORDER BY, defaulting to "rating" if not allowed
func validateSortBy(sortBy string) string {
	if col, ok := allowedSortColumns[sortBy]; ok {
		return col
	}
	return "rating" // default
}

type RecordRepository struct {
	db *gorm.DB
}

func NewRecordRepository(db *gorm.DB) *RecordRepository {
	return &RecordRepository{db: db}
}

// CreateRecord creates a new play record and updates the best record if necessary
func (r *RecordRepository) CreateRecord(record *model.PlayRecord, isReplaced bool) (*model.PlayRecord, error) {
	var result *model.PlayRecord
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var txErr error
		result, txErr = r.createRecordInTx(tx, record, isReplaced)
		return txErr
	})
	return result, err
}

// BatchCreateRecords creates multiple play records atomically in a single transaction
func (r *RecordRepository) BatchCreateRecords(records []*model.PlayRecord, isReplaced bool) ([]*model.PlayRecord, error) {
	var results []*model.PlayRecord
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			savedRecord, err := r.createRecordInTx(tx, record, isReplaced)
			if err != nil {
				return err
			}
			results = append(results, savedRecord)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// createRecordInTx handles creating a single record within an existing transaction.
func (r *RecordRepository) createRecordInTx(tx *gorm.DB, record *model.PlayRecord, isReplaced bool) (*model.PlayRecord, error) {
	var chart model.Chart
	if err := tx.Where("id = ?", record.ChartID).First(&chart).Error; err != nil {
		return nil, errors.New("chart does not exist")
	}

	// Calculate rating
	calculatedRating := rating.SingleRating(chart.Level, record.Score)
	record.Rating = calculatedRating
	record.RecordTime = time.Now()

	if err := tx.Create(record).Error; err != nil {
		return nil, err
	}

	// Find existing best record using indexed columns
	var bestRecord model.BestPlayRecord
	result := tx.Preload("PlayRecord").
		Where("username = ? AND chart_id = ?", record.Username, record.ChartID).
		First(&bestRecord)

	if result.Error == nil {
		// Best record exists
		if isReplaced || record.Score > bestRecord.PlayRecord.Score {
			bestRecord.PlayRecordID = record.ID
			bestRecord.PlayRecord = record
			if err := tx.Save(&bestRecord).Error; err != nil {
				return nil, err
			}
		}
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Create new best record
		newBestRecord := model.BestPlayRecord{
			Username:     record.Username,
			ChartID:      record.ChartID,
			PlayRecordID: record.ID,
			PlayRecord:   record,
		}
		if err := tx.Create(&newBestRecord).Error; err != nil {
			return nil, err
		}
	} else {
		return nil, result.Error
	}

	return record, nil
}

// GetBest50Records retrieves the best 35 (old) and 15 (new) records for B50 calculation
func (r *RecordRepository) GetBest50Records(username string, underflow int) ([]model.PlayRecord, []model.PlayRecord, error) {
	var b35 []model.PlayRecord
	var b15 []model.PlayRecord

	// Base query for best records
	baseQuery := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
		Joins("Chart").
		Joins("Chart.Song").
		Where("play_records.username = ?", username)

	// B35: Not B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where("Chart__Song.b15 = ?", false).
		Order("rating desc, play_records.id desc").
		Limit(config.GlobalConfig.Game.B35Limit + underflow).
		Find(&b35).Error; err != nil {
		return nil, nil, err
	}

	// B15: B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where("Chart__Song.b15 = ?", true).
		Order("rating desc, play_records.id desc").
		Limit(config.GlobalConfig.Game.B15Limit + underflow).
		Find(&b15).Error; err != nil {
		return nil, nil, err
	}

	return b35, b15, nil
}

// GetAllRecords retrieves all records for a user with pagination and sorting
func (r *RecordRepository) GetAllRecords(username string, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("username = ?", username).
		Joins("Chart").
		Joins("Chart.Song")

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	// Use whitelist validation to prevent SQL injection
	safeSortBy := validateSortBy(sortBy)
	query = query.Order(safeSortBy + " " + orderStr)

	// pageIndex is 0-indexed from the service layer
	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// GetBestRecords retrieves the best records for a user with pagination and sorting
func (r *RecordRepository) GetBestRecords(username string, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
		Joins("Chart").
		Joins("Chart.Song").
		Where("play_records.username = ?", username)

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	// Use whitelist validation to prevent SQL injection
	safeSortBy := validateSortBy(sortBy)
	query = query.Order(safeSortBy + " " + orderStr)

	// pageIndex is 0-indexed from the service layer
	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// GetAllChartsWithBestScores retrieves all charts with the user's best score (if any)
func (r *RecordRepository) GetAllChartsWithBestScores(username string) ([]model.ChartWithScore, error) {
	var results []model.ChartWithScore

	err := r.db.Table("charts").
		Select("charts.id, songs.title, songs.version, charts.difficulty, charts.level, COALESCE(play_records.score, 0) as score").
		Joins("JOIN songs ON charts.song_id = songs.id").
		Joins("LEFT JOIN play_records ON charts.id = play_records.chart_id AND play_records.username = ?", username).
		Joins("LEFT JOIN best_play_records ON play_records.id = best_play_records.play_record_id").
		Where("play_records.id IS NULL OR best_play_records.play_record_id IS NOT NULL").
		Scan(&results).Error

	return results, err
}

// CountBestRecords counts the number of best records for a user
func (r *RecordRepository) CountBestRecords(username string) (int64, error) {
	var count int64
	err := r.db.Model(&model.BestPlayRecord{}).
		Where("username = ?", username).
		Count(&count).Error
	return count, err
}

// CountAllRecords counts the total number of records for a user
func (r *RecordRepository) CountAllRecords(username string) (int64, error) {
	var count int64
	err := r.db.Model(&model.PlayRecord{}).
		Where("username = ?", username).
		Count(&count).Error
	return count, err
}

// GetBestRecordsBySong retrieves the best record per difficulty for a specific song
func (r *RecordRepository) GetBestRecordsBySong(username string, songID int) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	err := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
		Joins("Chart").
		Joins("Chart.Song").
		Where("play_records.username = ? AND Chart.song_id = ?", username, songID).
		Order("rating desc").
		Find(&records).Error
	return records, err
}

// GetAllRecordsBySong retrieves all records for a specific song with pagination and sorting
func (r *RecordRepository) GetAllRecordsBySong(username string, songID int, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("play_records.username = ?", username).
		Joins("Chart").
		Joins("Chart.Song").
		Where("Chart.song_id = ?", songID)

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	safeSortBy := validateSortBy(sortBy)
	query = query.Order(safeSortBy + " " + orderStr)

	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// CountAllRecordsBySong counts the total number of records for a specific song
func (r *RecordRepository) CountAllRecordsBySong(username string, songID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN charts ON charts.id = play_records.chart_id").
		Where("play_records.username = ? AND charts.song_id = ?", username, songID).
		Count(&count).Error
	return count, err
}

// GetBestRecordByChart retrieves the best record for a specific chart
func (r *RecordRepository) GetBestRecordByChart(username string, chartID int) (*model.PlayRecord, error) {
	var record model.PlayRecord
	err := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
		Joins("Chart").
		Joins("Chart.Song").
		Where("play_records.username = ? AND play_records.chart_id = ?", username, chartID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// GetAllRecordsByChart retrieves all records for a specific chart with pagination and sorting
func (r *RecordRepository) GetAllRecordsByChart(username string, chartID int, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("play_records.username = ? AND play_records.chart_id = ?", username, chartID).
		Joins("Chart").
		Joins("Chart.Song")

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	safeSortBy := validateSortBy(sortBy)
	query = query.Order(safeSortBy + " " + orderStr)

	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// CountAllRecordsByChart counts the total number of records for a specific chart
func (r *RecordRepository) CountAllRecordsByChart(username string, chartID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.PlayRecord{}).
		Where("username = ? AND chart_id = ?", username, chartID).
		Count(&count).Error
	return count, err
}
