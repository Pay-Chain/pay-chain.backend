package middleware

import (
	"slices"
	"sync"
	"time"
)

const (
	LegacyModeWarn     = "warn"
	LegacyModeDisabled = "disabled"
)

type LegacyEndpointObservabilityEntry struct {
	EndpointFamily string           `json:"endpoint_family"`
	Replacement    string           `json:"replacement"`
	SunsetAt       time.Time        `json:"sunset_at"`
	Mode           string           `json:"mode"`
	TotalHits      int64            `json:"total_hits"`
	LastSeenAt     *time.Time       `json:"last_seen_at,omitempty"`
	MerchantHits   map[string]int64 `json:"merchant_hits,omitempty"`
}

type LegacyEndpointObservabilitySummary struct {
	TrackedEndpoints int        `json:"tracked_endpoints"`
	TotalHits        int64      `json:"total_hits"`
	LastSeenAt       *time.Time `json:"last_seen_at,omitempty"`
}

type LegacyEndpointObservabilitySnapshot struct {
	GeneratedAt time.Time                          `json:"generated_at"`
	Summary     LegacyEndpointObservabilitySummary `json:"summary"`
	Endpoints   []LegacyEndpointObservabilityEntry `json:"endpoints"`
}

type legacyEndpointObservation struct {
	Replacement  string
	SunsetAt     time.Time
	Mode         string
	TotalHits    int64
	LastSeenAt   *time.Time
	MerchantHits map[string]int64
}

var legacyEndpointObservabilityStore = struct {
	mu      sync.RWMutex
	entries map[string]*legacyEndpointObservation
}{
	entries: map[string]*legacyEndpointObservation{},
}

func NormalizeLegacyMode(mode string) string {
	switch mode {
	case LegacyModeDisabled:
		return LegacyModeDisabled
	default:
		return LegacyModeWarn
	}
}

func recordLegacyEndpointHit(endpointFamily, replacement string, sunsetAt time.Time, mode string, merchantID string) {
	if endpointFamily == "" {
		return
	}
	if merchantID == "" {
		merchantID = "unknown"
	}

	now := time.Now().UTC()
	mode = NormalizeLegacyMode(mode)

	legacyEndpointObservabilityStore.mu.Lock()
	defer legacyEndpointObservabilityStore.mu.Unlock()

	entry, ok := legacyEndpointObservabilityStore.entries[endpointFamily]
	if !ok {
		entry = &legacyEndpointObservation{
			Replacement:  replacement,
			SunsetAt:     sunsetAt,
			Mode:         mode,
			MerchantHits: map[string]int64{},
		}
		legacyEndpointObservabilityStore.entries[endpointFamily] = entry
	}

	entry.Replacement = replacement
	entry.SunsetAt = sunsetAt
	entry.Mode = mode
	entry.TotalHits++
	entry.LastSeenAt = &now
	entry.MerchantHits[merchantID]++
}

func GetLegacyEndpointObservabilitySnapshot() LegacyEndpointObservabilitySnapshot {
	legacyEndpointObservabilityStore.mu.RLock()
	defer legacyEndpointObservabilityStore.mu.RUnlock()

	out := LegacyEndpointObservabilitySnapshot{
		GeneratedAt: time.Now().UTC(),
		Endpoints:   make([]LegacyEndpointObservabilityEntry, 0, len(legacyEndpointObservabilityStore.entries)),
	}

	for endpointFamily, entry := range legacyEndpointObservabilityStore.entries {
		merchantHits := make(map[string]int64, len(entry.MerchantHits))
		for merchantID, hits := range entry.MerchantHits {
			merchantHits[merchantID] = hits
		}
		out.Endpoints = append(out.Endpoints, LegacyEndpointObservabilityEntry{
			EndpointFamily: endpointFamily,
			Replacement:    entry.Replacement,
			SunsetAt:       entry.SunsetAt,
			Mode:           entry.Mode,
			TotalHits:      entry.TotalHits,
			LastSeenAt:     entry.LastSeenAt,
			MerchantHits:   merchantHits,
		})
		out.Summary.TrackedEndpoints++
		out.Summary.TotalHits += entry.TotalHits
		if entry.LastSeenAt != nil && (out.Summary.LastSeenAt == nil || entry.LastSeenAt.After(*out.Summary.LastSeenAt)) {
			lastSeen := *entry.LastSeenAt
			out.Summary.LastSeenAt = &lastSeen
		}
	}

	slices.SortFunc(out.Endpoints, func(a, b LegacyEndpointObservabilityEntry) int {
		if a.TotalHits == b.TotalHits {
			if a.EndpointFamily < b.EndpointFamily {
				return -1
			}
			if a.EndpointFamily > b.EndpointFamily {
				return 1
			}
			return 0
		}
		if a.TotalHits > b.TotalHits {
			return -1
		}
		return 1
	})

	return out
}

func resetLegacyEndpointObservabilityForTests() {
	legacyEndpointObservabilityStore.mu.Lock()
	defer legacyEndpointObservabilityStore.mu.Unlock()
	legacyEndpointObservabilityStore.entries = map[string]*legacyEndpointObservation{}
}

func ResetLegacyEndpointObservabilityForTests() {
	resetLegacyEndpointObservabilityForTests()
}
