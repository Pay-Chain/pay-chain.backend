package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/internal/infrastructure/metrics"
)

type DeprecationOptions struct {
	Replacement    string
	Sunset         time.Time
	EndpointFamily string
	Mode           string
}

func DeprecationMiddleware(opts DeprecationOptions) gin.HandlerFunc {
	replacement := opts.Replacement
	endpointFamily := opts.EndpointFamily
	sunset := opts.Sunset.UTC().Format(time.RFC1123)
	mode := NormalizeLegacyMode(opts.Mode)

	return func(c *gin.Context) {
		merchantID := ""
		if raw, exists := c.Get(MerchantIDKey); exists {
			if id, ok := raw.(interface{ String() string }); ok {
				merchantID = id.String()
			}
		}

		metrics.RecordLegacyEndpointUsage(endpointFamily, merchantID)
		recordLegacyEndpointHit(endpointFamily, replacement, opts.Sunset.UTC(), mode, merchantID)

		c.Header("Deprecation", "true")
		c.Header("Sunset", sunset)
		c.Header("Link", "<"+replacement+">; rel=\"successor-version\"")
		c.Header("X-Deprecated-Replaced-By", replacement)
		c.Header("X-Legacy-Endpoint-Mode", mode)

		if mode == LegacyModeDisabled {
			c.AbortWithStatusJSON(http.StatusGone, gin.H{
				"code":                 "legacy_endpoint_disabled",
				"message":              "legacy endpoint has been disabled",
				"replacement":          replacement,
				"deprecated_endpoint":  endpointFamily,
				"legacy_endpoint_mode": mode,
			})
			return
		}

		c.Next()
	}
}
