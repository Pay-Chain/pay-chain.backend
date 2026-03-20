package middleware

import "os"

func LegacyModeFromEnv(envKey string) string {
	return NormalizeLegacyMode(os.Getenv(envKey))
}
