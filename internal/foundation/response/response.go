package response

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// SuccessResponse is the standard envelope for successful responses
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

// ErrorDetail describes the error body
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ErrorResponse is the standard envelope for error responses
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// Success returns a standard JSON success response
func Success(c *fiber.Ctx, statusCode int, message string, data any) error {
	return c.Status(statusCode).JSON(SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta returns a standard JSON success response with metadata (e.g. pagination)
func SuccessWithMeta(c *fiber.Ctx, statusCode int, message string, data any, meta any) error {
	return c.Status(statusCode).JSON(SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// OK returns a 200 OK success response
func OK(c *fiber.Ctx, message string, data any) error {
	return Success(c, http.StatusOK, message, data)
}

// Created returns a 201 Created success response
func Created(c *fiber.Ctx, message string, data any) error {
	return Success(c, http.StatusCreated, message, data)
}

// Error returns a standard JSON error response
func Error(c *fiber.Ctx, statusCode int, errorCode, message string, details any) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	})
}
