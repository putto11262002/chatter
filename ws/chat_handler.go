package ws

import (
	"encoding/json"
	"fmt"

	hub "github.com/putto11262002/chatter/pkg/ws"
	"github.com/putto11262002/chatter/store"
)

type ChatWSHandler struct {
	store store.ChatStore
}

func (h *ChatWSHandler) MessageHandler(hu *hub.Hub, ctx *hub.Request) error {
	var msg MessageData
	if err := json.Unmarshal(ctx.Packet.Body, &msg); err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	createdMsg, err := h.store.SendMessageToRoom(ctx.Context(), store.MessageCreateInput{
		Type:   msg.Type,
		Data:   msg.Data,
		RoomID: msg.RoomID,
		Sender: ctx.Sender.ID,
	})
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
		hu.BroadcastToClients(packet, user.Username)
	}

	return nil
}

func (h *ChatWSHandler) ReadMessage(hu *hub.Hub, req *hub.Request) error {
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
		hu.BroadcastToClients(packet, user.Username)
	}
	return nil
}

func (h *ChatWSHandler) TypingHandler(hu *hub.Hub, req *hub.Request) error {
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
		hu.BroadcastToClients(packet, member.Username)
	}
	return nil
}
