package chatter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/putto11262002/chatter/core"
)

const (
	MessageEvent     = "message"
	ReadMessageEvent = "read_message"
	OnlineEvent      = "online"
	OfflineEvent     = "offline"
	IsOnlineEvent    = "is_online"
	TypingEvent      = "typing"
)

type MessageEventPayload struct {
	ID     int       `json:"id"`
	RoomID string    `json:"room_id"`
	Type   int       `json:"type"`
	Data   string    `json:"data"`
	Sender string    `json:"sender"`
	SentAt time.Time `json:"sent_at"`
}

type ReadMessageEventPayload struct {
	RoomID          string    `json:"room_id"`
	ReadAt          time.Time `json:"read_at"`
	ReadBy          string    `json:"read_by"`
	LastReadMessage int       `json:"last_read_message"`
}

type TypingEventPayload struct {
	Typing   bool   `json:"typing"`
	Username string `json:"username"`
	RoomID   string `json:"room_id"`
}

type OnlineEventPayload struct {
	Username string `json:"username"`
}

type OfflineEventPayload struct {
	Username string `json:"username"`
}

type IsOnlineEventPayload struct {
	Username string `json:"username"`
}

func (app *App) MessageEventHandler(ctx context.Context, e *core.Event) error {
	var msg MessageEventPayload
	if err := json.Unmarshal(e.Payload, &msg); err != nil {
		return fmt.Errorf("unmarshal message event payload: %w", err)
	}

	input := core.MessageCreateInput{
		Type:   msg.Type,
		Data:   msg.Data,
		RoomID: msg.RoomID,
		Sender: e.Dispatcher,
	}

	createdMsg, err := app.chatStore.SendMessageToRoom(ctx, input)
	if err != nil {
		return fmt.Errorf("SendMessageToRoom: %w", err)
	}

	msg.Sender = createdMsg.Sender
	msg.SentAt = createdMsg.SentAt
	msg.ID = createdMsg.ID

	members, err := app.chatStore.GetRoomMembers(ctx, msg.RoomID)
	if err != nil {
		return fmt.Errorf("GetRoomMembers: %w", err)
	}

	usernames := make([]string, 0, len(members))
	for _, member := range members {
		usernames = append(usernames, member.Username)
	}

	if err := app.eventRouter.EmitTo(MessageEvent, msg, usernames...); err != nil {
		return err
	}
	return nil
}

func (app *App) ReadMessageHandler(ctx context.Context, e *core.Event) error {
	var readMsg ReadMessageEventPayload
	if err := json.Unmarshal(e.Payload, &readMsg); err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	lastMessageRead, readAt, err := app.chatStore.ReadRoomMessages(ctx, readMsg.RoomID, readMsg.ReadBy)
	if err != nil {
		return fmt.Errorf("ReadRoomMessages: %w", err)
	}

	readMsg.ReadAt = readAt
	readMsg.LastReadMessage = lastMessageRead

	members, err := app.chatStore.GetRoomMembers(ctx, readMsg.RoomID)
	if err != nil {
		return fmt.Errorf("GetRoomMembers: %w", err)
	}

	usernames := make([]string, 0, len(members))
	app.eventRouter.EmitTo(ReadMessageEvent, readMsg, usernames...)

	return nil
}

func (app *App) TypingHandler(ctx context.Context, e *core.Event) error {
	var typing TypingEventPayload
	if err := json.Unmarshal(e.Payload, &typing); err != nil {
		return fmt.Errorf("Unmarshal: %w", err)
	}

	members, err := app.chatStore.GetRoomMembers(ctx, typing.RoomID)
	if err != nil {
		return fmt.Errorf("GetRoomMembers: %w", err)
	}

	usernames := make([]string, 0, len(members))
	for _, member := range members {
		usernames = append(usernames, member.Username)
	}

	if err := app.eventRouter.EmitTo(TypingEvent, typing, usernames...); err != nil {
		return err
	}

	return nil

}
