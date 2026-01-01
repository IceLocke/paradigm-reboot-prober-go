package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
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
func (r *RecordRepository) CreateRecord(req *request.BatchCreatePlayRecordRequest, recordBase *model.PlayRecordBase, username string, isReplaced bool) (*model.PlayRecord, error) {
	var songLevel model.SongLevel
	if err := r.db.Where("song_level_id = ?", recordBase.SongLevelID).First(&songLevel).Error; err != nil {
		return nil, errors.New("song level does not exist")
	}

	// Calculate rating
	calculatedRating := rating.SingleRating(songLevel.Level, recordBase.Score)

	record := model.PlayRecord{
		PlayRecordBase: *recordBase,
		Username:       username,
		RecordTime:     time.Now(),
		Rating:         calculatedRating,
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		var bestRecord model.BestPlayRecord
		result := tx.Joins("PlayRecord").
			Where("PlayRecord.song_level_id = ? AND PlayRecord.username = ?", record.SongLevelID, username).
			First(&bestRecord)

		if result.Error == nil {
			// Best record exists
			if isReplaced || record.Score > bestRecord.PlayRecord.Score {
				bestRecord.PlayRecordID = record.PlayRecordID
				bestRecord.PlayRecord = &record
				if err := tx.Save(&bestRecord).Error; err != nil {
					return err
				}
			}
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new best record
			newBestRecord := model.BestPlayRecord{
				PlayRecordID: record.PlayRecordID,
				PlayRecord:   &record,
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

	return &record, nil
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
