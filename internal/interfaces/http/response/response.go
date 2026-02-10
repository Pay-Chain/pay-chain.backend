package response

import (
	"github.com/gin-gonic/gin"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// Success sends a success response
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// Error sends an error response
func Error(c *gin.Context, err error) {
	var appErr *domainerrors.AppError
	if e, ok := err.(*domainerrors.AppError); ok {
		appErr = e
	} else {
		// Default to Internal Server Error if not an AppError
		appErr = domainerrors.InternalError(err)
	}

	c.JSON(appErr.Status, gin.H{
		"code":    appErr.Code,
		"message": appErr.Message,
		"error":   appErr.Message, // Backward compatibility
	})
}

// ErrorWithStatus sends an error response with a specific status and message
func ErrorWithError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"code":    code,
		"message": message,
	})
}
