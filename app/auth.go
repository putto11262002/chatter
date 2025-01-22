package chatter

import (
	"net/http"
)

type Authenticator struct {
}

func (a *Authenticator) Authenticate(r *http.Request) (string, error) {
	session := SessionFromRequest(r)
	return session.Username, nil
}
