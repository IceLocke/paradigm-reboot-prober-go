package request

import "paradigm-reboot-prober-go/internal/model"

// BatchCreatePlayRecordRequest represents the request to batch upload play records
type BatchCreatePlayRecordRequest struct {
	UploadToken string                 `json:"upload_token"`
	CSVFilename string                 `json:"csv_filename"`
	IsReplace   bool                   `json:"is_replace"`
	PlayRecords []model.PlayRecordBase `json:"play_records"`
}
