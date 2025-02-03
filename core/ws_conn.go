package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	conn             *websocket.Conn
	context          context.Context
	username         string
	id               int
	writeStream      chan *Event
	readStream       chan *Event
	notifyDisconnect func()
	ticker           *time.Ticker
	logger           *slog.Logger
}

func (c *Conn) close() {
	close(c.writeStream)
}
func (c *Conn) readLoop() {
	c.logger.Info("read loop started")
	defer func() {
		// TODO: do i need try send?
		c.notifyDisconnect()
		c.conn.Close()
		c.logger.Info("read loop stoped")
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		format, r, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.Info(fmt.Sprintf("expected close: %v", err))
				return

			}
			if websocket.IsUnexpectedCloseError(err) {
				c.logger.Error(fmt.Sprintf("unexpected close: %v", err))
				return
			}
			c.logger.Error(fmt.Sprintf("NextReader: %v", err))
			return
		}

		if format != websocket.TextMessage {
			c.logger.Error(fmt.Sprintf("unexpected message format: %v", format))
			continue
		}

		var event Event
		if err := DecodeEvent(r, &event); err != nil {
			c.logger.Error(err.Error())
		}
		event.Dispatcher = c.username

		c.logger.Debug(event.String())

		c.readStream <- &event
	}

}

func (c *Conn) writeLoop() {
	c.logger.Info("write loop started")
	var err error
	defer func() {
		c.ticker.Stop()
		if err != nil {
			c.conn.Close()
		}
		c.logger.Info("write loop stoped")
	}()

	for {
		select {
		case e, ok := <-c.writeStream:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.logger.Info("write stream closed")
				c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				c.logger.Info("sending close message")
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.logger.Error(fmt.Sprintf("getting next writer: %v", err))
				return
			}
			if err := EncodeEvent(w, e); err != nil {
				c.logger.Error(err.Error())

			}
			w.Close()
		case <-c.context.Done():
			c.logger.Info("context done")
			return
		case <-c.ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err = c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error(fmt.Sprintf("writing ping: %v", err))
				return
			}

		}
	}
}
