package chat

import (
	"context"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	// ErrInvalidUser is returned when a user is not found or is invalid.
	ErrInvalidUser = errors.New("invalid user")
	// ErrConflictedRoom is returned when a private chat room already exists between two users.
	ErrConflictedRoom = errors.New("chat already exists")
	// ErrInvalidRoom is returned when a chat room is not found for
	ErrInvalidRoom = errors.New("invalid room")
	// ErrInvalidMessage is returned when a message is invalid.
	ErrInvalidMessage = errors.New("invalid message")
	// ErrInvalidMessageType is returned when the type of the message is not supported.
	ErrInvalidMessageType = errors.New("invalid message type")
)

var validate = validator.New()

type ChatStore interface {
	// CreatePrivateChat create a private room between two users.
	// If a private room between the two users already exists, it should return ErrConflictedRoom.
	// If one of the users does not exist, it returns ErrInvalidUser.
	// If the two users are the same, it returns ErrInvalidUser.
	// If the error is nil, it returns the ID of the created room.
	CreatePrivateChat(ctx context.Context, users [2]string) (string, error)

	// CreateGroupChat creates a group chat room with the given name and users.
	// If the one of the users does not exist, it returns ErrInvalidUser.
	// If the number of users is less than 2, it returns ErrInvalidUser.
	// If there are duplicate users, it is deduplicated.
	// If the error is nil, it returns the ID of the created room.
	CreateGroupChat(ctx context.Context, name string, users ...string) (string, error)

	// GetRoomByID returns the room with the given ID.
	// If the room is not found, it returns nil.
	GetRoomByID(ctx context.Context, roomID string) (*Room, error)

	// GetRoomSummaries returns a list of room summaries for the given user.
	// The rooms are ordered by the last message sent time to the room then room name.
	// Reading offset and limit can be specified to paginate the results.
	// If the limit is a zero value, the limit is set to 100.
	// A nil slice is returned if there are no rooms.
	GetRoomSummaries(ctx context.Context, user string, offset, limit int) ([]RoomSummary, error)

	// SendMessageToRoom sends a message to the room.
	// If the user is not a member of the room, it returns ErrInvalidRoom.
	// If the message type is not supported, it returns ErrInvaidMessageType.
	// If the message is invalid, it returns ErrInvalidMessage.
	// The validity of the message is determined by the MessageCreateInput.Validate method.
	// Sender's last read message will be set to the message ID. This assumes that the sender
	// has read all previous messages in the room.
	SendMessageToRoom(ctx context.Context, message MessageCreateInput) (*Message, error)

	// GetRoomMessages returns a list of messages in the room.
	// Reading offset and limit can be specified to paginate the results.
	// If the limit is a zero value, the limit is set to 100.
	GetRoomMessages(ctx context.Context, roomID string, offset, limit int) ([]Message, error)

	// IsRoomMember returns true if the user is a member of the room.
	IsRoomMember(ctx context.Context, roomID, user string) (bool, error)

	// ReadRoomMessages marks the messages in the room as read up to a message.
	// It returns the message ID of the last message read and the time the messages were read.
	ReadRoomMessages(ctx context.Context, roomID, user string) (int, time.Time, error)

	// GetRoomMembers returns a list of members in the room.
	// If the room is not found, it returns nil.
	GetRoomMembers(ctx context.Context, roomID string) ([]RoomMember, error)

	// GetFriends returns a list of friends for the given user.
	// A friend is a user that has a private chat room with the user
	// or is a member of the same group chat room.
	GetFriends(ctx context.Context, username string) ([]string, error)
}

// MessageCreateInput represents the input for creating a message.
type MessageCreateInput struct {
	Type   MessageType `json:"type" validate:"required"`
	Data   string      `json:"data" validate:"required"`
	Sender string      `json:"sender" validate:"required"`
	RoomID string      `json:"roomID" validate:"required"`
}

// Validate validates the message input.
func (m *MessageCreateInput) Validate() error {
	return validate.Struct(m)
}
