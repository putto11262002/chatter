package core

import (
	"context"
	"errors"
	"time"
)

type Session struct {
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

var (
	ErrBadCredentials  = errors.New("invalid credentials")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnauthorized    = errors.New("unauthorized")
)

type AuthStore interface {
	NewSession(ctx context.Context, username, password string) (sesion *Session, err error)

	DestroySession(ctx context.Context, session Session) error

	Session(ctx context.Context, token string) (payload *Session, err error)
}

type HttpAuth struct {
}
