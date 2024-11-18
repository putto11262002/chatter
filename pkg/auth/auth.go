package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"example.com/go-chat/pkg/user"
)

var (
	ErrBadCredentials  = errors.New("invalid credentials")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnauthorized    = errors.New("unauthorized")
)

type Auth interface {
	NewSession(ctx context.Context, username, password string) (token string, exp time.Time, err error)
	DestroySession(ctx context.Context, token string) error
	Session(ctx context.Context, token string) (payload *Session, err error)
}

type Session struct {
	Username string
}

type SimpleAuth struct {
	tokenOptions TokenOptions
	userStore    user.UserStore
	db           *sql.DB
}

func NewSimpleAuth(userStore user.UserStore, db *sql.DB, tokenOptions TokenOptions) *SimpleAuth {
	return &SimpleAuth{
		tokenOptions: tokenOptions,
		userStore:    userStore,
		db:           db,
	}
}

func (a *SimpleAuth) NewSession(ctx context.Context, username, password string) (token string, exp time.Time, err error) {
	user, err := a.userStore.GetUserByUsername(ctx, username)
	if err != nil {
		return "", exp, fmt.Errorf("get user by username: %w", err)
	}
	if user == nil {
		return "", exp, ErrBadCredentials
	}

	ok, err := a.userStore.ComparePassword(ctx, username, password)
	if err != nil {
		return "", exp, fmt.Errorf("compare password: %w", err)

	}

	if !ok {
		return "", exp, ErrBadCredentials
	}

	token, exp, err = createToken(*user, a.tokenOptions)

	if err != nil {
		return "", exp, fmt.Errorf("creating token: %w", err)
	}

	if err := a.unblacklistToken(ctx, token); err != nil {
		return "", exp, fmt.Errorf("unblacklisting token: %w", err)
	}

	return token, exp, nil
}

func (a *SimpleAuth) DestroySession(ctx context.Context, token string) error {
	session, err := a.Session(ctx, token)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil
	}

	if err := a.blacklistToken(ctx, token); err != nil {
		return fmt.Errorf("blacklisting token: %w", err)
	}

	return nil
}

func (a *SimpleAuth) unblacklistToken(ctx context.Context, token string) error {
	_, err := a.db.ExecContext(ctx, "DELETE FROM blacklists WHERE token = @token", sql.Named("token", token))
	if err != nil {
		return err
	}
	return nil
}

func (a *SimpleAuth) blacklistToken(ctx context.Context, token string) error {
	_, err := a.db.ExecContext(ctx, "INSERT INTO blacklists (token) VALUES (@token)", sql.Named("token", token))
	if err != nil {
		return err
	}
	return nil
}

func (a *SimpleAuth) isBlacklisted(ctx context.Context, token string) (bool, error) {
	row := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blacklists WHERE token = @token", sql.Named("token", token))
	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("scanning count: %w", err)
	}
	return count > 0, nil
}

func (a *SimpleAuth) Session(ctx context.Context, token string) (session *Session, err error) {
	claims, err := verfiyToken(token, a.tokenOptions)
	if err != nil {
		if errors.Is(err, errTokenExpired) || errors.Is(err, errTokenInvalid) {
			return nil, ErrUnauthenticated
		}
		return nil, fmt.Errorf("verifying token: %w", err)
	}

	isBlacklisted, err := a.isBlacklisted(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("checking blacklist: %w", err)
	}

	if isBlacklisted {
		return nil, ErrUnauthenticated
	}

	session = &Session{
		Username: claims.Username,
	}

	return session, nil
}
