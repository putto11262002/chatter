package core

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired      = errors.New("token expired")
	ErrTokenInvalid      = errors.New("token invalid")
	ErrUnrecognizedToken = errors.New("unrecognized token")
)

type AuthClaims struct {
	Username string
	jwt.RegisteredClaims
}

func NewClaim(user UserWithoutSecrets, exp time.Time) *AuthClaims {
	return &AuthClaims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "chatter",
		},
	}
}

func NewToken(user UserWithoutSecrets, expiration time.Duration, secret []byte) (string, time.Time, error) {
	exp := time.Now().Add(expiration)
	claims := NewClaim(user, exp)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(secret)
	if err != nil {
		return signed, exp, err
	}

	return signed, exp, err
}

func VerifyToken(token string, secret []byte) (*AuthClaims, error) {

	claims := &AuthClaims{}
	_token, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

	switch {
	case _token.Valid:
		return claims, nil
	case errors.Is(err, jwt.ErrTokenMalformed):
		return nil, ErrTokenInvalid
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return nil, ErrTokenInvalid
	case errors.Is(err, jwt.ErrTokenExpired):
		return nil, ErrTokenExpired
	default:
		return nil, ErrUnrecognizedToken
	}
}
