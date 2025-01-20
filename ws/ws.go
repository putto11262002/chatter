package ws

import (
	"net/http"
)

type Hub interface {
	Connect(Conn)
	Disconnect(Conn)
	Start()
	// Close closes the hub and releases any resources with time out.
	// It should wait for the clean up to complete or until the time out.
	Close()
	// ServeHTTP handles the HTTP request and upgrade the connection to a websocket connection
	// then add the connection to the hub.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	// pass passes a packet to the hub.
	// if the id in not register to the hub the packet is drop
	pass(*Packet)

	OnPacket(func(HubActions, *Packet))

	OnConnect(func(HubActions, Conn))

	OnDisconnect(func(HubActions, Conn))
}

type ConnFactory interface {
	// NewConn creates a new connection from the request and response.
	// If the connection is created successfully, it should return the connection and true.
	// If the connection is not created successfully, it should return nil and false.
	NewConn(w http.ResponseWriter, r *http.Request, hub Hub, id string) (Conn, bool)
}

type Conn interface {
	// pass returns a write-only channel that the hub can use to send messages to the client.
	pass() chan<- *Packet
	// close initiates the closing of the connection.
	// It should close the connection and release any resources.
	// It should be non-blocking.
	close()
	// ID returns the unique identifier of the client that the connection belongs to.
	// A client can have multiple connections.
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
