package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a standardized application error
type AppError struct {
	StatusCode int    `json:"statusCode"`
	ErrorCode  string `json:"errorCode,omitempty"`
	Message    string `json:"message"`
	Details    any    `json:"details,omitempty"`
	RawError   error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.RawError != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.RawError)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.RawError
}

func New(statusCode int, errorCode, message string, details any, rawErr error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		ErrorCode:  errorCode,
		Message:    message,
		Details:    details,
		RawError:   rawErr,
	}
}

func NewBadRequest(message string, details ...any) *AppError {
	var d any
	if len(details) > 0 {
		d = details[0]
	}
	return &AppError{
		StatusCode: http.StatusBadRequest,
		ErrorCode:  "BAD_REQUEST",
		Message:    message,
		Details:    d,
	}
}

func NewNotFound(message string) *AppError {
	return &AppError{
		StatusCode: http.StatusNotFound,
		ErrorCode:  "NOT_FOUND",
		Message:    message,
	}
}

func NewUnauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return &AppError{
		StatusCode: http.StatusUnauthorized,
		ErrorCode:  "UNAUTHORIZED",
		Message:    message,
	}
}

func NewForbidden(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return &AppError{
		StatusCode: http.StatusForbidden,
		ErrorCode:  "FORBIDDEN",
		Message:    message,
	}
}

func NewConflict(message string) *AppError {
	return &AppError{
		StatusCode: http.StatusConflict,
		ErrorCode:  "CONFLICT",
		Message:    message,
	}
}

func NewValidation(details any, message ...string) *AppError {
	msg := "Validation failed"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &AppError{
		StatusCode: http.StatusUnprocessableEntity,
		ErrorCode:  "VALIDATION_ERROR",
		Message:    msg,
		Details:    details,
	}
}

func NewServiceUnavailable(message string) *AppError {
	if message == "" {
		message = "Service unavailable"
	}
	return &AppError{
		StatusCode: http.StatusServiceUnavailable,
		ErrorCode:  "SERVICE_UNAVAILABLE",
		Message:    message,
	}
}

func NewPaymentRequired(message string) *AppError {
	if message == "" {
		message = "Payment required"
	}
	return &AppError{
		StatusCode: http.StatusPaymentRequired,
		ErrorCode:  "LICENSE_EXPIRED",
		Message:    message,
	}
}

func NewInternal(rawErr error, message ...string) *AppError {
	msg := "Internal server error"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &AppError{
		StatusCode: http.StatusInternalServerError,
		ErrorCode:  "INTERNAL_ERROR",
		Message:    msg,
		RawError:   rawErr,
	}
}
