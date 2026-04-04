package model

// Token represents the authentication token
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type" example:"Bearer"`
}

// UploadToken represents the upload token
type UploadToken struct {
	UploadToken string `json:"upload_token"`
}
