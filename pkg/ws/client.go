package ws

import (
	"context"
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Delegate the check to CORS middleware
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type HubClient struct {
	id          string
	hub         Hub
	conn        *websocket.Conn
	mu          sync.RWMutex
	send        chan *Response
	decoder     PacketDecoder
	encoder     PacketEncoder
	ctx         context.Context
	messageType int
}

func (c *HubClient) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

func (c *HubClient) Send(data *Response) {
	if c == nil {
		return
	}
	// TODO: non blocking write using select

	c.send <- data
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
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		mt, r, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}

			if websocket.IsUnexpectedCloseError(err) {
				log.Printf("unexpected close: %v", err)
				return
			}

			log.Printf("conn.ReadJsono: %v", err)

			return
		}

		packet, err := c.decoder.Decode(r, mt)
		if err != nil {
			log.Printf("decoder.Decode: %v", err)
			continue
		}

		ctx := Request{
			Payload:       packet.Payload,
			Type:          packet.Type,
			Context:       c.ctx,
			CorrelationID: packet.CorrelationID,
			Src:           c.id,
		}

		c.hub.Broadcast(&ctx)

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
			fmt.Printf("writePump close connection because: %v\n", err)
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
			w, err := c.conn.NextWriter(c.encoder.MessageType())

			if err != nil {
				return fmt.Errorf("conn.NextWriter: %v", err)
			}

			c.encoder.Encode(w, &Packet{
				Type:          message.Type,
				Payload:       message.Payload,
				CorrelationID: message.CorrelationID,
			})

			w.Close()

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			fmt.Printf("sending ping to %v\n", c.id)
			err = c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return err

			}
		}
	}
}

type HubClientFactory struct {
	hub Hub
	// AuthAdapter is used to retrieve the client id
	adapter AuthAdapter
	// Context passwed to the client
	// If the is nill the request context is used
	baseCtx  context.Context
	decoder  PacketDecoder
	enconder PacketEncoder
}

func NewHubClientFactory(hub Hub, adapter AuthAdapter, baseCtx context.Context, encoder PacketEncoder, decoder PacketDecoder) *HubClientFactory {
	return &HubClientFactory{
		hub:      hub,
		adapter:  adapter,
		baseCtx:  baseCtx,
		enconder: encoder,
		decoder:  decoder,
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

	var ctx context.Context
	if f.baseCtx != nil {
		ctx = f.baseCtx
	} else {
		ctx = r.Context()
	}

	client := &HubClient{
		ctx:     ctx,
		hub:     f.hub,
		conn:    conn,
		send:    make(chan *Response),
		id:      id,
		encoder: f.enconder,
		decoder: f.decoder,
	}

	f.hub.Register(client)

	go client.writePump()
	go client.readPump()

}
