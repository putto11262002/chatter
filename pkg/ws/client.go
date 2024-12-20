package ws

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type HubClient struct {
	id   string
	hub  Hub
	conn *websocket.Conn
	mu   sync.RWMutex
	send chan Message
}

func (c *HubClient) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

func (c *HubClient) Send(msg Message) {
	if c == nil {
		return
	}
	// TODO: non blocking write using select

	c.send <- msg
}

func (c *HubClient) Close() error {
	if c == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *HubClient) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	fmt.Printf("readPump: %v\n", c.id)
	for {
		var message JsonMessage
		err := c.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}

			if websocket.IsUnexpectedCloseError(err) {
				log.Printf("unexpected close: %v", err)
				return
			}

			log.Printf("reading message: %v", err)

			return
		}

		fmt.Println("message: ", message)

		c.hub.Broadcast(message.Message(c.id))

	}

}

// The error is used to indicate if the connection must be closed forcefully.
// If the error is nil, don't close the connection just yet
// wait for the close message from the peer and close the connection in readPump.
// Otherwise, close the connection immediately.
func (c *HubClient) writePump() (err error) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err != nil {
			c.hub.Unregister(c)
			c.conn.Close()
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// the hub has closed the channel
				c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

				return nil
			}

			err = c.conn.WriteJSON(message)

			if err != nil {
				return err
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err = c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return err
			}
		}
	}
}

type AuthAdapter interface {
	Authenticate(r *http.Request) (string, error)
}

type HubClientFactory struct {
	hub     Hub
	adapter AuthAdapter
}

func NewHubClientFactory(hub Hub, adapter AuthAdapter) *HubClientFactory {
	return &HubClientFactory{
		hub:     hub,
		adapter: adapter,
	}
}

func (f *HubClientFactory) HandleFunc(w http.ResponseWriter, r *http.Request) {
	id, err := f.adapter.Authenticate(r)
	if err != nil {
		fmt.Printf("error: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %v", err)
		return
	}

	client := &HubClient{
		hub:  f.hub,
		conn: conn,
		send: make(chan Message, SendChannelSize),
		id:   id,
	}

	f.hub.Register(client)

	go client.writePump()
	go client.readPump()

}
