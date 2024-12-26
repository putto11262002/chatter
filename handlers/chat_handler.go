package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/putto11262002/chatter/models"
	"github.com/putto11262002/chatter/pkg/router"
	"github.com/putto11262002/chatter/store"
)

type ChatHandler struct {
	chatStore store.ChatStore
}

func NewChatHandler(chatStore store.ChatStore) *ChatHandler {
	return &ChatHandler{chatStore: chatStore}
}

func (h *ChatHandler) CreateGroupChatHandler(w http.ResponseWriter, r *http.Request) error {
	var payload CreateGroupChatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return err
	}
	r.Body.Close()
	session := SessionFromRequest(r)
	payload.Members = append(payload.Members, session.Username)

	id, err := h.chatStore.CreateGroupChat(r.Context(), payload.Name, payload.Members...)
	if err != nil {
		if err == store.ErrInvalidUser {
			return router.NewJsonError(http.StatusBadRequest, err.Error())
		}
		return err
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NewCreateChatResponse(id))
	return nil
}

func (h *ChatHandler) CreatePrivateChatHandler(w http.ResponseWriter, r *http.Request) error {
	var payload CreatePrivateChatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return err
	}
	r.Body.Close()
	session := SessionFromRequest(r)
	users := [2]string{session.Username, payload.Other}

	id, err := h.chatStore.CreatePrivateChat(r.Context(), users)
	if err != nil {

		switch err {
		case store.ErrInvalidUser:
			return router.NewJsonError(http.StatusBadRequest, err.Error())
		case store.ErrConflictedRoom:
			return router.NewJsonError(http.StatusConflict, err.Error())
		default:
			return err
		}

	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NewCreateChatResponse(id))
	return nil
}

func (h *ChatHandler) GetRoomByIDHandler(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("roomID")
	session := SessionFromRequest(r)

	inRoom, err := h.chatStore.IsRoomMember(r.Context(), id, session.Username)
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

func (h *ChatHandler) GetMyRoomSummaries(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	r.SetPathValue("userID", session.Username)
	return h.GetRoomSummariesByID(w, r)
}

func (h *ChatHandler) GetRoomSummariesByID(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("userID")
	query := r.URL.Query()
	limitStr := query.Get("limit")
	limit, _ := strconv.Atoi(limitStr)
	offsetStr := query.Get("offset")
	offset, _ := strconv.Atoi(offsetStr)

	roomSummaries, err := h.chatStore.GetRoomSummaries(r.Context(), id, offset, limit)
	if err != nil {
		return err
	}

	if roomSummaries == nil {
		roomSummaries = []models.RoomSummary{}
	}

	if err := json.NewEncoder(w).Encode(roomSummaries); err != nil {
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
		messages = []models.Message{}
	}

	json.NewEncoder(w).Encode(messages)
	return nil
}
