package common

import (
	"errors"
	"fmt"
)

// Sentinel errors for the system.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrTimeout       = errors.New("operation timed out")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrInternal      = errors.New("internal error")
	ErrClosed        = errors.New("closed")
)

// AppError wraps an error with code and message.
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// NewAppError creates a new AppError.
func NewAppError(code int, msg string, err error) *AppError {
	return &AppError{Code: code, Message: msg, Err: err}
}

// WrapError wraps an error with additional context.
func WrapError(msg string, err error) *AppError {
	return &AppError{Code: 500, Message: msg, Err: err}
}
