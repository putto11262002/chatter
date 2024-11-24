package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/putto11262002/chatter/pkg/chat"
	"github.com/putto11262002/chatter/pkg/chat/proto"
)

type PacketType uint8

type Client interface {
	ID() string
	Send(*proto.Packet)
	Close() error
}

type Hub interface {
	Register(client Client)
	Unregister(client Client)
	Broadcast(message *proto.Packet)
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

// TODO: wrap base context with hub owns context so we can cancel it in close
// also for each transaction i may need to create a new timeout context
// and perhaps pass the context to client send. In the client.Send()
// unregister the client if the context times out

type ChatterHub struct {
	// clients contains all the connected clients
	clients map[string]Client

	register   chan Client
	unregister chan Client
	request    chan *proto.Packet
	res        chan *proto.Packet
	baseCtx    context.Context
	store      chat.ChatStore
	done       chan struct{}
	mu         sync.RWMutex
}

func NewChatterHub(ctx context.Context, store chat.ChatStore) *ChatterHub {
	return &ChatterHub{
		register:   make(chan Client, 1),
		unregister: make(chan Client, 1),
		request:    make(chan *proto.Packet, 1),
		res:        make(chan *proto.Packet, 1),
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

func (hub *ChatterHub) Broadcast(packet *proto.Packet) {
	if hub == nil {
		return
	}
	hub.request <- packet
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

// handleReadMessagePacket update the database then broadcast the read event to other users in the room.
func handleReadMessagePacket(ctx context.Context, hub *ChatterHub, packet *proto.Packet) error {
	var p proto.ReadMessagePayload
	err := json.Unmarshal(packet.Payload, &p)
	if err != nil {
		return fmt.Errorf("invalid payload format")
	}

	lastMessageRead, readAt, err := hub.store.ReadRoomMessages(ctx, p.RoomID, packet.From)
	if err != nil {
		return fmt.Errorf("store.ReadRoomMessages: %v", err)
	}

	roomUsers, err := hub.store.GetRoomUsers(ctx, p.RoomID, packet.From)

	if err != nil {
		return fmt.Errorf("store.GetRoomUsers: %w", err)
	}

	broadcastReadMessagePayload := proto.BroadcastReadMessagePayload{
		RoomID:    p.RoomID,
		MessageID: lastMessageRead,
		Username:  packet.From,
		ReadAt:    readAt,
	}

	encodedBrmp, err := json.Marshal(&broadcastReadMessagePayload)
	if err != nil {
		return fmt.Errorf("Marshal(broadcastReadMessagePayload)): %w", broadcastReadMessagePayload, err)
	}

	brmpp := proto.NewPacket(
		packet.CorrelationID,
		proto.BroadcastReadMessagePacket,
		encodedBrmp,
		packet.From,
	)

	for _, u := range roomUsers {
		if u.Username == packet.From {
			continue
		}
		client, ok := hub.clients[u.Username]
		if ok {
			client.Send(brmpp)
		}
	}

	return nil
}

func handleRequestSendMessagePacket(ctx context.Context, hub *ChatterHub, packet *proto.Packet) error {
	var inPayload proto.SendMessageRequestPayload
	err := json.Unmarshal(packet.Payload, &inPayload)
	if err != nil {
		return fmt.Errorf("invalid payload format")
	}

	created, err := hub.store.SendMessageToRoom(ctx, chat.MessageCreateInput{
		Type:   inPayload.Type,
		Sender: packet.From,
		RoomID: inPayload.RoomID,
		Data:   inPayload.Data,
	})

	if err != nil {
		return fmt.Errorf("store.SendMessageToRoom: %w", err)
	}

	roomUsers, err := hub.store.GetRoomUsers(ctx, inPayload.RoomID, packet.From)
	if err != nil {
		return fmt.Errorf("store.GetRoomUsers: %w", err)
	}

	broadcastPayload := proto.BroadcastMessagePayload(*created)

	encodedBroadcastPayload, err := json.Marshal(&broadcastPayload)
	if err != nil {
		return fmt.Errorf("json.Marshal(broadcastPayload): %w", err)
	}

	resPayload := proto.SendMessageResponsePayload{
		Code:      proto.Success,
		RoomID:    created.RoomID,
		MessageID: created.ID,
		SentAt:    created.SentAt,
	}

	encodedResPayload, err := json.Marshal(&resPayload)
	if err != nil {
		return fmt.Errorf("json.Marshal(resPayload): %v", err)
	}

	broadcastPacket := proto.NewPacket(
		packet.CorrelationID,
		proto.BroadcastMessagePacket,
		encodedBroadcastPayload,
		packet.From)

	resPacket := proto.NewPacket(
		packet.CorrelationID,
		proto.SendMessageResponsePacket,
		encodedResPayload,
		"",
	)

	for _, u := range roomUsers {
		client, ok := hub.clients[u.Username]
		if ok {

			if u.Username == packet.From {
				client.Send(resPacket)
			} else {
				client.Send(broadcastPacket)
			}
		}
	}

	return nil

}

func handleTypingEventPacket(ctx context.Context, hub *ChatterHub, packet *proto.Packet) error {
	var inPayload proto.TypingEventPayload
	err := json.Unmarshal(packet.Payload, &inPayload)
	if err != nil {
		return fmt.Errorf("invalid payload format")
	}
	roomUsers, err := hub.store.GetRoomUsers(ctx, inPayload.RoomID, packet.From)
	if err != nil {
		return fmt.Errorf("store.GetRoomUsers: %v", err)
	}

	for _, u := range roomUsers {
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

func broadcastPresence(ctx context.Context, hub *ChatterHub, client Client, precense bool) error {
	friends, err := hub.store.GetFriends(ctx, client.ID())
	if err != nil {
		return fmt.Errorf("store.GetFriends: %v", err)
	}

	payload := proto.PresencePayload{
		Username: client.ID(),
		Presence: precense,
	}

	encodedPayload, err := json.Marshal(&payload)
	if err != nil {
		return fmt.Errorf("json.Marshal(presencePayload): %v", err)
	}

	packet := proto.NewPacket(
		proto.NewCorrelationID(),
		proto.PresencePacket,
		encodedPayload,
		client.ID(),
	)

	for _, f := range friends {
		c, ok := hub.clients[f]
		if ok {
			fmt.Printf("sending presence to %s\n", f)
			c.Send(packet)
		}
	}

	return nil

}

func asyncFriendsPresence(ctx context.Context, hub *ChatterHub, client Client) error {
	friends, err := hub.store.GetFriends(ctx, client.ID())
	if err != nil {
		return fmt.Errorf("store.GetFriends: %v", err)
	}
	fmt.Printf("friends: %v\n", friends)

	for _, f := range friends {
		if _, ok := hub.clients[f]; ok {
			payload := proto.PresencePayload{
				Username: client.ID(),
				Presence: true,
			}

			encodedPayload, err := json.Marshal(&payload)
			if err != nil {
				return fmt.Errorf("json.Marshal(presencePayload): %v", err)
			}

			packet := proto.NewPacket(
				proto.NewCorrelationID(),
				proto.PresencePacket,
				encodedPayload,
				client.ID(),
			)

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
			broadcastPresence(hub.baseCtx, hub, client, true)
			asyncFriendsPresence(hub.baseCtx, hub, client)
			log.Printf("client registered: %v", client.ID())

		case client := <-hub.unregister:
			broadcastPresence(hub.baseCtx, hub, client, false)
			if c, ok := hub.clients[client.ID()]; ok {
				delete(hub.clients, client.ID())
				c.Close()
			}
			log.Printf("client unregistered: %v", client.ID())

		case packet := <-hub.request:
			ctx, _ := context.WithTimeout(hub.baseCtx, time.Second*5)
			switch packet.Type {
			case proto.SendMessageRequestPacket:
				if err := handleRequestSendMessagePacket(ctx, hub, packet); err != nil {
					log.Printf("handleChatMessagePacket: %v", err)
				}
			case proto.ReadMessagePacket:
				if err := handleReadMessagePacket(ctx, hub, packet); err != nil {
					log.Printf("handleReadMessagePacket: %v", err)
				}
			case proto.TypingEventPacket:
				if err := handleTypingEventPacket(ctx, hub, packet); err != nil {
					log.Printf("handleTypingEventPacket: %v", err)
				}
			default:
				log.Printf("unknown packet type")
			}
		case <-hub.done:
			return

		}

	}
}
