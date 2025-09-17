package helpers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// RespSuccess returns a successful API response
func RespSuccess(c echo.Context, message string, data interface{}) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// RespFailure returns a failure API response
func RespFailure(c echo.Context, message string, err error) error {
	return c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Message: message,
		Error: &ErrorInfo{
			Type:    "internal_error",
			Message: err.Error(),
		},
	})
}

// RespBadRequest returns a bad request response
func RespBadRequest(c echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Message: message,
		Error: &ErrorInfo{
			Type: "bad_request",
			Message: message,
			Code: http.StatusBadRequest,
		},
	})
}

// RespNotFound returns a not found response
func RespNotFound(c echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Message: message,
		Error: &ErrorInfo{
			Type: "not_found",
			Message: message,
			Code: http.StatusNotFound,
		},
	})
}

// RespUnauthorized returns an unauthorized response
func RespUnauthorized(c echo.Context, message string) error {
	return c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Message: message,
		Error: &ErrorInfo{
			Type: "unauthorized",
			Message: message,
			Code: http.StatusUnauthorized,
		},
	})
}

// RespConflict returns a conflict response
func RespConflict(c echo.Context, message string) error {
	return c.JSON(http.StatusConflict, APIResponse{
		Success: false,
		Message: message,
		Error: &ErrorInfo{
			Type: "conflict",
			Message: message,
			Code: http.StatusConflict,
		},
	})
}