package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type AuthFixture struct {
	*BaseFixture
	userStore UserStore
	authStore AuthStore
}

func NewAuthFixture(t *testing.T) *AuthFixture {
	base := NewBaseFixture(t)

	userStore := NewSqlieUserStore(base.db)

	authStore := NewSQLiteAuthStore(base.db, userStore, secret)

	return &AuthFixture{
		userStore:   userStore,
		authStore:   authStore,
		BaseFixture: base,
	}
}

var secret = []byte("c2VjcmV0")

var user = User{
	Username: "username",
	Password: "password",
	Name:     "User",
}

func TestNewSession(t *testing.T) {
	t.Run("user does not exist", func(t *testing.T) {
		f := NewAuthFixture(t)
		defer f.tearDown()
		session, err := f.authStore.NewSession(f.ctx, "random", "random")
		require.Nil(t, session)
		require.NotNil(t, err)
		assert.Equal(t, ErrBadCredentials, err)
	})

	t.Run("invalid password", func(t *testing.T) {
		f := NewAuthFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, user)

		session, err := f.authStore.NewSession(
			f.ctx, user.Username, user.Password+"69")
		require.Nil(t, session)
		require.NotNil(t, err)
		assert.Equal(t, ErrBadCredentials, err)
	})

	t.Run("successfully create new session", func(t *testing.T) {
		f := NewAuthFixture(t)
		defer f.tearDown()
		seedUsers(f.ctx, f.t, f.userStore, user)

		session, err := f.authStore.NewSession(f.ctx, user.Username, user.Password)
		require.Nil(t, err)
		require.NotNil(t, session)
		assert.Greater(t, session.ExpiresAt, time.Now())
		assert.Equal(t, user.Username, session.Username)
		require.NotEmpty(t, session.Token)
		claims, err := VerifyToken(session.Token, secret)
		assert.Nil(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, user.Username, claims.Username)
	})
}

func TestSession(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		f := NewAuthFixture(t)
		seedUsers(f.ctx, t, f.userStore, user)
		defer f.tearDown()
		token, exp, err := NewToken(
			UserWithoutSecrets{Username: user.Username}, time.Hour, secret)
		require.Nil(t, err)
		require.True(t, time.Now().Before(exp))

		session, err := f.authStore.Session(f.ctx, token)
		require.Nil(t, err)
		require.NotNil(t, session)
		assert.Equal(t, user.Username, session.Username)
	})

	t.Run("blacklisted token", func(t *testing.T) {
		f := NewAuthFixture(t)
		seedUsers(f.ctx, t, f.userStore, user)
		defer f.tearDown()
		token, _, err := NewToken(
			UserWithoutSecrets{Username: user.Username}, time.Hour, secret)
		err = f.authStore.(*SQLiteAuthStore).blacklistToken(f.ctx, token)
		require.Nil(t, err)

		session, err := f.authStore.Session(f.ctx, token)
		require.NotNil(t, err)
		require.Nil(t, session)
		assert.Equal(t, ErrUnauthenticated, err)
	})

	t.Run("invalid token", func(t *testing.T) {
		f := NewAuthFixture(t)
		seedUsers(f.ctx, t, f.userStore, user)
		defer f.tearDown()
		token, exp, err := NewToken(
			UserWithoutSecrets{Username: user.Username}, -time.Hour, secret)
		require.Nil(t, err)
		require.NotZero(t, token)
		require.True(t, exp.Before(time.Now()))

		session, err := f.authStore.Session(f.ctx, token)
		require.NotNil(t, err)
		require.Nil(t, session)
		assert.Equal(t, ErrUnauthenticated, err)
	})
}

func TestDestroySession(t *testing.T) {
	f := NewAuthFixture(t)
	defer f.tearDown()
	seedUsers(f.ctx, t, f.userStore, user)

	session, err := f.authStore.NewSession(f.ctx, user.Username, user.Password)
	require.Nil(t, err)
	require.NotNil(t, session)

	session, err = f.authStore.Session(f.ctx, session.Token)
	require.Nil(t, err)
	require.NotNil(t, session)

	err = f.authStore.DestroySession(f.ctx, *session)
	require.Nil(t, err)

	session, err = f.authStore.Session(f.ctx, session.Token)
	require.Nil(t, session)
	require.NotNil(t, err)
	assert.Equal(t, ErrUnauthenticated, err)
}
