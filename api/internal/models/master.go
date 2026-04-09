package models

type DiscordAuthentication struct {
	ClientId     string `json:"clientid" validate:"required"`
	ClientSecret string `json:"clientsecret" validate:"required"`
}
