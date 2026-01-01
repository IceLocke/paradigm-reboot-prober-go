package request

import "paradigm-reboot-prober-go/internal/model"

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	model.UserBase
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
}

// UpdateUserRequest represents the request to update an existing user
type UpdateUserRequest struct {
	Nickname       *string `json:"nickname"`
	QQNumber       *int    `json:"qq_number"`
	Account        *string `json:"account"`
	AccountNumber  *int    `json:"account_number"`
	UUID           *string `json:"uuid"`
	AnonymousProbe *bool   `json:"anonymous_probe"`
}
