package errors

import (
	"fmt"
)

// Application-specific error types
type ErrType string

const (
	ErrTypeConfig   ErrType = "CONFIG"
	ErrTypeAI       ErrType = "AI"
	ErrTypeDiscord  ErrType = "DISCORD"
	ErrTypeDatabase ErrType = "DATABASE"
	ErrTypeNetwork  ErrType = "NETWORK"
)

type AppError struct {
	Type         ErrType
	Message      string
	Cause        error
	UserFriendly string // User-facing error message
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// Constructor functions
func NewConfigError(message string, cause error) *AppError {
	return &AppError{
		Type:         ErrTypeConfig,
		Message:      message,
		Cause:        cause,
		UserFriendly: "Configuration error occurred",
	}
}

func NewAIError(message string, cause error) *AppError {
	return &AppError{
		Type:         ErrTypeAI,
		Message:      message,
		Cause:        cause,
		UserFriendly: "🔧 My AI circuits are experiencing difficulties. Please try again.",
	}
}

// ... more error constructors
