package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/putto11262002/chatter/models"
	"github.com/putto11262002/chatter/pkg/router"
	"github.com/putto11262002/chatter/store"
)

type UserHandler struct {
	store store.UserStore
}

func NewUserHandler(store store.UserStore) *UserHandler {
	return &UserHandler{store: store}
}

func (h *UserHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) error {
	var payload CreateUserPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return fmt.Errorf("Decode: %w", err)
	}
	defer r.Body.Close()
	input := models.User{
		Username: payload.Username,
		Password: payload.Password,
		Name:     payload.Name,
	}

	if err := h.store.CreateUser(r.Context(), input); err != nil {
		switch err {
		case store.ErrConflictedUser:
			return router.NewJsonError(http.StatusConflict, "user already exists")
		default:
			return err
		}
	}

	w.WriteHeader(http.StatusCreated)

	return nil
}

func (h *UserHandler) MeHandler(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	user, err := h.store.GetUserByUsername(r.Context(), session.Username)
	if err != nil {
		return fmt.Errorf("get user by username: %w", err)
	}

	if user == nil {
		return router.NewJsonError(http.StatusNotFound, "user not found")
	}

	json.NewEncoder(w).Encode(user)
	return nil
}

func (h *UserHandler) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) error {
	user, err := h.store.GetUserByUsername(r.Context(), r.PathValue("username"))
	if err != nil {
		return err
	}

	if user == nil {
		return router.NewJsonError(http.StatusNotFound, "user not found")
	}

	json.NewEncoder(w).Encode(user)
	return nil
}
