package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/usecases"
)

const (
	PartnerAPIKeyHeader       = "X-PK-Key"
	PartnerAPITimestampHeader = "X-PK-Timestamp"
	PartnerAPISignatureHeader = "X-PK-Signature"
)

// ApiKeyPartnerMiddleware validates partner-only API key auth without
// affecting the existing dual-auth runtime used by /v1/payment-app.
func ApiKeyPartnerMiddleware(apiKeyUsecase *usecases.ApiKeyUsecase, merchantRepo repositories.MerchantRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(PartnerAPIKeyHeader)
		signature := c.GetHeader(PartnerAPISignatureHeader)
		timestamp := c.GetHeader(PartnerAPITimestampHeader)

		if apiKey == "" || signature == "" || timestamp == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing partner auth headers",
			})
			return
		}

		var bodyBytes []byte
		var err error
		if c.Request.Body != nil {
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "Failed to read request body",
				})
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		user, err := apiKeyUsecase.ValidatePartnerApiKey(
			c.Request.Context(),
			apiKey,
			signature,
			timestamp,
			c.Request.Method,
			c.Request.URL.RequestURI(),
			sha256Hex(bodyBytes),
		)
		if err != nil {
			log.Printf("[ApiKeyPartnerMiddleware] auth failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid partner API key or signature",
			})
			return
		}

		merchant, err := merchantRepo.GetByUserID(c.Request.Context(), user.ID)
		if err != nil || merchant == nil {
			log.Printf("[ApiKeyPartnerMiddleware] merchant context missing for user %s: %v", user.ID, err)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "merchant account required",
			})
			return
		}

		c.Set(UserIDKey, user.ID)
		c.Set(UserEmailKey, user.Email)
		c.Set(UserRoleKey, string(user.Role))
		c.Set(MerchantIDKey, merchant.ID)
		c.Set(IsMerchantAuthenticatedKey, true)
		c.Next()
	}
}
