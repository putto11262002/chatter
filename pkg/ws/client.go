package hub

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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

type Client struct {
	conn   *websocket.Conn
	ID     string
	send   chan *Packet
	hub    *Hub
	ticker *time.Ticker
	logger *slog.Logger
}

func NewClient(hub *Hub, conn *websocket.Conn, logger *slog.Logger, id string) *Client {
	return &Client{
		send:   make(chan *Packet),
		ticker: time.NewTicker(pingPeriod),
		conn:   conn,
		hub:    hub,
		logger: logger,
	}
}

func (c *Client) readLoop() {
	defer func() {
		c.hub.disconnectChan <- c
		c.conn.Close()
		c.hub.wg.Done()
		c.logger.Debug("exited read loop")
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		mt, r, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.hub.logger.Debug(fmt.Sprintf("expected close: %v", err))
				return

			}
			if websocket.IsUnexpectedCloseError(err) {
				c.hub.logger.Error(fmt.Sprintf("unexpected close: %v", err))
				return
			}
			c.hub.logger.Error(fmt.Sprintf("NextReader: %v", err))
			return
		}

		packet, err := decodePacket(mt, r)
		if err != nil {
			c.hub.logger.Error(fmt.Sprintf("DecodePacket: %v", err))
			continue

		}

		ctx := NewContext(packet, c, c.hub, c.hub.baseCtx)
		c.hub.request <- ctx
	}

}

func (c *Client) writeLoop() {
	defer func() {
		c.ticker.Stop()
		c.conn.Close()
		c.hub.wg.Done()
		c.logger.Debug("exited write loop")
	}()

	for {
		select {
		case packet, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
				return
			}

			err := encodePacket(c.conn.NextWriter, packet)
			if err != nil {
				c.hub.logger.Error(fmt.Sprintf("EncodePacket: %v", err))
			}
		case <-c.ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.hub.logger.Error(fmt.Sprintf("WritePing: %v", err))
				return
			}

		}
	}
}
