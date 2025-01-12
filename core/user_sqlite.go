package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type SQLiteUserStore struct {
	db *sql.DB
}

func NewSqlieUserStore(db *sql.DB) *SQLiteUserStore {
	return &SQLiteUserStore{
		db: db,
	}
}

func (s *SQLiteUserStore) CreateUser(ctx context.Context, user User) error {
	eu, err := s.GetUserByUsername(ctx, user.Username)
	if err != nil {
		return fmt.Errorf("checking if user exists: %w", err)
	}

	if eu != nil {
		return ErrConflictedUser
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO users (name, username, password) VALUES (@name , @username, @password)",
		sql.Named("name", user.Name), sql.Named("username", user.Username), sql.Named("password", fmt.Sprintf("%s", hashed)))
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	return nil
}

func (s *SQLiteUserStore) GetUsersByUsernames(ctx context.Context, usernames ...string) ([]UserWithoutSecrets, error) {
	if len(usernames) == 0 {
		return nil, nil
	}

	values := make([]interface{}, 0, len(usernames))

	for _, username := range usernames {
		values = append(values, username)
	}

	rows, err := s.db.QueryContext(ctx, "SELECT name, username FROM users WHERE username IN ("+strings.Repeat("?,", len(usernames)-1)+"?)", values...)

	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}
	defer rows.Close()

	var users []UserWithoutSecrets

	for rows.Next() {
		user := UserWithoutSecrets{}
		if err := rows.Scan(&user.Name, &user.Username); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *SQLiteUserStore) GetUserByUsername(ctx context.Context, username string) (*UserWithoutSecrets, error) {

	row := s.db.QueryRowContext(ctx, "SELECT name, username FROM users WHERE username = ? LIMIT 1", username)

	user := new(UserWithoutSecrets)

	err := row.Scan(
		&user.Name,
		&user.Username,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("scanning user: %w", err)

	}

	return user, nil
}

func (s *SQLiteUserStore) ComparePassword(ctx context.Context, username, password string) (bool, error) {
	row := s.db.QueryRowContext(ctx, "SELECT password FROM users WHERE username = ? LIMIT 1", username)

	var storedPassword string

	err := row.Scan(&storedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("user not found")
		}

		return false, fmt.Errorf("scanning password: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
		return false, nil
	}

	return true, nil
}

func (s *SQLiteUserStore) GetUsers(ctx context.Context, options *GetUsersOptions) ([]UserWithoutSecrets, error) {
	template := make([]string, 0)
	values := make([]interface{}, 0)
	if options != nil {
		if options.q != "" {
			template = append(template, "username LIKE @q")
			values = append(values, sql.Named("q", "%"+options.q))
		}

		template = append(template, "LIMIT @limit")
		if options.limit > 0 {
			values = append(values, sql.Named("limit", options.limit))
		} else {
			values = append(values, sql.Named("limit", 10))
		}

		template = append(template, "OFFSET @offset")
		if options.offset > 0 {
			values = append(values, sql.Named("offset", options.offset))
		} else {
			values = append(values, sql.Named("offset", 0))
		}
	}

	row, err := s.db.QueryContext(ctx, "SELECT name, username FROM users WHERE "+strings.Join(template, " AND "), values...)
	if err != nil {
		return nil, fmt.Errorf("querying users: %w", err)
	}
	defer row.Close()

	users := []UserWithoutSecrets{}

	for row.Next() {
		var user UserWithoutSecrets
		if err := row.Scan(&user.Name, &user.Username); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break
			}
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
