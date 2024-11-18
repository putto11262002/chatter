package auth

import (
	"errors"
	"time"

	"example.com/go-chat/pkg/user"
	"github.com/golang-jwt/jwt/v5"
)

var (
	errTokenExpired      = errors.New("token expired")
	errTokenInvalid      = errors.New("token invalid")
	errUnrecognizedToken = errors.New("unrecognized token")
)

type TokenOptions struct {
	Exp    time.Duration
	Secret []byte
}

type UserClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewUserClaims(username string, exp time.Time) *UserClaims {
	return &UserClaims{
		username,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "chatter",
		},
	}
}

func createToken(user user.UserWithoutSecrets, options TokenOptions) (signed string, exp time.Time, err error) {
	exp = time.Now().Add(options.Exp)
	claims := NewUserClaims(user.Username, exp)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err = token.SignedString(options.Secret)
	if err != nil {
		return signed, exp, err
	}

	claims, err = verfiyToken(signed, options)

	return signed, exp, err
}

func verfiyToken(token string, options TokenOptions) (*UserClaims, error) {
	claims := &UserClaims{}

	_token, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return options.Secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

	switch {
	case _token.Valid:
		return claims, nil
	case errors.Is(err, jwt.ErrTokenMalformed):
		return nil, errTokenInvalid
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return nil, errTokenInvalid
	case errors.Is(err, jwt.ErrTokenExpired):
		return nil, errTokenExpired
	default:
		return nil, errUnrecognizedToken
	}
}
