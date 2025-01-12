package ws

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

type WSConn struct {
	conn   *websocket.Conn
	id     string
	in     chan *OutPacket
	hub    Hub
	ticker *time.Ticker
	logger *slog.Logger
}

func (c *WSConn) pass() chan<- *OutPacket {
	return c.in
}

func (c *WSConn) close() {
	close(c.in)
}

func (c *WSConn) ID() string {
	return c.id
}

func NewClient(hub *ConnHub, conn *websocket.Conn, logger *slog.Logger, id string) *WSConn {
	return &WSConn{
		id:     id,
		in:     make(chan *OutPacket),
		ticker: time.NewTicker(pingPeriod),
		conn:   conn,
		hub:    hub,
		logger: logger,
	}
}

func (c *WSConn) readLoop() {
	defer func() {
		// TODO: do i need try send?
		c.hub.disconnect(c)
		c.conn.Close()
		c.logger.Info("exited read loop", slog.String("client.id", c.id))
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
				c.logger.Debug(fmt.Sprintf("expected close: %v", err))
				return

			}
			if websocket.IsUnexpectedCloseError(err) {
				c.logger.Error(fmt.Sprintf("unexpected close: %v", err))
				return
			}
			c.logger.Error(fmt.Sprintf("NextReader: %v", err))
			return
		}

		packet, err := partiallyDecodeInPacket(mt, r)
		if err != nil {
			c.logger.Error(fmt.Sprintf("DecodePacket: %v", err))
			continue

		}
		packet.Sender = c.id

		c.hub.pass(packet)
	}

}

func (c *WSConn) writeLoop() {
	defer func() {
		c.ticker.Stop()
		c.conn.Close()
		c.logger.Info("exited write loop", slog.String("client.id", c.id))
	}()

	for {
		select {
		case packet, ok := <-c.in:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
				return
			}

			err := encodeOutPacket(c.conn.NextWriter, packet)
			if err != nil {
				c.logger.Error(fmt.Sprintf("EncodePacket: %v", err))
			}
		case <-c.ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error(fmt.Sprintf("WritePing: %v", err))
				return
			}

		}
	}
}
