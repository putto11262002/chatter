package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/putto11262002/chatter/pkg/user"
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
	Username        string
	RoomID          string
	RoomName        string
	LastMessageRead int
}

type Room struct {
	ID    string
	Users []RoomUser
	Type  ChatType
}

type RoomView struct {
	RoomID          string   `json:"roomID"`
	RoomName        string   `json:"roomName"`
	Users           []string `json:"users"`
	LastMessageRead int      `json:"lastMessageRead"`
}

type MessageStatus = uint8

type Message struct {
	ID           int                  `json:"id"`
	Type         MessageType          `json:"type"`
	Data         string               `json:"data"`
	RoomID       string               `json:"roomID"`
	Sender       string               `json:"sender"`
	SentAt       time.Time            `json:"sentAt"`
	Status       MessageStatus        `json:"status"`
	Interactions []MessageInteraction `json:"interactions"`
}

type MessageInteraction struct {
	MessageID int       `json:"id"`
	Username  string    `json:"username"`
	ReadAt    time.Time `json:"readAt"`
}

type MessageCreateInput struct {
	Type   MessageType `json:"type"`
	Data   string      `json:"data"`
	Sender string      `json:"sender"`
	RoomID string      `json:"roomID"`
}

type ChatStore interface {
	CreatePrivateChat(ctx context.Context, users [2]string) (string, error)

	CreateGroupChat(ctx context.Context, name string, users ...string) (string, error)

	GetRoomRoomByID(ctx context.Context, roomID string) (*Room, error)

	GetUserRooms(ctx context.Context, username string) ([]RoomUser, error)

	SendMessageToRoom(ctx context.Context, message MessageCreateInput) (*Message, error)

	GetRoomMessages(ctx context.Context, roomID, user string) ([]Message, error)

	ReadRoomMessages(ctx context.Context, roomID, user string) (int, time.Time, error)

	GetRoomUsers(ctx context.Context, roomID string, user string) ([]RoomUser, error)

	GetFriends(ctx context.Context, username string) ([]string, error)

	GetRoomViews(ctx context.Context, user string) ([]RoomView, error)
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

	u, err := s.userStore.GetUsersByUsernames(ctx, users[0], users[1])
	if err != nil {
		return "", fmt.Errorf("GetUsersByUsernames: %w", err)
	}

	if len(u) != 2 {
		return "", ErrInvalidUser
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

func (s *SQLiteChatStore) CreateGroupChat(ctx context.Context, name string, users ...string) (string, error) {
	uniqueUsers := make(map[string]struct{}, len(users))
	for _, u := range users {
		uniqueUsers[u] = struct{}{}
	}

	if len(uniqueUsers) < 2 {
		return "", ErrInvalidUser
	}

	// check if these users exist

	us, err := s.userStore.GetUsersByUsernames(ctx, users...)
	if err != nil {
		return "", fmt.Errorf("GetUsersByUsernames: %w", err)
	}

	if len(us) != len(uniqueUsers) {
		return "", ErrInvalidUser
	}

	id := uuid.New().String()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("BeginTx: %w", err)
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO rooms (id, type) VALUES
	(@id, @type)`, sql.Named("id", id), sql.Named("type", GroupChat))

	if err != nil {
		return "", fmt.Errorf("ExecContext(insert room): %w", err)
	}

	valuesTeml := make([]string, 0, len(users))
	values := make([]interface{}, 0, len(users)+2)
	values = append(values,
		sql.Named("room_id", id),
		sql.Named("room_name", name),
		sql.Named("last_message_read", -1))
	for i, u := range users {
		valuesTeml = append(valuesTeml,
			fmt.Sprintf("(@room_id, @room_name, @username%d, @last_message_read)", i))
		values = append(values, sql.Named(fmt.Sprintf("username%d", i), u))
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO room_users (room_id, room_name, username, last_message_read) VALUES `+strings.Join(valuesTeml, ","),
		values...)

	if err != nil {
		return "", fmt.Errorf("ExecContext(insert room_users): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("Commit: %w", err)
	}

	return id, nil
}

func (s *SQLiteChatStore) GetRoomRoomByID(ctx context.Context, roomID string) (*Room, error) {

	row, err := s.db.QueryContext(ctx, `SELECT r.id, r.type, ru.room_name, ru.username, ru.last_message_read
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
		if err := row.Scan(&room.ID, &room.Type, &user.RoomName,
			&user.Username, &user.LastMessageRead); err != nil {
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

func (s *SQLiteChatStore) SendMessageToRoom(ctx context.Context, message MessageCreateInput) (*Message, error) {
	ok, err := s.userInRoom(ctx, message.RoomID, message.Sender)
	if err != nil {
		return nil, fmt.Errorf("userInRoom: %w", err)
	}

	if !ok {
		return nil, ErrChatNotFound
	}

	sentAt := time.Now().UTC()

	if message.Type != TextMessage {
		return nil, ErrInvalidMessage
	}

	row := s.db.QueryRowContext(ctx, `INSERT INTO messages (type, room_id, sender, data) 
	VALUES ( @type, @room_id, @sender, @data) RETURNING id`,
		sql.Named("type", message.Type),
		sql.Named("room_id", message.RoomID), sql.Named("sender", message.Sender),
		sql.Named("data", message.Data), sql.Named("sent_at", sentAt))

	var id int

	if err := row.Scan(&id); err != nil {
		return nil, fmt.Errorf("row.Scan: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `UPDATE room_users SET last_message_read = @last_message_read WHERE room_id = @room_id AND username = @username`,
		sql.Named("last_message_read", id), sql.Named("room_id", message.RoomID), sql.Named("username", message.Sender))
	if err != nil {
		return nil, fmt.Errorf("ExecContext(update room_users): %w", err)
	}

	createdMessage := &Message{
		ID:           id,
		Type:         message.Type,
		Data:         message.Data,
		RoomID:       message.RoomID,
		Sender:       message.Sender,
		SentAt:       sentAt,
		Interactions: []MessageInteraction{},
	}

	return createdMessage, nil
}

func (s *SQLiteChatStore) GetRoomViews(ctx context.Context, user string) ([]RoomView, error) {
	query := `
	WITH my_rooms AS (
	    SELECT ru.room_id, ru.room_name FROM room_users AS ru WHERE ru.username = @username
	) SELECT my_rooms.room_id, my_rooms.room_name, ru.username, ru.last_message_read FROM my_rooms 
	INNER JOIN room_users AS ru ON my_rooms.room_id = ru.room_id
	`

	rows, err := s.db.QueryContext(ctx, query, sql.Named("username", user))
	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}
	defer rows.Close()

	var roomViews []RoomView
	roomUserMap := make(map[string]*RoomView)

	var (
		roomID          string
		roomName        string
		username        string
		lastMessageRead sql.NullInt64
	)

	for rows.Next() {
		if err := rows.Scan(&roomID, &roomName, &username, &lastMessageRead); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)

		}

		var roomView *RoomView
		var exists bool
		roomView, exists = roomUserMap[roomID]
		var lmr int
		if lastMessageRead.Valid {
			lmr = int(lastMessageRead.Int64)
		}
		if !exists {
			rv := RoomView{
				RoomID:          roomID,
				RoomName:        roomName,
				Users:           []string{},
				LastMessageRead: lmr,
			}

			roomViews = append(roomViews, rv)
			roomView = &roomViews[len(roomViews)-1]
			roomUserMap[roomID] = roomView
		}

		roomView.Users = append(roomView.Users, username)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return roomViews, nil
}

func (s *SQLiteChatStore) GetRoomMessages(ctx context.Context, roomID, user string) ([]Message, error) {
	// Check if the user is part of the room
	ok, err := s.userInRoom(ctx, roomID, user)
	if err != nil {
		return nil, fmt.Errorf("userInRoom: %w", err)
	}
	if !ok {
		return nil, ErrChatNotFound
	}

	// Query messages with interactions
	rows, err := s.db.QueryContext(ctx, `
	SELECT m.id, m.type, m.data, m.room_id, m.sender, m.sent_at,  mi.read_at, mi.username
	FROM (SELECT id, type, data, room_id, sender, sent_at FROM messages 
	WHERE room_id = @room_id ORDER BY sent_at DESC LIMIT @limit OFFSET @offset) AS m
	LEFT JOIN message_interactions AS mi ON m.id = mi.message_id
	`, sql.Named("room_id", roomID), sql.Named("offset", 0), sql.Named("limit", 100))
	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}
	defer rows.Close()

	// Process rows
	messageMap := make(map[int]*Message) // Map to track messages by ID
	var messages []Message

	for rows.Next() {
		var (
			messageID   int
			messageType MessageType
			messageData string
			roomID      string
			sender      string
			sentAt      time.Time
			readAt      sql.NullString
			username    sql.NullString
		)

		// Scan row
		if err := rows.Scan(&messageID, &messageType, &messageData, &roomID, &sender, &sentAt, &readAt, &username); err != nil {
			// Return nil, nil for sql.ErrNoRows
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}

		// Find or create the message
		message, exists := messageMap[messageID]
		if !exists {
			messages = append(messages, Message{
				ID:           messageID,
				Type:         messageType,
				Data:         messageData,
				RoomID:       roomID,
				Sender:       sender,
				SentAt:       sentAt,
				Interactions: []MessageInteraction{},
			})
			message = &messages[len(messages)-1]

			messageMap[messageID] = message
		}

		// Append interaction if available
		if readAt.Valid && username.Valid {
			parsedReadAt, err := time.Parse(time.RFC3339, readAt.String)
			if err != nil {
				return nil, fmt.Errorf("invalid readAt format: %w", err)
			}

			message.Interactions = append(message.Interactions, MessageInteraction{
				Username:  username.String,
				ReadAt:    parsedReadAt,
				MessageID: messageID,
			})

		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return messages, nil
}

func (s *SQLiteChatStore) ReadRoomMessages(ctx context.Context, roomID, user string) (int, time.Time, error) {
	ok, err := s.userInRoom(ctx, roomID, user)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("userInRoom: %w", err)
	}

	if !ok {
		return 0, time.Time{}, ErrChatNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("BeginTx: %w", err)
	}
	defer tx.Rollback()

	readAt := time.Now()

	// Get the lastest message before or equal to the readAt time
	query := `
	SELECT id FROM messages WHERE room_id = @room_id AND sent_at <= @read_at ORDER BY sent_at DESC LIMIT 1`

	rows, err := tx.QueryContext(ctx, query,
		sql.Named("username", user),
		sql.Named("room_id", roomID),
		sql.Named("read_at", readAt.Format(time.RFC3339)),
	)

	if err != nil {
		return 0, time.Time{}, fmt.Errorf("QueryContext(insert message_interactions): %w", err)
	}

	var lastMessageRead int
	for rows.Next() {
		if err := rows.Scan(&lastMessageRead); err != nil {
			break
		}
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, time.Time{}, nil
		}
		return 0, time.Time{}, fmt.Errorf("rows.Err: %w", err)
	}

	query = `
	UPDATE room_users
	SET last_message_read = @last_message_read
	WHERE username = @username AND room_id = @room_id
	`
	_, err = tx.ExecContext(ctx, query,
		sql.Named("last_message_read", lastMessageRead),
		sql.Named("username", user), sql.Named("room_id", roomID))
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("ExecContext(update room_users): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, time.Time{}, fmt.Errorf("Commit: %w", err)
	}

	return lastMessageRead, readAt, nil
}

func (s *SQLiteChatStore) GetRoomUsers(ctx context.Context, roomID string, user string) ([]RoomUser, error) {

	ok, err := s.userInRoom(ctx, roomID, user)
	if err != nil {
		return nil, fmt.Errorf("userInRoom: %w", err)
	}

	if !ok {
		return nil, ErrChatNotFound
	}

	rows, err := s.db.QueryContext(ctx, `SELECT room_id, room_name, username FROM room_users WHERE room_id = @room_id`, sql.Named("room_id", roomID))
	if err != nil {
		return nil, fmt.Errorf("QueryContext(select room_users): %w", err)
	}

	var roomUsers []RoomUser
	for rows.Next() {
		var roomUser RoomUser
		if err := rows.Scan(&roomUser.RoomID, &roomUser.RoomName, &roomUser.Username); err != nil {
			break
		}

		roomUsers = append(roomUsers, roomUser)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("rows.Scan: %w", err)
	}

	return roomUsers, nil

}

// GetConnectedUsers returns a list of friends of the user.
// A friend is a user that has a private chat or are part of a group chat with the user.
func (s *SQLiteChatStore) GetFriends(ctx context.Context, username string) ([]string, error) {
	query := `
	WITH user_rooms AS (
		SELECT room_id
		FROM room_users
		WHERE username = ?
	),
	friends_in_rooms AS (
		SELECT DISTINCT ru.username AS friend
		FROM room_users ru
		INNER JOIN user_rooms ur ON ru.room_id = ur.room_id
		WHERE ru.username != ?
	)
	SELECT DISTINCT friend
	FROM friends_in_rooms
	ORDER BY friend;
	`

	rows, err := s.db.QueryContext(ctx, query, username, username)
	if err != nil {
		if err == sql.ErrNoRows {
			// No friends found
			return nil, nil
		}
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var friends []string
	for rows.Next() {
		var friend string
		if err := rows.Scan(&friend); err != nil {
			if err == sql.ErrNoRows {
				// No rows in this iteration, return empty result
				return nil, nil
			}
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		friends = append(friends, friend)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return friends, nil
}
