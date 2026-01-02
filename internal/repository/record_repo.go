package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"
	"time"

	"gorm.io/gorm"
)

type RecordRepository struct {
	db *gorm.DB
}

func NewRecordRepository(db *gorm.DB) *RecordRepository {
	return &RecordRepository{db: db}
}

// CreateRecord creates a new play record and updates the best record if necessary
func (r *RecordRepository) CreateRecord(record *model.PlayRecord, isReplaced bool) (*model.PlayRecord, error) {
	var songLevel model.SongLevel
	if err := r.db.Where("song_level_id = ?", record.SongLevelID).First(&songLevel).Error; err != nil {
		return nil, errors.New("song level does not exist")
	}

	// Calculate rating
	calculatedRating := rating.SingleRating(songLevel.Level, record.Score)
	record.Rating = calculatedRating
	record.RecordTime = time.Now()

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		var bestRecord model.BestPlayRecord
		result := tx.Joins("PlayRecord").
			Where("PlayRecord.song_level_id = ? AND PlayRecord.username = ?", record.SongLevelID, record.Username).
			First(&bestRecord)

		if result.Error == nil {
			// Best record exists
			if isReplaced || record.Score > bestRecord.PlayRecord.Score {
				bestRecord.PlayRecordID = record.PlayRecordID
				bestRecord.PlayRecord = record
				if err := tx.Save(&bestRecord).Error; err != nil {
					return err
				}
			}
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new best record
			newBestRecord := model.BestPlayRecord{
				PlayRecordID: record.PlayRecordID,
				PlayRecord:   record,
			}
			if err := tx.Create(&newBestRecord).Error; err != nil {
				return err
			}
		} else {
			return result.Error
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return record, nil
}

// GetBest50Records retrieves the best 35 (old) and 15 (new) records for B50 calculation
func (r *RecordRepository) GetBest50Records(username string, underflow int) ([]model.PlayRecord, []model.PlayRecord, error) {
	var b35 []model.PlayRecord
	var b15 []model.PlayRecord

	// Base query for best records
	baseQuery := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.play_record_id").
		Joins("SongLevel").
		Joins("SongLevel.Song").
		Where("play_records.username = ?", username)

	// B35: Not B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where("SongLevel__Song.b15 = ?", false).
		Order("rating desc").
		Limit(35 + underflow).
		Find(&b35).Error; err != nil {
		return nil, nil, err
	}

	// B15: B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where("SongLevel__Song.b15 = ?", true).
		Order("rating desc").
		Limit(15 + underflow).
		Find(&b15).Error; err != nil {
		return nil, nil, err
	}

	return b35, b15, nil
}

// GetAllRecords retrieves all records for a user with pagination and sorting
func (r *RecordRepository) GetAllRecords(username string, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("username = ?", username).
		Joins("SongLevel").
		Joins("SongLevel.Song")

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	// Handle sorting logic (simplified mapping)
	// In production, map sortBy string to actual column names safely
	query = query.Order(sortBy + " " + orderStr)

	err := query.Offset(pageSize * (pageIndex - 1)).Limit(pageSize).Find(&records).Error
	return records, err
}

// GetBestRecords retrieves the best records for a user with pagination and sorting
func (r *RecordRepository) GetBestRecords(username string, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.play_record_id").
		Joins("SongLevel").
		Joins("SongLevel.Song").
		Where("play_records.username = ?", username)

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	// Handle sorting logic (simplified mapping)
	query = query.Order(sortBy + " " + orderStr)

	err := query.Offset(pageSize * (pageIndex - 1)).Limit(pageSize).Find(&records).Error
	return records, err
}

// GetAllLevelsWithBestScores retrieves all song levels with the user's best score (if any)
func (r *RecordRepository) GetAllLevelsWithBestScores(username string) ([]model.SongLevelWithScore, error) {
	var results []model.SongLevelWithScore

	err := r.db.Table("song_levels").
		Select("song_levels.song_level_id, songs.title, songs.version, song_levels.difficulty, song_levels.level, COALESCE(play_records.score, 0) as score").
		Joins("JOIN songs ON song_levels.song_id = songs.song_id").
		Joins("LEFT JOIN play_records ON song_levels.song_level_id = play_records.song_level_id AND play_records.username = ?", username).
		Joins("LEFT JOIN best_play_records ON play_records.play_record_id = best_play_records.play_record_id").
		Where("play_records.play_record_id IS NULL OR best_play_records.play_record_id IS NOT NULL").
		Scan(&results).Error

	return results, err
}

// CountBestRecords counts the number of best records for a user
func (r *RecordRepository) CountBestRecords(username string) (int64, error) {
	var count int64
	err := r.db.Model(&model.BestPlayRecord{}).
		Joins("JOIN play_records ON best_play_records.play_record_id = play_records.play_record_id").
		Where("play_records.username = ?", username).
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
