package core

import (
	"context"
	"errors"
)

type User struct {
	Name     string `json:"name" validate:"required,min=3"`
	Username string `json:"username" validate:"required,min=3"`
	Password string `json:"password" validate:"required,min=8"`
}

func (u User) Validate() error {
	return validate.Struct(u)
}

type UserWithoutSecrets struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

var (
	ErrConflictedUser = errors.New("user already exists")
)

type GetUsersOptions struct {
	limit  int
	offset int
	q      string
}

type UserStore interface {
	CreateUser(ctx context.Context, user User) error

	GetUserByUsername(ctx context.Context, username string) (*UserWithoutSecrets, error)

	GetUsersByUsernames(ctx context.Context, usernames ...string) ([]UserWithoutSecrets, error)

	ComparePassword(ctx context.Context, username, password string) (bool, error)

	GetUsers(ctx context.Context, opts *GetUsersOptions) ([]UserWithoutSecrets, error)
}
