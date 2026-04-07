package models

type LoginRequest struct {
	Email    string `json:"email" validate:"required,min=4,max=24"`
	Password string `json:"password" validate:"required, min=10"`
}
