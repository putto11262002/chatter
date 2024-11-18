package ws

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// TODO: ping pong
// TODO: fine tune buffer sizes
const (
	ReadBufferSize  = 1024
	WriteBufferSize = 102

	WriteWait = 10 * time.Second
	ReadWait  = 10 * time.Second

	maxMsgSize = 512

	SendChannelSize = 10
)

type JsonMessage struct {
	Type MessageType `json:"type"`
	To   string      `json:"to"`
	Data string      `json:"data"`
}

func (m *JsonMessage) Message(from string) Message {
	return Message{
		Type: m.Type,
		Data: m.Data,
		To:   m.To,
		From: from,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  ReadBufferSize,
	WriteBufferSize: WriteBufferSize,
}

type WSClient struct {
	id   string
	hub  *Hub
	conn *websocket.Conn
	mu   sync.RWMutex

	// outbound messages
	send chan Message
}

func (c *WSClient) Close() {
	close(c.send)
}

func (c *WSClient) readPump() {
	defer func() {
		c.hub.Unregister <- c
	}()

	c.conn.SetReadLimit(maxMsgSize)

	for {
		jsonMessage := new(JsonMessage)
		err := c.conn.ReadJSON(jsonMessage)
		if err != nil {

			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure) {
				log.Printf("close gracefully: %v", err)
				return
			}

			if websocket.IsUnexpectedCloseError(err) {
				log.Printf("close unexpectedly: %v", err)
				return
			}

			return

		}

		c.hub.Incoming <- jsonMessage.Message(c.id)
	}

}

func (c *WSClient) writePump() {
	defer func() {
		c.conn.Close()

	}()

	for {

		select {
		case message, ok := <-c.send:

			if !ok {
				// if the hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

				// if the connection cannot be closed cleanly force close it.

				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				c.hub.Unregister <- c
			}
			// TODO: Batch messages

		}

	}

}

type WSClientFactory struct {
	hub     *Hub
	adapter AuthAdapter
}

type AuthAdapter interface {
	Authenticate(r *http.Request) (string, error)
}

func NewWSClientFactory(hub *Hub, adapter AuthAdapter) *WSClientFactory {
	return &WSClientFactory{
		hub:     hub,
		adapter: adapter,
	}
}

func (f *WSClientFactory) HandleFunc(w http.ResponseWriter, r *http.Request) {
	id, err := f.adapter.Authenticate(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %v", err)
		return
	}

	client := &WSClient{
		hub:  f.hub,
		conn: conn,
		send: make(chan Message, SendChannelSize),
		id:   id,
	}

	f.hub.Register <- client

	go client.writePump()
	go client.readPump()

}
