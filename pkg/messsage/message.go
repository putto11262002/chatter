package messsage

import "time"

type Message struct {
	Sender  string
	RoomID  string
	Content string
	SendAt  time.Time
}

type MessageStore interface {
	CreateMessage(Message) error
	GetMessages(roomID string) ([]Message, error)
}

type SQLITEMessageStore struct {
}
