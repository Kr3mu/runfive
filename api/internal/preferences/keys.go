// Package preferences defines the whitelist of per-user preference keys
// accepted by the /v1/preferences/:key endpoints.
//
// Adding a new preference: append one entry to AllowedKeys. The value
// format (JSON, base62 code, raw string) is the caller's responsibility —
// the backend only stores opaque TEXT up to MaxValueBytes.
package preferences

// MaxValueBytes is the hard cap on a single preference value.
// Comfortably fits a packed dashboard layout (≤ 22 chars) and small JSON
// blobs without leaving room for abuse.
const MaxValueBytes = 4096

// AllowedKeys is the set of preference keys the API accepts.
var AllowedKeys = map[string]bool{
	"dashboard-layout": true,
}

// IsAllowed reports whether the given key is a known preference.
func IsAllowed(key string) bool {
	return AllowedKeys[key]
}
