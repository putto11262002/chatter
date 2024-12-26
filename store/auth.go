package store

import (
	"context"
	"errors"
	"time"

	"github.com/putto11262002/chatter/models"
)

var (
	ErrBadCredentials  = errors.New("invalid credentials")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnauthorized    = errors.New("unauthorized")
)

type AuthStore interface {
	NewSession(ctx context.Context, username, password string) (token string, exp time.Time, user *models.UserWithoutSecrets, err error)
	DestroySession(ctx context.Context, session models.Session) error
	Session(ctx context.Context, token string) (payload *models.Session, err error)
}
