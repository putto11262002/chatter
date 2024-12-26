package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/putto11262002/chatter/models"
)

type SQLiteChatStore struct {
	db        *sql.DB
	userStore UserStore
}

func NewSQLiteChatStore(db *sql.DB, userStore UserStore) *SQLiteChatStore {
	return &SQLiteChatStore{
		db:        db,
		userStore: userStore,
	}
}

func (s *SQLiteChatStore) CreatePrivateChat(ctx context.Context, users [2]string) (string, error) {

	if users[0] == users[1] {
		return "", ErrInvalidUser
	}

	u, err := s.userStore.GetUsersByUsernames(ctx, users[0], users[1])
	if err != nil {
		return "", fmt.Errorf("GetUserByUsernames: %w", err)
	}

	if len(u) != 2 {
		return "", ErrInvalidUser
	}

	query := `
	SELECT count(*) FROM room_members AS ru1 
	INNER JOIN room_members AS ru2 ON ru1.room_id = ru2.room_id INNER JOIN rooms AS r ON ru1.room_id = r.id
	WHERE ru1.username = @username1 AND ru2.username = @username2 AND r.type = @type
	`

	// Continue with the rest of the logic
	row := s.db.QueryRowContext(ctx, query,
		sql.Named("username1", users[0]), sql.Named("username2", users[1]),
		sql.Named("type", models.PrivateChat))

	var count int

	if err := row.Scan(&count); err != nil {
		return "", fmt.Errorf("row.Scan: %w", err)
	}

	if count > 0 {
		return "", ErrConflictedRoom
	}

	id := uuid.New().String()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("BeginTx: %w", err)
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO rooms (id, type, last_message_sent_at, last_message_sent) 
		VALUES (@id, @type, @last_message_sent_at, @last_message_sent)`,
		sql.Named("id", id), sql.Named("type", models.PrivateChat),
		sql.Named("last_message_sent_at", time.Time{}),
		sql.Named("last_message_sent", -1))

	if err != nil {
		return "", fmt.Errorf("ExecContext(insert rooms): %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO room_members (room_id, room_name, username) VALUES 
		(@room_id, @room_name1, @username1),
		(@room_id, @room_name2, @username2)`,
		sql.Named("room_id", id), sql.Named("room_name1", users[1]),
		sql.Named("username1", users[0]), sql.Named("room_name2", users[0]),
		sql.Named("username2", users[1]))

	if err != nil {
		return "", fmt.Errorf("ExectContext(insert room_members): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("Commit: %w", err)
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

	var i int
	for u, _ := range uniqueUsers {
		users[i] = u
		i++
	}

	// check if these users exist
	us, err := s.userStore.GetUsersByUsernames(ctx, users[:len(uniqueUsers)]...)
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

	_, err = tx.ExecContext(ctx,
		`INSERT INTO rooms (id, type, last_message_sent_at, last_message_sent) 
		VALUES (@id, @type, @last_message_sent_at, @last_message_sent)`,
		sql.Named("id", id), sql.Named("type", models.GroupChat),
		sql.Named("last_message_sent_at", time.Time{}),
		sql.Named("last_message_sent", -1))

	if err != nil {
		return "", fmt.Errorf("ExecContext(insert room): %w", err)
	}

	valuesTeml := make([]string, 0, len(us))
	values := make([]interface{}, 0, len(us)+2)
	values = append(values,
		sql.Named("room_id", id),
		sql.Named("room_name", name),
		sql.Named("last_message_read", -1))
	for i, u := range us {
		valuesTeml = append(valuesTeml,
			fmt.Sprintf("(@room_id, @room_name, @username%d, @last_message_read)", i))
		values = append(values, sql.Named(fmt.Sprintf("username%d", i), u.Username))
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO room_members (room_id, room_name, username, last_message_read) VALUES `+strings.Join(valuesTeml, ","),
		values...)

	if err != nil {
		return "", fmt.Errorf("ExecContext(insert room_members): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("Commit: %w", err)
	}

	return id, nil
}

func (s *SQLiteChatStore) GetRoomByID(ctx context.Context, roomID string) (*models.Room, error) {

	query := `
		SELECT r.id, r.type, r.last_message_sent_at, r.last_message_sent, ru.room_name, ru.username, ru.last_message_read FROM rooms AS r 
		INNER JOIN room_members AS ru ON r.id = ru.room_id 
		WHERE r.id = @id
    `

	row, err := s.db.QueryContext(ctx, query, sql.Named("id", roomID))

	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}

	var room models.Room
	room.Members = make([]models.RoomMember, 0, 2)

	for row.Next() {
		var member models.RoomMember
		if err := row.Scan(
			&room.ID, &room.Type, &room.LastMessageSentAt,
			&room.LastMessageSent, &member.RoomName,
			&member.Username, &member.LastMessageRead,
		); err != nil {
			break
		}
		member.RoomID = roomID
		room.Members = append(room.Members, member)
	}

	if err := row.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("row.Scan: %w", err)
	}

	return &room, nil
}

// func (s *SQLiteChatStore) GetRoomSummaries(ctx context.Context, user string) ([]RoomSummary, error) {
// 	query := `
// 		SELECT ru.room_id, ru.room_name, ru.username
// 		FROM room_members AS ru
// 		WHERE ru.username = @username
//     `
// 	rows, err := s.db.QueryContext(ctx, query, sql.Named("username", user))
// 	defer rows.Close()
// 	if err != nil {
// 		return nil, fmt.Errorf("querying rooms: %w", err)
// 	}
//
// 	var rus []RoomUser
//
// 	for rows.Next() {
// 		var ru RoomUser
// 		if err := rows.Scan(&ru.RoomID, &ru.RoomName, &ru.Username); err != nil {
// 			break
// 		}
// 		rus = append(rus, ru)
// 	}
//
// 	if err := rows.Err(); err != nil {
// 		if errors.Is(err, sql.ErrNoRows) {
// 			return []RoomUser{}, nil
// 		}
// 		return nil, fmt.Errorf("iterating rows: %w", err)
// 	}
//
// 	return rus, nil
//
// }

func (s *SQLiteChatStore) IsRoomMember(ctx context.Context, roomID, user string) (bool, error) {
	query := `
	SELECT count(*) 
	FROM room_members AS ru 
	WHERE ru.room_id = @room_id AND ru.username = @username`

	row := s.db.QueryRowContext(ctx, query,
		sql.Named("room_id", roomID), sql.Named("username", user))

	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("scanning count: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteChatStore) SendMessageToRoom(ctx context.Context, message MessageCreateInput) (*models.Message, error) {
	err := message.Validate()
	if err != nil {
		return nil, ErrInvalidMessage
	}
	ok, err := s.IsRoomMember(ctx, message.RoomID, message.Sender)
	if err != nil {
		return nil, fmt.Errorf("IsUserInRoom: %w", err)
	}
	if !ok {
		return nil, ErrInvalidRoom
	}
	if message.Type != models.TextMessage {
		return nil, ErrInvalidMessageType
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("BeginTx: %w", err)
	}
	defer tx.Rollback()

	sentAt := time.Now().UTC()
	query := `
	INSERT INTO messages (type, room_id, sender, data, sent_at) 
	VALUES ( @type, @room_id, @sender, @data, @sent_at) RETURNING id`
	row := tx.QueryRowContext(ctx, query,
		sql.Named("type", message.Type),
		sql.Named("room_id", message.RoomID), sql.Named("sender", message.Sender),
		sql.Named("data", message.Data), sql.Named("sent_at", sentAt))
	var id int
	if err := row.Scan(&id); err != nil {
		return nil, fmt.Errorf("row.Scan: %w", err)
	}

	query = `
	UPDATE room_members SET last_message_read = @last_message_read 
	WHERE room_id = @room_id AND username = @username
	`
	_, err = tx.ExecContext(ctx, query,
		sql.Named("last_message_read", id), sql.Named("room_id", message.RoomID),
		sql.Named("username", message.Sender))
	if err != nil {
		return nil, fmt.Errorf("ExecContext(update room_members): %w", err)
	}

	query = `
	UPDATE rooms SET 
	last_message_sent = @last_message_sent,
	last_message_sent_at = @last_message_sent_at
	WHERE id = @room_id
	`
	_, err = tx.ExecContext(ctx, query,
		sql.Named("room_id", message.RoomID),
		sql.Named("last_message_sent", id),
		sql.Named("last_message_sent_at", sentAt))
	if err != nil {
		return nil, fmt.Errorf("ExectContext(update room): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Commit: %w", err)
	}

	createdMessage := &models.Message{
		ID:     id,
		Type:   message.Type,
		Data:   message.Data,
		RoomID: message.RoomID,
		Sender: message.Sender,
		SentAt: sentAt,
	}

	return createdMessage, nil
}

func (s *SQLiteChatStore) GetRoomSummaries(ctx context.Context, user string, offset, limit int) ([]models.RoomSummary, error) {
	query := `
	WITH my_rooms AS 
	(SELECT ru.room_id, ru.room_name 
	FROM room_members AS ru 
	INNER JOIN rooms AS r ON ru.room_id = r.id
	WHERE ru.username = @username 
	ORDER BY r.last_message_sent_at DESC, ru.room_name ASC
	LIMIT @limit OFFSET @offset)
	SELECT my_rooms.room_id, my_rooms.room_name, ru.username, ru.last_message_read 
	FROM my_rooms 
	INNER JOIN room_members AS ru ON my_rooms.room_id = ru.room_id
	`

	rows, err := s.db.QueryContext(ctx, query,
		sql.Named("username", user), sql.Named("limit", limit), sql.Named("offset", offset))
	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}

	var roomViews []models.RoomSummary
	roomUserMap := make(map[string]*models.RoomSummary)

	var (
		roomID           string
		roomName         string
		username         string
		_lastMessageRead sql.NullInt64
	)

	for rows.Next() {
		if err := rows.Scan(&roomID, &roomName, &username, &_lastMessageRead); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)

		}

		var roomSummary *models.RoomSummary
		var exists bool
		roomSummary, exists = roomUserMap[roomID]
		var lastMessageRead int
		if _lastMessageRead.Valid {
			lastMessageRead = int(_lastMessageRead.Int64)
		}
		if !exists {
			rv := models.RoomSummary{
				ID:              roomID,
				Name:            roomName,
				Members:         []string{},
				LastMessageRead: lastMessageRead,
			}

			roomViews = append(roomViews, rv)
			roomSummary = &roomViews[len(roomViews)-1]
			roomUserMap[roomID] = roomSummary
		}

		roomSummary.Members = append(roomSummary.Members, username)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return roomViews, nil
}

func (s *SQLiteChatStore) GetRoomMessages(ctx context.Context, roomID string, offset, limit int) ([]models.Message, error) {

	query := `
	SELECT id, type, data, room_id, sender, sent_at 
	FROM messages 
	WHERE room_id = @room_id 
	ORDER BY id DESC
	LIMIT @limit OFFSET @offset
	`
	if limit == 0 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.QueryContext(ctx, query,
		sql.Named("room_id", roomID), sql.Named("offset", offset), sql.Named("limit", limit))
	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}

	// Process rows
	var messages []models.Message

	for rows.Next() {
		var message models.Message
		// Scan row
		if err := rows.Scan(&message.ID, &message.Type, &message.Data, &message.RoomID,
			&message.Sender, &message.SentAt); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		messages = append(messages, message)

	}

	return messages, nil
}

func (s *SQLiteChatStore) ReadRoomMessages(ctx context.Context, roomID, user string) (int, time.Time, error) {
	ok, err := s.IsRoomMember(ctx, roomID, user)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("userInRoom: %w", err)
	}

	if !ok {
		return 0, time.Time{}, ErrInvalidRoom
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("BeginTx: %w", err)
	}
	defer tx.Rollback()

	readAt := time.Now()

	// Get the lastest message before or equal to the readAt time
	query := `
	SELECT id 
	FROM messages 
	WHERE room_id = @room_id AND sent_at <= @read_at 
	ORDER BY sent_at DESC 
	LIMIT 1`

	row := tx.QueryRowContext(ctx, query,
		sql.Named("username", user),
		sql.Named("room_id", roomID),
		sql.Named("read_at", readAt.Format(time.RFC3339)),
	)

	var lastMessageRead int
	if err := row.Scan(&lastMessageRead); err != nil {
		// No message to read
		if errors.Is(err, sql.ErrNoRows) {
			return 0, time.Now(), nil
		}
		return 0, time.Time{}, fmt.Errorf("rows.Err: %w", err)
	}

	query = `
	UPDATE room_members
	SET last_message_read = @last_message_read
	WHERE username = @username AND room_id = @room_id
	`
	_, err = tx.ExecContext(ctx, query,
		sql.Named("last_message_read", lastMessageRead),
		sql.Named("username", user), sql.Named("room_id", roomID))
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("ExecContext(update room_members): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, time.Time{}, fmt.Errorf("Commit: %w", err)
	}

	return lastMessageRead, readAt, nil
}

func (s *SQLiteChatStore) GetRoomMembers(ctx context.Context, roomID string) ([]models.RoomMember, error) {

	query := `
	SELECT room_id, room_name, username 
	FROM room_members 
	WHERE room_id = @room_id
	`

	rows, err := s.db.QueryContext(ctx, query, sql.Named("room_id", roomID))
	if err != nil {
		return nil, fmt.Errorf("QueryContext(select room_members): %w", err)
	}

	var members []models.RoomMember
	for rows.Next() {
		var roomUser models.RoomMember
		if err := rows.Scan(&roomUser.RoomID, &roomUser.RoomName, &roomUser.Username); err != nil {
			break
		}

		members = append(members, roomUser)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("rows.Scan: %w", err)
	}

	return members, nil

}

// GetConnectedUsers returns a list of friends of the user.
// A friend is a user that has a private chat or are part of a group chat with the user.
func (s *SQLiteChatStore) GetFriends(ctx context.Context, username string) ([]string, error) {
	query := `
	WITH user_rooms AS (
		SELECT room_id
		FROM room_members
		WHERE username = ?
	),
	friends_in_rooms AS (
		SELECT DISTINCT ru.username AS friend
		FROM room_members ru
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
