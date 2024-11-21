package api

import (
	"net/http"
	"slices"
	"time"

	"github.com/putto11262002/chatter/pkg/auth"
	"github.com/putto11262002/chatter/pkg/chat"
)

type ChatHandler struct {
	chatStore chat.ChatStore
}

func NewChatHandler(chatStore chat.ChatStore) *ChatHandler {
	return &ChatHandler{chatStore: chatStore}
}

type CreatePrivateChatPayload struct {
	Other string `json:"other"`
}

type CreateChatResponse struct {
	ID string `json:"id"`
}

type UserRoomResponse struct {
	Username string `json:"username"`
	RoomID   string `json:"roomID"`
	RoomName string `json:"roomName"`
}

type RoomResponse struct {
	ID    string             `json:"id"`
	Type  chat.ChatType      `json:"type"`
	Users []UserRoomResponse `json:"users"`
}

func NewRoomResponse(room chat.Room) RoomResponse {
	var users []UserRoomResponse
	for _, user := range room.Users {
		users = append(users, UserRoomResponse{
			Username: user.Username,
			RoomID:   user.RoomID,
			RoomName: user.RoomName,
		})
	}
	return RoomResponse{
		ID:    room.ID,
		Type:  room.Type,
		Users: users,
	}
}

type UserRoomsResponse struct {
	Username string `json:"username"`
	RoomName string `json:"roomName"`
	RoomID   string `json:"roomID"`
}

type MessageCreateRequest struct {
	Data string           `json:"data"`
	Type chat.MessageType `json:"type"`
}

type CreateMessageResponse struct {
	ID string `json:"id"`
}

type MessageResponse struct {
	ID     string             `json:"id"`
	Type   chat.MessageType   `json:"type"`
	Data   string             `json:"data"`
	RoomID string             `json:"roomID"`
	Sender string             `json:"sender"`
	SentAt time.Time          `json:"sentAt"`
	Status chat.MessageStatus `json:"status"`
}

func NewMessageResponse(message chat.Message) MessageResponse {
	return MessageResponse{
		ID:     message.ID,
		Type:   message.Type,
		Data:   message.Data,
		RoomID: message.RoomID,
		Sender: message.Sender,
		SentAt: message.SentAt,
		Status: message.Status,
	}
}

func NewMessagesResponse(messages []chat.Message) []MessageResponse {
	messagesResponse := make([]MessageResponse, 0, len(messages))
	for _, message := range messages {
		messagesResponse = append(messagesResponse, NewMessageResponse(message))
	}
	return messagesResponse
}

func NewUserRoomResponse(user chat.RoomUser) UserRoomsResponse {
	return UserRoomsResponse{
		Username: user.Username,
		RoomName: user.RoomName,
		RoomID:   user.RoomID,
	}
}

func NewUserRoomsResponse(users []chat.RoomUser) []UserRoomsResponse {
	response := make([]UserRoomsResponse, 0, len(users))
	for _, user := range users {
		response = append(response, NewUserRoomResponse(user))
	}
	return response
}

func (h *ChatHandler) CreatePrivateChatHandler(w http.ResponseWriter, r *http.Request) error {
	var payload CreatePrivateChatPayload
	DecodeJson(r.Body, &payload)
	r.Body.Close()

	session := sessionFromRequest(r)

	users := [2]string{session.Username, payload.Other}

	id, err := h.chatStore.CreatePrivateChat(r.Context(), users)

	if err != nil {

		switch err {
		case chat.ErrInvalidUser:
			return NewApiError(err.Error(), http.StatusBadRequest)
		case chat.ErrConflictedChat:
			return NewApiError(err.Error(), http.StatusConflict)
		default:
			return err
		}

	}

	WriteJsonResponseWithStatusCode(w, CreateChatResponse{ID: id}, http.StatusCreated)

	return nil
}

func (h *ChatHandler) GetRoomByIDHandler(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("roomID")

	session := sessionFromRequest(r)

	room, err := h.chatStore.GetRoomRoomByID(r.Context(), id)
	if err != nil {
		return err
	}

	if room == nil {
		return NewApiError("room not found", http.StatusNotFound)
	}

	if !slices.ContainsFunc(room.Users, func(user chat.RoomUser) bool {
		return user.Username == session.Username
	}) {
		return NewApiError(auth.ErrUnauthorized.Error(), http.StatusUnauthorized)
	}

	WriteJsonResponse(w, NewRoomResponse(*room))

	return nil
}

func (h *ChatHandler) GetMyUserRoomsHandler(w http.ResponseWriter, r *http.Request) error {
	session := sessionFromRequest(r)
	r.SetPathValue("userID", session.Username)
	return h.GetUserRoomsByUserIDHandler(w, r)
}

func (h *ChatHandler) GetUserRoomsByUserIDHandler(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("userID")

	roomUsers, err := h.chatStore.GetUserRooms(r.Context(), id)
	if err != nil {
		return err
	}

	WriteJsonResponse(w, NewUserRoomsResponse(roomUsers))
	return nil
}

func (h *ChatHandler) SendMessageToRoomHandler(w http.ResponseWriter, r *http.Request) error {
	roomID := r.PathValue("roomID")
	defer r.Body.Close()
	var message MessageCreateRequest
	if err := DecodeJson(r.Body, &message); err != nil {
		WriteJsonResponseWithStatusCode(w,
			NewApiError("invalid json", http.StatusBadRequest), http.StatusBadRequest)
		return nil
	}

	session := sessionFromRequest(r)
	createInput := chat.MessageCreateInput{
		Data:   message.Data,
		Type:   message.Type,
		RoomID: roomID,
		Sender: session.Username,
	}

	created, err := h.chatStore.SendMessageToRoom(r.Context(), createInput)
	if err != nil {
		switch err {
		case chat.ErrChatNotFound:
			return NewApiError(err.Error(), http.StatusNotFound)
		case chat.ErrInvalidMessage:
			return NewApiError(err.Error(), http.StatusBadRequest)
		default:
			return err
		}
	}

	WriteJsonResponseWithStatusCode(w, CreateMessageResponse{ID: created.ID}, http.StatusCreated)
	return nil
}

func (h *ChatHandler) GetRoomMessagesHandler(w http.ResponseWriter, r *http.Request) error {
	roomID := r.PathValue("roomID")

	session := sessionFromRequest(r)

	message, err := h.chatStore.GetRoomMessages(r.Context(), roomID, session.Username)

	if err != nil {
		switch err {
		case chat.ErrChatNotFound:
			return NewApiError(err.Error(), http.StatusNotFound)
		default:
			return err
		}
	}

	WriteJsonResponse(w, NewMessagesResponse(message))
	return nil
}
