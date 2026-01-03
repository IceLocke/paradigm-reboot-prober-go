package model

// Response is a common response structure for errors and simple messages
type Response struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
