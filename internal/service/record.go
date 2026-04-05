package service

import (
	"context"
	"log/slog"
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

func (s *RecordService) CreateRecords(ctx context.Context, username string, records []model.PlayRecordBase, isReplaced bool) ([]*model.PlayRecord, error) {
	var playRecords []*model.PlayRecord
	for _, recordBase := range records {
		playRecords = append(playRecords, &model.PlayRecord{
			PlayRecordBase: recordBase,
			Username:       username,
		})
	}
	results, err := s.recordRepo.BatchCreateRecords(playRecords, isReplaced)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create records", "error", err, "count", len(records))
		return nil, err
	}
	slog.InfoContext(ctx, "records uploaded", "count", len(results))
	return results, nil
}

func (s *RecordService) GetAllRecords(ctx context.Context, username string, pageSize, pageIndex int, sortBy string, order string) ([]model.PlayRecord, error) {
	return s.recordRepo.GetAllRecords(username, pageSize, pageIndex, sortBy, order == "desc")
}

func (s *RecordService) GetBest50Records(ctx context.Context, username string, underflow int) ([]*model.PlayRecord, error) {
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

func (s *RecordService) GetBestRecords(ctx context.Context, username string, pageSize, pageIndex int, sortBy string, order string) ([]model.PlayRecord, error) {
	return s.recordRepo.GetBestRecords(username, pageSize, pageIndex, sortBy, order == "desc")
}

func (s *RecordService) GetAllChartsWithBestScores(ctx context.Context, username string) ([]model.ChartWithScore, error) {
	return s.recordRepo.GetAllChartsWithBestScores(username)
}

func (s *RecordService) CountBestRecords(ctx context.Context, username string) (int64, error) {
	return s.recordRepo.CountBestRecords(username)
}

func (s *RecordService) CountAllRecords(ctx context.Context, username string) (int64, error) {
	return s.recordRepo.CountAllRecords(username)
}

func (s *RecordService) GetBestRecordsBySong(ctx context.Context, username string, songID int) ([]model.PlayRecord, error) {
	return s.recordRepo.GetBestRecordsBySong(username, songID)
}

func (s *RecordService) GetAllRecordsBySong(ctx context.Context, username string, songID int, pageSize, pageIndex int, sortBy, order string) ([]model.PlayRecord, error) {
	return s.recordRepo.GetAllRecordsBySong(username, songID, pageSize, pageIndex, sortBy, order == "desc")
}

func (s *RecordService) CountAllRecordsBySong(ctx context.Context, username string, songID int) (int64, error) {
	return s.recordRepo.CountAllRecordsBySong(username, songID)
}

func (s *RecordService) GetBestRecordByChart(ctx context.Context, username string, chartID int) (*model.PlayRecord, error) {
	return s.recordRepo.GetBestRecordByChart(username, chartID)
}

func (s *RecordService) GetAllRecordsByChart(ctx context.Context, username string, chartID int, pageSize, pageIndex int, sortBy, order string) ([]model.PlayRecord, error) {
	return s.recordRepo.GetAllRecordsByChart(username, chartID, pageSize, pageIndex, sortBy, order == "desc")
}

func (s *RecordService) CountAllRecordsByChart(ctx context.Context, username string, chartID int) (int64, error) {
	return s.recordRepo.CountAllRecordsByChart(username, chartID)
}
