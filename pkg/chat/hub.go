package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type PacketType uint8

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
	Send(*Packet)
	Close() error
}

type Hub interface {
	Register(client Client)
	Unregister(client Client)
	Broadcast(message *Packet)
	Close() error
	Start()
}

const (
	_ PacketType = iota
	ChatMessagePacket
	RoomEventPacket
	ErrorPacket
	ChatMessageStatusUpdatePacket
	ReadMessagePacket
	TypingEventPacket
)

type HubMessage struct {
	Type PacketType
	From string
	Data []byte
}

type ReadMessagePacketPayload struct {
	RoomID string `json:"roomID"`
	ReadBy string `json:"readBy"`
}

type Packet struct {
	Type PacketType `json:"type"`
	// base64 encoded string
	Data          []byte `json:"data"`
	From          string `json:"from"`
	CorrelationID uint16 `json:"correlationID"`
}

type ChatMessageStatusUpdatePacketData struct {
	MessageID string        `json:"messageID"`
	Status    MessageStatus `json:"status"`
	RoomID    string        `json:"roomID"`
}

type TypingEventPacketPayload struct {
	RoomID string `json:"roomID"`
	Typing bool   `json:"typing"`
	User   string `json:"user"`
}

// TODO: wrap base context with hub owns context so we can cancel it in close
// also for each transaction i may need to create a new timeout context
// and perhaps pass the context to client send. In the client.Send()
// unregister the client if the context times out

type ChatterHub struct {
	// clients contains all the connected clients
	clients map[string]Client

	register   chan Client
	unregister chan Client
	broadcast  chan *Packet
	baseCtx    context.Context
	store      ChatStore
	done       chan struct{}
	mu         sync.RWMutex
}

func NewChatterHub(ctx context.Context, store ChatStore) *ChatterHub {
	return &ChatterHub{
		register:   make(chan Client, 1),
		unregister: make(chan Client, 1),
		broadcast:  make(chan *Packet, 1),
		clients:    make(map[string]Client),
		baseCtx:    ctx,
		store:      store,
		done:       make(chan struct{}, 1),
	}
}

func (hub *ChatterHub) Register(client Client) {
	if hub == nil {
		return
	}
	hub.register <- client
}

func (hub *ChatterHub) Unregister(client Client) {
	if hub == nil {
		return
	}
	hub.unregister <- client
}

func (hub *ChatterHub) Broadcast(packet *Packet) {
	if hub == nil {
		return
	}
	hub.broadcast <- packet
}

func (hub *ChatterHub) Close() error {
	if hub == nil {
		return nil
	}
	hub.done <- struct{}{}

	for _, client := range hub.clients {
		client.Close()
	}

	return nil

}

func handleReadMessagePacket(ctx context.Context, hub *ChatterHub, packet *Packet) error {
	var payload ReadMessagePacketPayload
	err := json.Unmarshal(packet.Data, &payload)
	if err != nil {
		return fmt.Errorf("invalid message format")
	}

	err = hub.store.ReadRoomMessages(ctx, payload.RoomID, packet.From)
	if err != nil {
		return fmt.Errorf("store.ReadRoomMessages: %v", err)
	}

	for _, client := range hub.clients {
		if client.ID() == packet.From {
			continue
		}
		client.Send(packet)
	}

	return nil
}

func handleChatMessagePacket(ctx context.Context, hub *ChatterHub, packet *Packet) error {
	var message MessageCreateInput
	err := json.Unmarshal(packet.Data, &message)
	if err != nil {
		return fmt.Errorf("invalid message format")
	}
	message.Sender = packet.From

	created, err := hub.store.SendMessageToRoom(ctx, message)

	if err != nil {
		return fmt.Errorf("store.PersistMessage: %v", err)
	}

	room, err := hub.store.GetRoomRoomByID(ctx, message.RoomID)
	if err != nil {
		return fmt.Errorf("store.GetRoomMembers: %v", err)
	}

	jsonMessage, err := json.Marshal(created)
	if err != nil {
		return fmt.Errorf("message: json.Marshal: %v", err)
	}

	jsonAck, err := json.Marshal(&ChatMessageStatusUpdatePacketData{MessageID: created.ID, Status: created.Status, RoomID: created.RoomID})
	if err != nil {
		return fmt.Errorf("ack: json.Marshal: %v", err)
	}

	outPacket := &Packet{
		Type:          ChatMessagePacket,
		From:          message.Sender,
		Data:          jsonMessage,
		CorrelationID: packet.CorrelationID,
	}

	ackPacket := &Packet{
		Type:          ChatMessageStatusUpdatePacket,
		Data:          jsonAck,
		CorrelationID: packet.CorrelationID,
	}

	for _, u := range room.Users {
		client, ok := hub.clients[u.Username]
		if ok {

			if u.Username == message.Sender {
				client.Send(ackPacket)
			} else {
				client.Send(outPacket)
			}
		}
	}

	return nil

}

// TODO: check user in room
func handleTypingEventPacket(ctx context.Context, hub *ChatterHub, packet *Packet) error {
	var payload TypingEventPacketPayload
	err := json.Unmarshal(packet.Data, &payload)
	if err != nil {
		return fmt.Errorf("invalid message format")
	}
	room, err := hub.store.GetRoomRoomByID(ctx, payload.RoomID)
	if err != nil {
		return fmt.Errorf("store.GetRoomMembers: %v", err)
	}

	if room == nil {
		return fmt.Errorf("room not found")
	}

	for _, u := range room.Users {
		if u.Username == packet.From {
			continue
		}
		client, ok := hub.clients[u.Username]
		if ok {
			client.Send(packet)
		}
	}

	return nil
}

func (hub *ChatterHub) Start() {
	for {

		select {
		case client := <-hub.register:
			hub.clients[client.ID()] = client
			log.Printf("client registered: %v", client.ID())

		case client := <-hub.unregister:
			if c, ok := hub.clients[client.ID()]; ok {
				delete(hub.clients, client.ID())
				c.Close()
			}
			log.Printf("client unregistered: %v", client.ID())

		case packet := <-hub.broadcast:

			switch packet.Type {
			case ChatMessagePacket:
				if err := handleChatMessagePacket(hub.baseCtx, hub, packet); err != nil {
					log.Printf("handleChatMessagePacket: %v", err)
				}
			case ReadMessagePacket:
				if err := handleReadMessagePacket(hub.baseCtx, hub, packet); err != nil {
					log.Printf("handleReadMessagePacket: %v", err)
				}
			case TypingEventPacket:
				if err := handleTypingEventPacket(hub.baseCtx, hub, packet); err != nil {
					log.Printf("handleTypingEventPacket: %v", err)
				}
			case RoomEventPacket:
				log.Printf("RoomEventPacket not implemented")
			default:
				log.Printf("unknown packet type")
			}
			// if targetClients is nil, broadcast to all clients

		case <-hub.done:
			return

		}

	}
}
