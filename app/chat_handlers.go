package chatter

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/putto11262002/chatter/core"
	"github.com/putto11262002/chatter/pkg/router"
)

type ChatHandler struct {
	chatStore core.ChatStore
}

func NewChatHandler(chatStore core.ChatStore) *ChatHandler {
	return &ChatHandler{chatStore: chatStore}
}

type CreateRoomPayload struct {
	Name string `json:"name"`
}

type CreateRoomResponse struct {
	ID string `json:"id"`
}

type IsInRoomResponse struct {
	OK   bool            `json:"ok"`
	Role core.MemberRole `json:"role"`
}

func (h *ChatHandler) CreateRoomHandler(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	var payload CreateRoomPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return err
	}
	r.Body.Close()

	id, err := h.chatStore.CreateRoom(r.Context(), payload.Name, session.Username)
	if err != nil {
		if err == core.ErrInvalidUser {
			return router.NewJsonError(http.StatusBadRequest, err.Error())
		}
		return err
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateRoomResponse{ID: id})
	return nil
}

type AddRoomMemberPayload struct {
	Username string          `json:"username" validate:"required"`
	Role     core.MemberRole `json:"role" validate:"required"`
}

func (h *ChatHandler) AddRoomMemberHandler(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	inRoom, role, err := h.chatStore.IsRoomMember(r.Context(), r.PathValue("roomID"), session.Username)
	if err != nil {
		return err
	}
	if !inRoom {
		return router.NewJsonError(http.StatusForbidden, core.ErrInvalidRoom.Error())
	}

	if !(role == core.Admin || role == core.Owner) {
		return router.NewJsonError(http.StatusForbidden, core.ErrDisAllowedOperation.Error())

	}

	var payload AddRoomMemberPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return err
	}
	r.Body.Close()

	if err := validate.Struct(payload); err != nil {
		return router.NewJsonError(http.StatusBadRequest, "invalid input")
	}

	if err := h.chatStore.AddRoomMember(r.Context(), r.PathValue("roomID"), payload.Username, payload.Role); err != nil {
		if err == core.ErrInvalidRoom || err == core.ErrInvalidUser {
			return router.NewJsonError(http.StatusBadRequest, err.Error())
		}
		if err == core.ErrDisAllowedOperation {
			return router.NewJsonError(http.StatusForbidden, err.Error())
		}
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *ChatHandler) RemoveRoomMemberHandler(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	roomID := r.PathValue("roomID")
	userID := r.PathValue("userID")
	inRoom, role, err := h.chatStore.IsRoomMember(r.Context(), roomID, session.Username)
	if err != nil {
		return err
	}
	if !inRoom {
		return router.NewJsonError(http.StatusForbidden, core.ErrInvalidRoom.Error())
	}

	if !(role == core.Admin || role == core.Owner) {
		return router.NewJsonError(http.StatusForbidden, core.ErrDisAllowedOperation.Error())
	}

	if err := h.chatStore.RemoveRoomMember(r.Context(), roomID, userID); err != nil {
		if err == core.ErrInvalidRoom || err == core.ErrInvalidMember {
			return router.NewJsonError(http.StatusBadRequest, err.Error())
		}
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *ChatHandler) GetRoomByIDHandler(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	id := r.PathValue("roomID")

	inRoom, _, err := h.chatStore.IsRoomMember(r.Context(), id, session.Username)
	if err != nil {
		return err
	}

	if !inRoom {
		return router.NewJsonError(http.StatusForbidden, "you are not in this room")
	}

	room, err := h.chatStore.GetRoomByID(r.Context(), id)
	if err != nil {
		return err
	}
	if room == nil {
		return router.NewJsonError(http.StatusNotFound, "room not found")
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(room)
	return nil
}

func (h *ChatHandler) GetMyRoomsHandler(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	r.SetPathValue("userID", session.Username)
	return h.GetRoomUserRoomsHandler(w, r)
}

func (h *ChatHandler) GetRoomUserRoomsHandler(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("userID")
	query := r.URL.Query()
	limitStr := query.Get("limit")
	limit, _ := strconv.Atoi(limitStr)
	offsetStr := query.Get("offset")
	offset, _ := strconv.Atoi(offsetStr)

	rooms, err := h.chatStore.GetUserRooms(r.Context(), id, offset, limit)
	if err != nil {
		return err
	}

	if rooms == nil {
		rooms = []core.Room{}
	}

	if err := json.NewEncoder(w).Encode(rooms); err != nil {
		return err
	}

	return nil
}

func (h *ChatHandler) GetRoomMessagesHandler(w http.ResponseWriter, r *http.Request) error {
	roomID := r.PathValue("roomID")
	query := r.URL.Query()
	limitStr := query.Get("limit")
	limit, _ := strconv.Atoi(limitStr)
	offsetStr := query.Get("offset")
	offset, _ := strconv.Atoi(offsetStr)

	messages, err := h.chatStore.GetRoomMessages(r.Context(), roomID, offset, limit)
	if err != nil {
		return err
	}

	if messages == nil {
		messages = []core.Message{}
	}

	json.NewEncoder(w).Encode(messages)
	return nil
}

func (h *ChatHandler) SendMessageHandler(w http.ResponseWriter, r *http.Request) error {
	var payload core.MessageCreateInput
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return err
	}
	r.Body.Close()

	if err := payload.Validate(); err != nil {
		return router.NewJsonError(http.StatusBadRequest, "invalid input")
	}

	message, err := h.chatStore.SendMessageToRoom(r.Context(), payload)
	if err != nil {
		if err == core.ErrInvalidRoom || err == core.ErrInvalidMessageType || err == core.ErrInvalidMessage {
			return router.NewJsonError(http.StatusBadRequest, err.Error())
		}
		return err
	}

	json.NewEncoder(w).Encode(message)
	return nil
}
