package chatter

import (
	"net/http"

	"github.com/putto11262002/chatter/handlers"
)

type Authenticator struct {
}

func (a *Authenticator) Authenticate(r *http.Request) (string, error) {
	session := handlers.SessionFromRequest(r)
	return session.Username, nil
}
