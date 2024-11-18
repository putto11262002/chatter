package room

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type UserRoom struct {
	Username        string
	RoomID          string
	RoomName        string
	ReadLastMessage bool
}

type Room struct {
	ID    string
	Users []string
}

type RoomCreateInput struct {
	Name string
}

type RoomStore interface {
	GetRoom(id string) (Room, error)
	GetUserRooms(userId string) ([]Room, error)
	AddUserToRoom(roomId, userId string) error
	RemoveUserFromRoom(roomId, userId string) error
	CreateRoom(id string) (Room, error)
}

type SQLiteRoomStore struct {
	db *sql.DB
}

func (s *SQLiteRoomStore) CreateRoom(ctx context.Context, name string, ownerId string) (string, error) {

	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO rooms (id) VALUES (@name)`,
		sql.Named("name", input.Name))

	if err != nil {
		return "", fmt.Errorf("inserting room: %w", err)
	}

	return input.Name, nil
}

func (s *SQLiteRoomStore) AddUserToRoom(ctx context.Context, roomId, userId string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO room_users (room_id, user_id) VALUES (@room_id, @user_id)", sql.Named("room_id", roomId), sql.Named("user_id", userId))
	if err != nil {
		return fmt.Errorf("inserting room_users: %w", err)
	}
	return nil
}

func (s *SQLiteRoomStore) RemoveUserFromRoom(ctx context.Context, roomId, userId string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM room_users WHERE room_id = @room_id AND user_id = @user_id", sql.Named("room_id", roomId), sql.Named("user_id", userId))
	if err != nil {
		return fmt.Errorf("deleting room_users: %w", err)
	}
	return nil
}

func (s *SQLiteRoomStore) GetUserRooms(ctx context.Context, userId string) ([]UserRoom, error) {
}

func (s *SQLiteRoomStore) GetRoom(ctx context.Context, id string) (*Room, error) {
}
