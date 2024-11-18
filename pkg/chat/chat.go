package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"example.com/go-chat/pkg/user"
	"github.com/google/uuid"
)

const (
	PrivateChat ChatType = iota
	GroupChat
)

const (
	_ MessageType = iota
	TextMessage
)

var (
	ErrInvalidUser    = errors.New("invalid user")
	ErrConflictedChat = errors.New("chat already exists")
	ErrChatNotFound   = errors.New("chat not found")
	ErrInvalidMessage = errors.New("invalid message")
)

type ChatType = int

type MessageType = int

type RoomUser struct {
	Username string
	RoomID   string
	RoomName string
}

type Room struct {
	ID    string
	Users []RoomUser
	Type  ChatType
}

type Message struct {
	ID     string
	Type   MessageType
	Data   string
	RoomID string
	Sender string
	SentAt string
}

type MessageCreateInput struct {
	Type MessageType
	Data string
}

type ChatStore interface {
	CreatePrivateChat(ctx context.Context, users [2]string) (string, error)

	CreateGroupChat(ctx context.Context, name string, users ...string) (string, error)

	GetRoomRoomByID(ctx context.Context, roomID string) (*Room, error)

	GetUserRooms(ctx context.Context, username string) ([]RoomUser, error)

	SendMessageToRoom(ctx context.Context, message Message) (string, error)

	GetRoomMessages(ctx context.Context, roomID, user string) ([]Message, error)
}

type SQLiteChatStore struct {
	db        *sql.DB
	userStore user.UserStore
}

func NewSQLiteChatStore(db *sql.DB, userStore user.UserStore) *SQLiteChatStore {
	return &SQLiteChatStore{
		db:        db,
		userStore: userStore,
	}
}

type getUserByUsernameResult struct {
	user *user.UserWithoutSecrets
	err  error
}

func (s *SQLiteChatStore) CreatePrivateChat(ctx context.Context, users [2]string) (id string, err error) {

	if users[0] == users[1] {
		return "", ErrInvalidUser
	}

	userCh := make(chan *user.UserWithoutSecrets, 2)
	errCh := make(chan error, 2)
	wg := &sync.WaitGroup{}

	for _, u := range users {
		wg.Add(1)
		go func(u string) {
			user, err := s.userStore.GetUserByUsername(ctx, u)
			if err != nil {
				errCh <- err
				return
			}
			userCh <- user

		}(u)
	}

	// Close channels once all goroutines are finished
	go func() {
		wg.Wait()
		close(userCh)
		close(errCh)
	}()

	// Read from channels
	for i := 0; i < 2; i++ {
		select {
		case user, ok := <-userCh:
			if !ok {
				break
			}
			if user == nil {
				return "", ErrInvalidUser
			}
		case err, ok := <-errCh:
			if ok {
				return "", fmt.Errorf("getting user: %w", err)
			}
		}
	}

	// Continue with the rest of the logic
	row := s.db.QueryRowContext(ctx, `SELECT count(*) FROM room_users AS ru1 
		INNER JOIN room_users AS ru2 ON ru1.room_id = ru2.room_id 
		WHERE ru1.username = @username1 AND ru2.username = @username2`, sql.Named("username1", users[0]), sql.Named("username2", users[1]))

	var count int

	if err := row.Scan(&count); err != nil {
		return "", fmt.Errorf("scanning count: %w", err)
	}

	if count > 0 {
		return "", ErrConflictedChat
	}

	id = uuid.New().String()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("beginning transaction: %w", err)
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO rooms (id, type) VALUES 
		(@id, @type)`, sql.Named("id", id), sql.Named("type", PrivateChat))

	if err != nil {
		return "", fmt.Errorf("inserting room: %w", err)
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO room_users (room_id, room_name, username) VALUES 
		(@room_id, @room_name1, @username1),
		(@room_id, @room_name2, @username2)`,
		sql.Named("room_id", id),
		sql.Named("room_name1", users[1]), sql.Named("username1", users[0]),
		sql.Named("room_name2", users[0]), sql.Named("username2", users[1]))

	if err != nil {
		return "", fmt.Errorf("inserting room_users: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("committing transaction: %w", err)
	}

	return id, nil
}

func (s *SQLiteChatStore) GetRoomRoomByID(ctx context.Context, roomID string) (*Room, error) {

	row, err := s.db.QueryContext(ctx, `SELECT r.id, r.type, ru.room_name,  ru.username 
	FROM rooms AS r 
	INNER JOIN room_users AS ru ON r.id = ru.room_id 
	WHERE r.id = @id`, sql.Named("id", roomID))

	if err != nil {
		return nil, fmt.Errorf("querying room: %w", err)
	}

	var room Room
	room.Users = make([]RoomUser, 0, 2)
	var scaned int

	for row.Next() {
		scaned++
		var user RoomUser
		if err := row.Scan(&room.ID, &room.Type, &user.RoomName, &user.Username); err != nil {
			break
		}
		user.RoomID = roomID
		room.Users = append(room.Users, user)
	}

	if scaned == 0 {
		return nil, nil
	}

	if err := row.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return &room, nil
}

func (s *SQLiteChatStore) GetUserRooms(ctx context.Context, username string) ([]RoomUser, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ru.room_id, ru.room_name, ru.username 
    FROM room_users AS ru
    WHERE ru.username = @username`, sql.Named("username", username))
	defer rows.Close()
	if err != nil {
		return nil, fmt.Errorf("querying rooms: %w", err)
	}

	var rus []RoomUser

	for rows.Next() {
		var ru RoomUser
		if err := rows.Scan(&ru.RoomID, &ru.RoomName, &ru.Username); err != nil {
			break
		}
		rus = append(rus, ru)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []RoomUser{}, nil
		}
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return rus, nil

}

func (s *SQLiteChatStore) userInRoom(ctx context.Context, roomID, sender string) (bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT count(*) FROM room_users AS ru WHERE ru.room_id = @room_id AND ru.username = @username`,
		sql.Named("room_id", roomID), sql.Named("username", sender))

	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("scanning count: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteChatStore) SendMessageToRoom(ctx context.Context, message Message) (string, error) {
	ok, err := s.userInRoom(ctx, message.RoomID, message.Sender)
	if err != nil {
		return "", fmt.Errorf("userInRoom: %w", err)
	}

	if !ok {
		return "", ErrChatNotFound
	}

	id := uuid.New().String()

	if message.Type != TextMessage {
		return "", ErrInvalidMessage
	}

	s.db.ExecContext(ctx, `INSERT INTO messages (id, type, room_id, sender, data) VALUES (@id, @type, @room_id, @sender, @data)`,
		sql.Named("id", id), sql.Named("type", message.Type),
		sql.Named("room_id", message.RoomID), sql.Named("sender", message.Sender), sql.Named("data", message.Data))

	return id, nil
}

func (s *SQLiteChatStore) GetRoomMessages(ctx context.Context, roomID, user string) ([]Message, error) {
	ok, err := s.userInRoom(ctx, roomID, user)
	if err != nil {
		return nil, fmt.Errorf("userInRoom: %w", err)
	}

	if !ok {
		return nil, ErrChatNotFound
	}

	rows, err := s.db.QueryContext(ctx, `
	SELECT m.id, m.type, m.data, m.room_id, m.sender, m.sent_at
	FROM messages AS m
	WHERE m.room_id = @room_id
	ORDER BY m.sent_at DESC
	`, sql.Named("room_id", roomID))

	if err != nil {
		return nil, fmt.Errorf("querying messages: %w", err)
	}

	var messages []Message

	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.Type, &message.Data, &message.RoomID, &message.Sender, &message.SentAt); err != nil {
			break
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return messages, nil
		}
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return messages, nil
}

func (s *SQLiteChatStore) CreateGroupChat(ctx context.Context, name string, users ...string) (string, error) {
	panic("not implemented")
}
