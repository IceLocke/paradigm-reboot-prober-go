package request

import "paradigm-reboot-prober-go/internal/model"

// BatchCreatePlayRecordRequest represents the request to batch upload play records
type BatchCreatePlayRecordRequest struct {
	UploadToken string                 `json:"upload_token"`
	IsReplace   bool                   `json:"is_replace"`
	PlayRecords []model.PlayRecordBase `json:"play_records" binding:"required,min=1,max=500,dive"`
}
