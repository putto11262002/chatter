package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/putto11262002/chatter/models"
	"github.com/putto11262002/chatter/pkg/token"
)

type SQLiteAuthStore struct {
	tokenExp  time.Duration
	secret    []byte
	userStore UserStore
	db        *sql.DB
}

type AuthOptions func(*SQLiteAuthStore)

func WithTokenExp(exp time.Duration) AuthOptions {
	return func(a *SQLiteAuthStore) {
		a.tokenExp = exp
	}
}

func NewSQLiteAuthStore(db *sql.DB, userStore UserStore, secret []byte, opts ...AuthOptions) *SQLiteAuthStore {
	auth := &SQLiteAuthStore{
		tokenExp:  time.Hour * 24,
		secret:    secret,
		userStore: userStore,
		db:        db,
	}
	for _, opt := range opts {
		opt(auth)
	}
	return auth
}

func (a *SQLiteAuthStore) NewSession(ctx context.Context, username, password string) (string, time.Time, *models.UserWithoutSecrets, error) {
	user, err := a.userStore.GetUserByUsername(ctx, username)
	if err != nil {
		return "", time.Time{}, nil, fmt.Errorf("get user by username: %w", err)
	}
	if user == nil {
		return "", time.Time{}, nil, ErrBadCredentials
	}

	ok, err := a.userStore.ComparePassword(ctx, username, password)
	if err != nil {
		return "", time.Time{}, nil, fmt.Errorf("compare password: %w", err)

	}

	if !ok {
		return "", time.Time{}, nil, ErrBadCredentials
	}

	t, exp, err := token.New(user, a.tokenExp, a.secret)

	if err != nil {
		return "", exp, nil, fmt.Errorf("creating token: %w", err)
	}

	if err := a.unblacklistToken(ctx, t); err != nil {
		return "", exp, nil, fmt.Errorf("unblacklisting token: %w", err)
	}

	return t, exp, user, nil
}

func (a *SQLiteAuthStore) DestroySession(ctx context.Context, session models.Session) error {
	if err := a.blacklistToken(ctx, session.Token); err != nil {
		return fmt.Errorf("blacklisting token: %w", err)
	}

	return nil
}

func (a *SQLiteAuthStore) unblacklistToken(ctx context.Context, token string) error {
	_, err := a.db.ExecContext(ctx, "DELETE FROM blacklists WHERE token = @token", sql.Named("token", token))
	if err != nil {
		return err
	}
	return nil
}

func (a *SQLiteAuthStore) blacklistToken(ctx context.Context, token string) error {
	_, err := a.db.ExecContext(ctx, "INSERT INTO blacklists (token) VALUES (@token)", sql.Named("token", token))
	if err != nil {
		return err
	}
	return nil
}

func (a *SQLiteAuthStore) isBlacklisted(ctx context.Context, token string) (bool, error) {
	row := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blacklists WHERE token = @token", sql.Named("token", token))
	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("scanning count: %w", err)
	}
	return count > 0, nil
}

func (a *SQLiteAuthStore) Session(ctx context.Context, t string) (session *models.Session, err error) {
	claims, err := token.Verify(t, a.secret)
	if err != nil {
		if errors.Is(err, token.ErrTokenExpired) || errors.Is(err, token.ErrTokenInvalid) {
			return nil, ErrUnauthenticated
		}
		return nil, fmt.Errorf("verifying token: %w", err)
	}

	isBlacklisted, err := a.isBlacklisted(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("checking blacklist: %w", err)
	}

	if isBlacklisted {
		return nil, ErrUnauthenticated
	}

	p, ok := claims.Payload.(models.UserWithoutSecrets)
	if !ok {
		return nil, errors.New("invalid token payload")
	}

	session = &models.Session{
		Username: p.Username,
		Token:    t,
	}

	return session, nil
}
