package domain

import "fmt"

// ErrorCode is a stable machine-readable error identifier returned to clients.
type ErrorCode string

const (
	ErrCodeInvalidUUID        ErrorCode = "INVALID_UUID"
	ErrCodeInvalidType        ErrorCode = "INVALID_TYPE"
	ErrCodeInvalidParameter   ErrorCode = "INVALID_PARAMETER"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// Error is a domain error carrying a stable code and a human-readable message.
type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func NewInvalidUUID(message string) *Error {
	return &Error{Code: ErrCodeInvalidUUID, Message: message}
}

func NewInvalidType(message string) *Error {
	return &Error{Code: ErrCodeInvalidType, Message: message}
}

func NewInvalidParameter(message string) *Error {
	return &Error{Code: ErrCodeInvalidParameter, Message: message}
}

func NewNotFound(message string) *Error {
	return &Error{Code: ErrCodeNotFound, Message: message}
}

func NewServiceUnavailable(message string, err error) *Error {
	return &Error{Code: ErrCodeServiceUnavailable, Message: message, Err: err}
}
