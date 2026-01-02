package service

import (
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/repository"
)

type RecordService struct {
	recordRepo *repository.RecordRepository
	songRepo   *repository.SongRepository
}

func NewRecordService(recordRepo *repository.RecordRepository, songRepo *repository.SongRepository) *RecordService {
	return &RecordService{
		recordRepo: recordRepo,
		songRepo:   songRepo,
	}
}

func (s *RecordService) CreateRecords(username string, records []model.PlayRecordBase, isReplaced bool) ([]*model.PlayRecord, error) {
	var responseRecords []*model.PlayRecord

	for _, recordBase := range records {
		// The repository handles rating calculation and best record update.
		// We just need to call it.

		// Construct the model.PlayRecord
		// Note: Rating and RecordTime will be set by repository
		record := &model.PlayRecord{
			PlayRecordBase: recordBase,
			Username:       username,
		}

		savedRecord, err := s.recordRepo.CreateRecord(record, isReplaced)
		if err != nil {
			return nil, err
		}
		responseRecords = append(responseRecords, savedRecord)
	}

	return responseRecords, nil
}

func (s *RecordService) GetAllRecords(username string, pageSize, pageIndex int, sortBy string, order string) ([]model.PlayRecord, error) {
	return s.recordRepo.GetAllRecords(username, pageSize, pageIndex, sortBy, order == "desc")
}

func (s *RecordService) GetBest50Records(username string, underflow int) ([]*model.PlayRecord, error) {
	b35, b15, err := s.recordRepo.GetBest50Records(username, underflow)
	if err != nil {
		return nil, err
	}

	var records []*model.PlayRecord
	for i := range b35 {
		records = append(records, &b35[i])
	}
	for i := range b15 {
		records = append(records, &b15[i])
	}

	return records, nil
}

func (s *RecordService) GetBestRecords(username string, pageSize, pageIndex int, sortBy string, order string) ([]model.PlayRecord, error) {
	return s.recordRepo.GetBestRecords(username, pageSize, pageIndex, sortBy, order == "desc")
}

func (s *RecordService) GetAllLevelsWithBestScores(username string) ([]model.SongLevelWithScore, error) {
	return s.recordRepo.GetAllLevelsWithBestScores(username)
}

func (s *RecordService) CountBestRecords(username string) (int64, error) {
	return s.recordRepo.CountBestRecords(username)
}

func (s *RecordService) CountAllRecords(username string) (int64, error) {
	return s.recordRepo.CountAllRecords(username)
}
