package chatter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/putto11262002/chatter/core"
	"github.com/putto11262002/chatter/pkg/router"
)

type AuthHandler struct {
	store core.AuthStore
}

func NewAuthHandler(store core.AuthStore) *AuthHandler {
	return &AuthHandler{store: store}
}

type SigninPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) SigninHandler(w http.ResponseWriter, r *http.Request) error {
	var payload SigninPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return fmt.Errorf("Decode: %w", err)
	}
	defer r.Body.Close()

	session, err := h.store.NewSession(r.Context(), payload.Username, payload.Password)

	if err != nil {
		if errors.Is(err, core.ErrBadCredentials) {
			return router.NewJsonError(http.StatusUnauthorized, err.Error())
		}
		return err
	}

	cookie := http.Cookie{
		Name:     AuthCookieName,
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Path:     "/",
	}

	http.SetCookie(w, &cookie)

	if err := json.NewEncoder(w).Encode(session); err != nil {
		return fmt.Errorf("Encode: %w", err)
	}
	return nil
}

func (h *AuthHandler) SignoutHandler(w http.ResponseWriter, r *http.Request) error {
	session := SessionFromRequest(r)
	if err := h.store.DestroySession(r.Context(), session); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})
	w.WriteHeader(http.StatusOK)
	return nil
}
