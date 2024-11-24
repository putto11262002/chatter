package proto

import (
	"math/rand"
	"time"
)

type PacketType int

// Packet naming conventions:
// [name] | [type]? | packet |
// - name is the name of the packet
// - type is the type of the packet it can be request or response. If the message does not need a response it can be omitted.
//
// Request/response packets:
// Request/response packets are used for operations that needs a feedback from the server.
const (
	_ PacketType = iota
	// SendMessageRequestPacket is a packet sent by a client to send a message to a room.
	SendMessageRequestPacket
	// SendMessageResponsePacket is a packet sent by a server to respond to a SendMessageRequestPacket.
	SendMessageResponsePacket

	// BroadcastMessagePacket is a packet sent by a server to broadcast a message to other clients in a room
	// when a new message is sent to a room.
	BroadcastMessagePacket

	// ReadMessagePacket is a packet sent by a client to indicate that a user has read a message.
	ReadMessagePacket

	BroadcastReadMessagePacket

	// PresencePacket is a packet sent by a server to update the presence of a user in a room.
	PresencePacket

	// TypingEventPacket is a packet sent by a client to indicate changes in user's typing status.
	TypingEventPacket
)

type Packet struct {
	// CorrelationID is the id of the request that this packet is responding to.
	// It is used to match a response with a request.
	// This is only used for packet types that require a response.
	CorrelationID int `json:"correlationID"`
	// Type is the type of the packet.
	// It is used to determine the schema of the Payload
	Type PacketType `json:"type"`
	// Payload is the actual data of the packet.
	// The schema of the payload is determined by the type
	Payload []byte `json:"payload"`
	// SentAt is the time the packet was sent by the sender
	// It should not be trusted if the packet is sent by an untrusted sender
	SentAt time.Time `json:"sentAt"`
	// From is the ID of sender of the packet. This is usually set on the server.
	From string `json:"from"`
}

func NewPacket(correlationID int, pt PacketType, payload []byte, from string) *Packet {
	return &Packet{
		CorrelationID: correlationID,
		Type:          pt,
		Payload:       payload,
		From:          from,
		SentAt:        time.Now(),
	}

}

var correlationIDRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// NewCorrelationID generates a random correlation id.
func NewCorrelationID() int {
	return correlationIDRand.Int()
}

type PacketCode int

const (
	Success PacketCode = iota
	MessageTooLarge
	InvalidDestination
)
