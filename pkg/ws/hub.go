package ws

import (
	"context"
)

const (
	ChatMessage MessageType = iota
)

// MessageType is the type of the message.
// 0-50 is reserved for system messages.
// 50-100 can be used to defined user-level messages.
type MessageType uint8

type StoreAapter interface {
	GetRoomMembers(string) ([]string, error)
	PersistMessage(Message) error
}

type HubStore interface {
	GetRoomMembers(context.Context, string) ([]string, error)
	PersistMessage(context.Context, Message) error
}

type Client interface {
	ID() string
	Send(Message)
	Close() error
}

type Hub interface {
	Register(client Client)
	Unregister(client Client)
	Broadcast(message Message)
	Close() error
	Start()
}

type HubMessageType = uint8

const (
	ChatMessageType HubMessageType = iota
	EventMessageType
	ErrorMessageType
)

type HubMessage struct {
	Type HubMessageType
	From string
	Data []byte
}

type Message struct {
	Type MessageType
	Data string
	// To is the room id the message is sent to.
	// If it is empty, the message is broadcast to all clients apart from the sender.
	To   string
	From string
}
