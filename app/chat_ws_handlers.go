package chatter

import (
	"encoding/json"
	"fmt"
	"time"

	hub "github.com/putto11262002/chatter/pkg/ws"
	"github.com/putto11262002/chatter/store"
)

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

type ChatWSHandler struct {
	store store.ChatStore
}

func NewChatWSHandler(store store.ChatStore) *ChatWSHandler {
	return &ChatWSHandler{store: store}
}

func (h *ChatWSHandler) MessageHandler(ctx *hub.Request) error {
	var msg MessageData
	if err := json.Unmarshal(ctx.Packet.Body, &msg); err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	input := store.MessageCreateInput{
		Type:   msg.Type,
		Data:   msg.Data,
		RoomID: msg.RoomID,
		Sender: ctx.Sender.ID,
	}

	createdMsg, err := h.store.SendMessageToRoom(ctx.Context(), input)
	if err != nil {
		return fmt.Errorf("SendMessageToRoom: %w", err)
	}

	msg.Sender = createdMsg.Sender
	msg.SentAt = createdMsg.SentAt
	msg.ID = createdMsg.ID

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Marshal: %w", err)
	}

	users, err := h.store.GetRoomMembers(ctx.Context(), msg.RoomID)
	if err != nil {
		return fmt.Errorf("GetRoomMembers: %w", err)
	}

	packet := &hub.Packet{Type: Message, Body: b}
	for _, user := range users {
		ctx.Hub.BroadcastToClients(packet, user.Username)
	}

	return nil
}

func (h *ChatWSHandler) ReadMessage(req *hub.Request) error {
	var readMsg ReadMessageData
	if err := json.Unmarshal(req.Packet.Body, &readMsg); err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	lastMessageRead, readAt, err := h.store.ReadRoomMessages(req.Context(), readMsg.RoomID, readMsg.ReadBy)
	if err != nil {
		return fmt.Errorf("ReadRoomMessages: %w", err)
	}

	readMsg.ReadAt = readAt
	readMsg.LastReadMessage = lastMessageRead

	users, err := h.store.GetRoomMembers(req.Context(), readMsg.RoomID)
	if err != nil {
		return fmt.Errorf("GetRoomMembers: %w", err)
	}

	b, err := json.Marshal(readMsg)
	if err != nil {
		return fmt.Errorf("Marshal: %w", err)
	}

	packet := &hub.Packet{Type: ReadMessage, Body: b}

	for _, user := range users {
		req.Hub.BroadcastToClients(packet, user.Username)
	}
	return nil
}

func (h *ChatWSHandler) TypingHandler(req *hub.Request) error {
	var typing TypingData
	if err := json.Unmarshal(req.Packet.Body, &typing); err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	members, err := h.store.GetRoomMembers(req.Context(), typing.RoomID)
	if err != nil {
		return fmt.Errorf("GetRoomMembers: %w", err)
	}

	packet := req.Packet
	for _, member := range members {
		req.Hub.BroadcastToClients(packet, member.Username)
	}
	return nil
}
