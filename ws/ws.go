package ws

import (
	"net/http"
)

type Hub interface {
	connect(c Conn)
	disconnect(c Conn)
	Start()
	Close()
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	pass(*InPacket)
}

type ConnFactory interface {
	// NewConn creates a new connection from the request and response.
	// If the connection is created successfully, it should return the connection and true.
	// If the connection is not created successfully, it should return nil and false.
	NewConn(w http.ResponseWriter, r *http.Request, hub Hub, id string) (Conn, bool)
}

type Conn interface {
	pass() chan<- *OutPacket
	close()
	ID() string
	readLoop()
	writeLoop()
}

type Authenticator interface {
	// Authenticate authenticates the request and returns the client id.
	// In the case of a successful authentication, it should return the client id.
	// In the case of a fail authentication, it should return an error.
	// Authenticate should be safe to be called concurrently.
	Authenticate(w http.ResponseWriter, req *http.Request) (string, bool)
}

type AuthneticateFunc func(req *http.Request) (string, bool)

func (f AuthneticateFunc) Authenticate(req *http.Request) (string, bool) {
	return f(req)
}
