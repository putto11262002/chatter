package token

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
	Payload interface{}
	jwt.RegisteredClaims
}

func newClaim(payload interface{}, exp time.Time) *AuthClaims {
	return &AuthClaims{
		Payload: payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "chatter",
		},
	}
}

func New(payload interface{}, expiration time.Duration, secret []byte) (string, time.Time, error) {
	exp := time.Now().Add(expiration)
	claims := newClaim(payload, exp)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(secret)
	if err != nil {
		return signed, exp, err
	}

	return signed, exp, err
}

func Verify(token string, claims *AuthClaims, secret []byte) error {

	_token, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

	switch {
	case _token.Valid:
		return nil
	case errors.Is(err, jwt.ErrTokenMalformed):
		return ErrTokenInvalid
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return ErrTokenInvalid
	case errors.Is(err, jwt.ErrTokenExpired):
		return ErrTokenExpired
	default:
		return ErrUnrecognizedToken
	}
}
