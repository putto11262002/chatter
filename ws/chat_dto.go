package ws

import "time"

const (
	Message     = "message"
	ReadMessage = "read_message"
	Online      = "online"
	Offline     = "offline"
	Typing      = "typing"
)

type MessageData struct {
	ID     int       `json:"id"`
	RoomID string    `json:"room_id"`
	Type   int       `json:"type"`
	Data   string    `json:"data"`
	Sender string    `json:"sender"`
	SentAt time.Time `json:"sent_at"`
}

type ReadMessageData struct {
	RoomID          string    `json:"room_id"`
	ReadAt          time.Time `json:"read_at"`
	ReadBy          string    `json:"read_by"`
	LastReadMessage int       `json:"last_read_message"`
}

type TypingData struct {
	Typing   bool   `json:"typing"`
	Username string `json:"username"`
	RoomID   string `json:"room_id"`
}
