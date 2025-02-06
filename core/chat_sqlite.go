package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
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

func (s *SQLiteChatStore) CreateRoom(ctx context.Context, name string, ownerUsername string) (string, error) {

	owner, err := s.userStore.GetUserByUsername(ctx, ownerUsername)
	if err != nil {
		return "", fmt.Errorf("GetUsersByUsernames: %w", err)
	}
	if owner == nil {
		return "", ErrInvalidUser
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("BeginTx: %w", err)
	}
	id := uuid.New().String()
	defer tx.Rollback()

	query := `INSERT INTO rooms (id, name, last_message_sent_at, last_message_sent, last_message_sent_data)
	          VALUES (@id, @name, @last_message_sent_at, @last_message_sent, @last_message_sent_data)`
	_, err = tx.ExecContext(ctx, query,
		sql.Named("id", id), sql.Named("name", name),
		sql.Named("last_message_sent_at", time.Time{}),
		sql.Named("last_message_sent", 0),
		sql.Named("last_message_sent_data", ""),
	)
	if err != nil {
		return "", fmt.Errorf("ExecContext(insert room): %w", err)
	}

	query = `
		INSERT INTO room_members (room_id, username, role, last_message_read)
		VALUES (@room_id, @username, @role, @last_message_read)`
	_, err = tx.ExecContext(ctx, query,
		sql.Named("room_id", id), sql.Named("username", ownerUsername),
		sql.Named("role", Owner), sql.Named("last_message_read", 0))
	if err != nil {
		return "", fmt.Errorf("ExecContext(insert room_members): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("Commit: %w", err)
	}

	return id, nil
}

func (s *SQLiteChatStore) AddRoomMember(ctx context.Context, roomID, username string, role MemberRole) error {

	// check if user exist
	user, err := s.userStore.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("GetUserByUsername: %w", err)
	}
	if user == nil {
		return ErrInvalidUser
	}
	// check if room exist
	room, err := s.GetRoomByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("GetRoomByID: %w", err)
	}
	if room == nil {
		return ErrInvalidRoom
	}

	if role == Owner {
		return ErrDisAllowedOperation

	}

	query := `INSERT INTO room_members (room_id, username, role, last_message_read)
		VALUES (@room_id, @username, @role, @last_message_read) ON CONFLICT DO NOTHING`
	_, err = s.db.ExecContext(ctx, query,
		sql.Named("room_id", roomID), sql.Named("username", username),
		sql.Named("role", role), sql.Named("last_message_read", 0))
	if err != nil {
		return fmt.Errorf("ExecContext: %w", err)
	}
	return nil
}

func (s *SQLiteChatStore) RemoveRoomMember(ctx context.Context, roomID, username string) error {
	room, err := s.GetRoomByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("GetRoomByID: %w", err)
	}
	if room == nil {
		return ErrInvalidRoom
	}
	// check if the member is the owner
	ok, role, err := s.IsRoomMember(ctx, roomID, username)
	if err != nil {
		return fmt.Errorf("IsRoomMember: %w", err)
	}
	if !ok {
		return ErrInvalidMember
	}
	if role == Owner {
		return ErrDisAllowedOperation
	}
	query := `DELETE FROM room_members WHERE room_id = @room_id AND username = @username`
	_, err = s.db.ExecContext(ctx, query,
		sql.Named("room_id", roomID), sql.Named("username", username))
	if err != nil {
		return fmt.Errorf("ExecContext: %w", err)
	}
	return nil
}

func (s *SQLiteChatStore) GetRoomByID(ctx context.Context, roomID string) (*Room, error) {

	query := `
		SELECT r.id, r.name, r.last_message_sent_at, r.last_message_sent, last_message_sent_data,
		ru.username, ru.role, ru.last_message_read FROM rooms AS r 
		INNER JOIN room_members AS ru ON r.id = ru.room_id 
		WHERE r.id = @id`

	row, err := s.db.QueryContext(ctx, query, sql.Named("id", roomID))

	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}

	var id string
	var name string
	var lastMessageSentAt time.Time
	var lastMessageSent int
	var lastMessageSentData string

	members := make([]RoomMember, 0, 2)

	for row.Next() {
		var member RoomMember
		if err := row.Scan(
			&id, &name, &lastMessageSentAt,
			&lastMessageSent, &lastMessageSentData,
			&member.Username, &member.Role, &member.LastMessageRead,
		); err != nil {
			break
		}
		member.RoomID = id
		members = append(members, member)
	}

	if err := row.Err(); err != nil {
		return nil, fmt.Errorf("row.Scan: %w", err)
	}

	if id == "" {
		return nil, nil
	}

	room := Room{
		ID:                  id,
		Name:                name,
		LastMessageSentAt:   lastMessageSentAt,
		LastMessageSent:     lastMessageSent,
		LastMessageSentData: lastMessageSentData,
		Members:             members,
	}

	return &room, nil
}

func (s *SQLiteChatStore) IsRoomMember(ctx context.Context, roomID, user string) (bool, MemberRole, error) {
	query := `
	SELECT count(*), ru.role
	FROM room_members AS ru 
	WHERE ru.room_id = @room_id AND ru.username = @username`

	row := s.db.QueryRowContext(ctx, query,
		sql.Named("room_id", roomID), sql.Named("username", user))

	var count int
	var _role sql.NullString
	if err := row.Scan(&count, &_role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("scanning count: %w", err)
	}
	if count == 0 {
		return false, "", nil
	}
	if !_role.Valid {
		panic("role is null")
	}
	role := MemberRole(_role.String)

	return true, role, nil
}

func (s *SQLiteChatStore) GetUserRooms(ctx context.Context, user string, offset, litmit int) ([]Room, error) {
	// Select all the rooms that belong to the user order in desc of id
	// then join it with room_member

	query := `
	WITH r as (
	    SELECT r.id, r.name, r.last_message_sent_at, r.last_message_sent, r.last_message_sent_data
	    FROM room_members as rm
	    INNER JOIN rooms as r ON rm.room_id = r.id
	    WHERE rm.username = @username
	    ORDER BY r.last_message_sent_at DESC, r.name ASC
	    LIMIT @limit OFFSET @offset
	)
	SELECT r.id, r.name, r.last_message_sent_at, r.last_message_sent, r.last_message_sent_data,
	rm.username, rm.role,  rm.last_message_read
	FROM r 
	INNER JOIN room_members as rm
	ON r.id = rm.room_id
	ORDER BY r.last_message_sent_at DESC, r.name ASC
	`

	if litmit == 0 {
		litmit = 20
	}

	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.QueryContext(ctx, query,
		sql.Named("username", user), sql.Named("limit", litmit), sql.Named("offset", offset))
	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}

	roomMap := make(map[string]*Room)
	var (
		id, name, username, lastMessageSentData string
		lastMessageSentAt                       time.Time
		lastMessageSent, lastMessageRead        int
		role                                    MemberRole
	)
	for rows.Next() {
		if err := rows.Scan(&id, &name, &lastMessageSentAt,
			&lastMessageSent, &lastMessageSentData, &username, &role, &lastMessageRead); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}

		room, ok := roomMap[id]
		if !ok {
			room = &Room{
				ID:                  id,
				Name:                name,
				LastMessageSentAt:   lastMessageSentAt,
				LastMessageSent:     lastMessageSent,
				LastMessageSentData: lastMessageSentData,
			}
			roomMap[id] = room
		}

		member := RoomMember{
			Username:        username,
			Role:            role,
			RoomID:          id,
			LastMessageRead: lastMessageRead,
		}
		room.Members = append(room.Members, member)
	}

	rooms := make([]Room, 0, len(roomMap))

	for _, r := range roomMap {
		rooms = append(rooms, *r)
	}
	slices.SortFunc(rooms, func(i, j Room) int {
		lastMessageSentCmp := j.LastMessageSentAt.Compare(i.LastMessageSentAt)
		if lastMessageSentCmp != 0 {
			return lastMessageSentCmp
		}
		return strings.Compare(i.Name, j.Name)

	})

	return rooms, nil

}

func (s *SQLiteChatStore) GetRoomSummaries(ctx context.Context, user string, offset, limit int) ([]RoomSummary, error) {
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
	if limit == 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.QueryContext(ctx, query,
		sql.Named("username", user), sql.Named("limit", limit), sql.Named("offset", offset))
	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}

	var roomViews []RoomSummary
	roomUserMap := make(map[string]*RoomSummary)

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

		var roomSummary *RoomSummary
		var exists bool
		roomSummary, exists = roomUserMap[roomID]
		var lastMessageRead int
		if _lastMessageRead.Valid {
			lastMessageRead = int(_lastMessageRead.Int64)
		}
		if !exists {
			rv := RoomSummary{
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

func (s *SQLiteChatStore) SendMessageToRoom(ctx context.Context, message MessageCreateInput) (*Message, error) {
	err := message.Validate()
	if err != nil {
		return nil, ErrInvalidMessage
	}
	ok, _, err := s.IsRoomMember(ctx, message.RoomID, message.Sender)
	if err != nil {
		return nil, fmt.Errorf("IsUserInRoom: %w", err)
	}
	if !ok {
		return nil, ErrInvalidRoom
	}
	if message.Type != TextMessage {
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
	last_message_sent_at = @last_message_sent_at,
	last_message_sent_data = @last_message_sent_data
	WHERE id = @room_id
	`
	_, err = tx.ExecContext(ctx, query,
		sql.Named("room_id", message.RoomID),
		sql.Named("last_message_sent", id),
		sql.Named("last_message_sent_at", sentAt),
		sql.Named("last_message_sent_data", message.Data),
	)
	if err != nil {
		return nil, fmt.Errorf("ExectContext(update room): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Commit: %w", err)
	}

	createdMessage := &Message{
		ID:     id,
		Type:   message.Type,
		Data:   message.Data,
		RoomID: message.RoomID,
		Sender: message.Sender,
		SentAt: sentAt,
	}

	return createdMessage, nil
}

func (s *SQLiteChatStore) GetRoomMessages(ctx context.Context, roomID string, offset, limit int) ([]Message, error) {

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
	var messages []Message

	for rows.Next() {
		var message Message
		// Scan row
		if err := rows.Scan(&message.ID, &message.Type, &message.Data, &message.RoomID,
			&message.Sender, &message.SentAt); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		messages = append(messages, message)

	}

	slices.Reverse(messages)

	return messages, nil
}

func (s *SQLiteChatStore) ReadRoomMessages(ctx context.Context, roomID, user string) (int, time.Time, error) {
	ok, _, err := s.IsRoomMember(ctx, roomID, user)
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

func (s *SQLiteChatStore) GetRoomMembers(ctx context.Context, roomID string) ([]RoomMember, error) {

	query := `
	SELECT room_id, username, role
	FROM room_members 
	WHERE room_id = @room_id
	`

	rows, err := s.db.QueryContext(ctx, query, sql.Named("room_id", roomID))
	if err != nil {
		return nil, fmt.Errorf("QueryContext(select room_members): %w", err)
	}

	var members []RoomMember
	for rows.Next() {
		var member RoomMember
		if err := rows.Scan(&member.RoomID, &member.Username, &member.Role); err != nil {
			break
		}

		members = append(members, member)
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
