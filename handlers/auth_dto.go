package handlers

import (
	"time"

	"github.com/putto11262002/chatter/models"
)

type SigninPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SigninResponse struct {
	Token    string                     `json:"token"`
	ExpireAt time.Time                  `json:"expireAt"`
	User     *models.UserWithoutSecrets `json:"user"`
}

func NewSigninResponse(token string, exp time.Time, u *models.UserWithoutSecrets) *SigninResponse {
	return &SigninResponse{
		Token:    token,
		ExpireAt: exp,
		User:     u,
	}
}
