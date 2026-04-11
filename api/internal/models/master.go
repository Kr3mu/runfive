// Package models provides request DTOs for master/owner-only admin endpoints.
package models

// DiscordAuthentication is the request body for saving Discord OAuth credentials.
// The owner configures these via the Master Actions panel so that Discord login
// can be enabled for all users.
type DiscordAuthentication struct {
	// ClientId is the Discord application's OAuth2 client ID.
	ClientId string `json:"clientId" validate:"required"`
	// ClientSecret is the Discord application's OAuth2 client secret.
	ClientSecret string `json:"clientSecret" validate:"required"`
}
