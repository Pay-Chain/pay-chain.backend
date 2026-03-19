package middleware

import "errors"

var (
	// ErrIdempotencyKeyTooLong is returned when the idempotency key exceeds the maximum length
	ErrIdempotencyKeyTooLong = errors.New("idempotency key too long")
	
	// ErrIdempotencyKeyInvalidChar is returned when the idempotency key contains invalid characters
	ErrIdempotencyKeyInvalidChar = errors.New("idempotency key contains invalid characters")
)
