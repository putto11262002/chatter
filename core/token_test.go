package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken(t *testing.T) {
	secret := []byte("secret")
	user := UserWithoutSecrets{
		Username: "username",
		Name:     "User",
	}
	t.Run("valid token", func(t *testing.T) {
		before := time.Now()
		token, expiresAt, err := NewToken(user, time.Hour, secret)
		require.Nil(t, err)
		require.NotEmpty(t, token)
		require.True(t, expiresAt.After(before.Add(time.Hour)))
		// verify token
		claims, err := VerifyToken(token, secret)
		require.Nil(t, err)
		assert.Equal(t, user.Username, claims.Username)
	})
}
