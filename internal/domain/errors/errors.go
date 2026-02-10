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

// Standard Error Codes
const (
	CodeNotFound           = "ERR_NOT_FOUND"
	CodeAlreadyExists      = "ERR_ALREADY_EXISTS"
	CodeInvalidInput       = "ERR_INVALID_INPUT"
	CodeBadRequest         = "ERR_BAD_REQUEST"
	CodeUnauthorized       = "ERR_UNAUTHORIZED"
	CodeForbidden          = "ERR_FORBIDDEN"
	CodeInternalError      = "ERR_INTERNAL_ERROR"
	CodeInvalidCredentials = "ERR_INVALID_CREDENTIALS"
	CodeTokenExpired       = "ERR_TOKEN_EXPIRED"
	CodeEmailNotVerified   = "ERR_EMAIL_NOT_VERIFIED"
	CodePaymentFailed      = "ERR_PAYMENT_FAILED"
	CodeInsufficientFunds  = "ERR_INSUFFICIENT_FUNDS"
	CodeConflict           = "ERR_CONFLICT"
)

// AppError represents application error with HTTP status and string code
type AppError struct {
	Status  int    `json:"-"`       // HTTP Status Code
	Code    string `json:"code"`    // Machine-readable Code
	Message string `json:"message"` // Human-readable Message
	Err     error  `json:"-"`       // Underlying Error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewAppError creates a new app error
func NewAppError(status int, code string, message string, err error) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error constructors
func NotFound(message string) *AppError {
	return NewAppError(http.StatusNotFound, CodeNotFound, message, ErrNotFound)
}

func BadRequest(message string) *AppError {
	return NewAppError(http.StatusBadRequest, CodeInvalidInput, message, ErrInvalidInput)
}

func Unauthorized(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, CodeUnauthorized, message, ErrUnauthorized)
}

func Forbidden(message string) *AppError {
	return NewAppError(http.StatusForbidden, CodeForbidden, message, ErrForbidden)
}

func InternalError(err error) *AppError {
	return NewAppError(http.StatusInternalServerError, CodeInternalError, "internal server error", err)
}

// Conflict creates a conflict error
func Conflict(message string) *AppError {
	return NewAppError(http.StatusConflict, CodeConflict, message, ErrAlreadyExists)
}

// NewError creates a new error with a custom message wrapping an existing error
// Defaulting to Bad Request generic for compatibility, but ideally should be specific.
func NewError(message string, err error) error {
	return &AppError{
		Status:  http.StatusBadRequest,
		Code:    CodeBadRequest,
		Message: message,
		Err:     err,
	}
}
