package chatter

import (
	"net/http"

	"github.com/putto11262002/chatter/core"
)

type WSAuthenticator struct {
	authStore core.AuthStore
}

func NewWSAuthenticator(authStore core.AuthStore) *WSAuthenticator {
	return &WSAuthenticator{authStore: authStore}
}

func (a *WSAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) (string, bool) {
	session := SessionFromRequest(req)
	return session.Username, true
}
