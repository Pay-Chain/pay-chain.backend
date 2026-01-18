package errors

import (
	"errors"
	"net/http"
)

// Domain errors
var (
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyExists      = errors.New("resource already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrBadRequest         = errors.New("bad request")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("token expired")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrMerchantNotActive  = errors.New("merchant not active")
	ErrPaymentFailed      = errors.New("payment failed")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrUnsupportedChain   = errors.New("unsupported chain")
	ErrUnsupportedToken   = errors.New("unsupported token")
)

// AppError represents application error with HTTP status
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewAppError creates a new app error
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error constructors
func NotFound(message string) *AppError {
	return NewAppError(http.StatusNotFound, message, ErrNotFound)
}

func BadRequest(message string) *AppError {
	return NewAppError(http.StatusBadRequest, message, ErrInvalidInput)
}

func Unauthorized(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, message, ErrUnauthorized)
}

func Forbidden(message string) *AppError {
	return NewAppError(http.StatusForbidden, message, ErrForbidden)
}

func InternalError(err error) *AppError {
	return NewAppError(http.StatusInternalServerError, "internal server error", err)
}

// NewError creates a new error with a custom message wrapping an existing error
func NewError(message string, err error) error {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     err,
	}
}
