package models

import "time"

// UserPreference stores a single per-user key/value preference.
//
// Value is an opaque TEXT blob whose format is decided by the caller
// (JSON, base62-packed layout code, raw string, etc.). The set of allowed
// keys is whitelisted in internal/preferences/keys.go.
type UserPreference struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null;uniqueIndex:idx_user_pref,priority:1"`
	Key       string `gorm:"not null;uniqueIndex:idx_user_pref,priority:2"`
	Value     string `gorm:"type:text;not null"`
	UpdatedAt time.Time
}

// PreferenceResponse is the JSON shape returned by GET /v1/preferences/:key.
type PreferenceResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PreferenceUpdateRequest is the body for PUT /v1/preferences/:key.
type PreferenceUpdateRequest struct {
	Value string `json:"value"`
}
