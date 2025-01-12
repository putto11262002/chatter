package models

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

type MemberRole string

const (
	Owner  MemberRole = "owner"
	Admin  MemberRole = "admin"
	Member MemberRole = "member"
)

// RoomUser represents a user in a chat room.
// It is used to store additional information about the room that is specific to the user.
type RoomMember struct {
	Role            MemberRole `json:"role"`
	Username        string     `json:"username"`
	RoomID          string     `json:"room_id"`
	LastMessageRead int        `json:"last_message_read"`
}

// Room represents a chat room.
type Room struct {
	ID                string       `json:"id"`
	Members           []RoomMember `json:"users"`
	Name              string       `json:"name"`
	LastMessageSentAt time.Time    `json:"last_message_sent_at"`
	LastMessageSent   int          `json:"last_message_sent"`
}

// RoomSummary represents a summary of a chat room from the perspective of a member.
type RoomSummary struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Members         []string `json:"members"`
	LastMessageRead int      `json:"last_message_read"`
}

// Message represents a chat message sent by a user to a room.
type Message struct {
	ID int `json:"id"`
	// Type is used to determined how the message data should be interpreted.
	Type   MessageType `json:"type"`
	Data   string      `json:"data"`
	RoomID string      `json:"room_id"`
	Sender string      `json:"sender"`
	SentAt time.Time   `json:"sent_at"`
}
