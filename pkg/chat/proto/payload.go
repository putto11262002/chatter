package proto

import (
	"time"

	"github.com/putto11262002/chatter/pkg/chat"
)

type SendMessageRequestPayload struct {
	RoomID string           `json:"roomID"`
	Type   chat.MessageType `json:"type"`
	Data   string           `json:"data"`
}

type SendMessageResponsePayload struct {
	// Code indicates the status of the message delivery.
	Code   PacketCode `json:"code"`
	RoomID string     `json:"roomID"`
	// MessageID is the generated server for the message.
	// It can be used to reference the message in the future.
	MessageID int `json:"messageID"`
	// SentAt is the time the message was persisted in the database
	// which is the point that the delivery is considered successful
	SentAt time.Time `json:"sentAt"`
}

type ReadMessagePayload struct {
	RoomID string `json:"roomID"`
}

type BroadcastReadMessagePayload struct {
	RoomID    string    `json:"roomID"`
	MessageID int       `json:"messageID"`
	ReadAt    time.Time `json:"readAt"`
	Username  string    `json:"username"`
}

type BroadcastMessagePayload chat.Message

type TypingEventPayload struct {
	RoomID   string `json:"roomID"`
	Typing   bool   `json:"typing"`
	Username string `json:"username"`
}

type PresencePayload struct {
	Username string `json:"username"`
	Presence bool   `json:"presence"`
}
