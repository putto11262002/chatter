package chat

import (
	"time"
)

const (
	// PrivateChat is a chat room where only two users can participate.
	// Only one private chat room can exist between two users.
	PrivateChat ChatType = iota
	// GroupChat is a chat room where multiple users can participate.
	// A group chat must have two or more users in it.
	GroupChat
)

const (
	_ MessageType = iota
	// TextMessage indicates that the message is a text message.
	// The Data field of the message should be interpreted as a UTF-8 encoded string.
	TextMessage
)

// ChatType represents the type of a chat room.
type ChatType = int

// MessageType represents the type of a chat message.
// It is used to determine how the message data should be interpreted.
type MessageType = int

// RoomUser represents a user in a chat room.
// It is used to store additional information about the room that is specific to the user.
type RoomMember struct {
	Username        string
	RoomID          string
	RoomName        string
	LastMessageRead int
}

// Room represents a chat room.
type Room struct {
	ID                string
	Members           []RoomMember
	Type              ChatType
	LastMessageSentAt time.Time
	LastMessageSent   int
}

// RoomSummary represents a summary of a chat room from the perspective of a member.
type RoomSummary struct {
	ID              string   `json:"roomID"`
	Name            string   `json:"roomName"`
	Members         []string `json:"users"`
	LastMessageRead int      `json:"lastMessageRead"`
}

// Message represents a chat message sent by a user to a room.
type Message struct {
	ID int `json:"id"`
	// Type is used to determined how the message data should be interpreted.
	Type   MessageType `json:"type"`
	Data   string      `json:"data"`
	RoomID string      `json:"roomID"`
	Sender string      `json:"sender"`
	SentAt time.Time   `json:"sentAt"`
}
