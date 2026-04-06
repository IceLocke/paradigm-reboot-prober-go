package request

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required" example:"user123"`
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Nickname string `json:"nickname" example:"小明"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
}

// UpdateUserRequest represents the request to update an existing user
type UpdateUserRequest struct {
	Nickname       *string `json:"nickname" binding:"omitempty,min=0"`
	QQAccount      *string `json:"qq_account" binding:"omitempty"`
	Account        *string `json:"account"`
	AccountNumber  *int    `json:"account_number" binding:"omitempty,min=0"`
	UUID           *string `json:"uuid"`
	AnonymousProbe *bool   `json:"anonymous_probe"`
}

// ChangePasswordRequest represents the request to change the current user's password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required" example:"oldpass123"`
	NewPassword string `json:"new_password" binding:"required,min=6" example:"newpass456"`
}

// ResetPasswordRequest represents the admin request to reset a user's password
type ResetPasswordRequest struct {
	Username    string `json:"username" binding:"required" example:"targetuser"`
	NewPassword string `json:"new_password" binding:"required,min=6" example:"newpass456"`
}
