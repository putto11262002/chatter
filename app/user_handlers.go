package chatter

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/putto11262002/chatter/core"
	"github.com/putto11262002/chatter/pkg/router"
)

type UserHandler struct {
	store core.UserStore
}

func NewUserHandler(store core.UserStore) *UserHandler {
	return &UserHandler{store: store}
}

func (h *UserHandler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) error {
	var user core.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return fmt.Errorf("Decode: %w", err)
	}
	defer r.Body.Close()

	if err := user.Validate(); err != nil {
		return router.NewJsonError(http.StatusBadRequest, "invalid input")
	}

	if err := h.store.CreateUser(r.Context(), user); err != nil {
		switch err {
		case core.ErrConflictedUser:
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

func (h *UserHandler) GetUserByUsernameHandler(w http.ResponseWriter, r *http.Request) error {
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
