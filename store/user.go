package store

import (
	"context"
	"errors"

	"github.com/putto11262002/chatter/models"
)

var (
	ErrConflictedUser = errors.New("user already exists")
)

type GetUsersOptions struct {
	limit  int
	offset int
	q      string
}

type UserStore interface {
	CreateUser(ctx context.Context, user models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.UserWithoutSecrets, error)
	GetUsersByUsernames(ctx context.Context, usernames ...string) ([]models.UserWithoutSecrets, error)
	ComparePassword(ctx context.Context, username, password string) (bool, error)
	GetUsers(ctx context.Context, opts *GetUsersOptions) ([]models.UserWithoutSecrets, error)
}
