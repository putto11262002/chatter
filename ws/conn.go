package ws

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
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

type WSConn struct {
	conn   *websocket.Conn
	id     string
	in     chan *Packet
	hub    Hub
	ticker *time.Ticker
	logger *slog.Logger
}

func (c *WSConn) pass() chan<- *Packet {
	return c.in
}

func (c *WSConn) close() {
	close(c.in)
}

func (c *WSConn) ID() string {
	return c.id
}

func (c *WSConn) readLoop() {
	defer func() {
		// TODO: do i need try send?
		c.hub.Disconnect(c)
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
	var err error
	defer func() {
		c.ticker.Stop()
		if err != nil {
			c.conn.Close()
		}
		c.logger.Info("exited write loop", slog.String("client.id", c.id))
	}()

	for {
		select {
		case packet, ok := <-c.in:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			err = encodeOutPacket(c.conn.NextWriter, packet)
			if err != nil {
				c.logger.Error(fmt.Sprintf("EncodePacket: %v", err))
			}
		case <-c.ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err = c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error(fmt.Sprintf("WritePing: %v", err))
				return
			}

		}
	}
}

type WSConnFactory struct {
	upgrader websocket.Upgrader
}

func NewWSConnFactory(opts ...WSConnFactoryOpt) *WSConnFactory {
	cf := &WSConnFactory{
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
			return true
		}},
	}

	for _, opt := range opts {
		opt(cf)
	}

	return cf
}

type WSConnFactoryOpt func(*WSConnFactory)

func WithUpgrader(upgrader *websocket.Upgrader) WSConnFactoryOpt {
	return func(wf *WSConnFactory) {
		wf.upgrader = *upgrader
	}
}

func (f *WSConnFactory) NewConn(w http.ResponseWriter, r *http.Request, hub Hub, id string) (Conn, bool) {
	_conn, err := f.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, false
	}
	conn := &WSConn{
		conn:   _conn,
		id:     id,
		in:     make(chan *Packet),
		hub:    hub,
		ticker: time.NewTicker(pingPeriod),
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})),
	}
	return conn, true
}
