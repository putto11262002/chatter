package hub

import (
	"context"
	"fmt"
	"iter"
	"log"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"sync"
	"time"
)

type Hub struct {
	clients        map[string]*Client
	channels       map[string]*Channel
	connectChan    chan *Client
	disconnectChan chan *Client
	request        chan *Request
	// exit is used to signal that the hub should start exiting
	exit              chan struct{}
	logger            *slog.Logger
	handlers          map[string]Handler
	connectHandler    func(*Hub, *Client) error
	disconnectHandler func(*Hub, *Client) error
	baseCtx           context.Context
	wg                sync.WaitGroup
	authenticator     Authenticator
}

func New(opts ...HubOption) *Hub {
	hub := &Hub{
		clients:        make(map[string]*Client),
		channels:       make(map[string]*Channel),
		connectChan:    make(chan *Client),
		disconnectChan: make(chan *Client),
		request:        make(chan *Request),
		exit:           make(chan struct{}),
		logger: slog.New(slog.NewJSONHandler(os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug})),
		baseCtx:       context.TODO(),
		handlers:      make(map[string]Handler),
		authenticator: &QueryAuthenticator{queryParam: "id"},
	}

	for _, opt := range opts {
		opt(hub)
	}
	return hub
}

type HubOption func(*Hub)

func WithLogger(logger *slog.Logger) HubOption {
	return func(h *Hub) {
		h.logger = logger
	}
}

func WithBaseContext(ctx context.Context) HubOption {
	return func(h *Hub) {
		h.baseCtx = ctx
	}
}

func WithAuthenticator(auth Authenticator) HubOption {
	return func(h *Hub) {
		h.authenticator = auth
	}
}

func (hub *Hub) Start() {
	hub.wg.Add(1)
	go hub.start()
	hub.logger.Debug("hub started")
}

func (hub *Hub) start() {
	defer func() {
		hub.wg.Done()
		hub.logger.Debug("hub exited")
	}()
	for {

		select {
		case <-hub.exit:
			return
		case newC := <-hub.connectChan:
			hub.clients[newC.ID] = newC
			newC.logger.Debug("connected")
			if hub.connectHandler != nil {
				hub.connectHandler(hub, newC)
			}
		case c := <-hub.disconnectChan:
			hub.disconnect(c)
		case ctx := <-hub.request:
			h, ok := hub.handlers[ctx.Packet.Type]
			if !ok {
				ctx.Sender.logger.Error(fmt.Sprintf("handler(%s): not found", ctx.Packet.Type))
				continue
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						ctx.Sender.logger.Error("handler(%s): %v", ctx.Packet.Type, r)

					}

				}()
				err := h(ctx)
				if err != nil {
					ctx.Sender.logger.Error(
						fmt.Sprintf("handler(%s): %v", ctx.Packet.Type, err))
				}
			}()
		}

	}
}

// Close start closing the hub.
// It does not wait for the clean up to complete.
// The closing sequence is as following:
//  1. Deregister connection from the hub then signal connection handler goroutine to close the connection then exit.
//  2. Signal the hub main goroutine to exit.
func (hub *Hub) Close() {
	hub.logger.Debug("closing client connections")
	for _, c := range hub.clients {
		hub.disconnect(c)
	}
	hub.logger.Debug("exiting hub")
	close(hub.exit)
}

func (hub *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := hub.authenticator.Authenticate(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade: %v", err)
	}

	c := NewClient(hub, conn,
		hub.logger.With(slog.String("client.id", id)), id)

	hub.connectChan <- c

	hub.wg.Add(2)
	go c.readLoop()
	go c.writeLoop()
}

func (hub *Hub) Wait() {
	hub.logger.Debug("waiting for hub to close")
	hub.wg.Wait()
	hub.logger.Debug("hub closed")
}

func (hub *Hub) WaitWithTimeout(timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	done := make(chan struct{})
	go func() {
		hub.wg.Wait()
		close(done)
	}()

	select {
	case <-timer.C:
		hub.logger.Debug("hub closed with timeout")
	case <-done:
		hub.logger.Debug("hub closed gracefully")
	}
}

func (hub *Hub) SetHandle(t string, h Handler) {
	hub.handlers[t] = h
}

func (hub *Hub) SetConnectHandler(h func(*Hub, *Client) error) {
	hub.connectHandler = h

}

func (hub *Hub) SetDisconnectHandler(h func(*Hub, *Client) error) {
	hub.disconnectHandler = h
}

// CreateChannel creates a new channel with a given id and adds it to the hub.
func (hub *Hub) CreateChannel(id string) {
	cha := NewChannel(id)
	hub.channels[id] = cha
}

// ListChannels returns a sequence of all the channels that are added to the hub.
func (hub *Hub) ListChannels() iter.Seq[*Channel] {
	return maps.Values(hub.channels)
}

// GetChannel returns a channel with a given id.
// If the channel is not found, the second return value is false.
func (hub *Hub) GetChannel(id string) (*Channel, bool) {
	cha, ok := hub.channels[id]
	return cha, ok
}

// Subscribe subscribes a client to a channel.
// If the channel does not exist, it does nothing.
func (hub *Hub) Subscribe(client string, channel string) {
	if _, ok := hub.clients[client]; !ok {
		return
	}
	if _, ok := hub.channels[channel]; !ok {
		return
	}
	hub.channels[channel].subscribe(client)
}

// Unsubscribe unsubscribes a client from a channel.
// If the channel does not exist, it does nothing.
func (hub *Hub) Unsubscribe(client *Client, channel string) {
	cha, ok := hub.channels[channel]
	if !ok {
		return
	}
	cha.unsubscribe(client)
}

// BroadcastToClients broadcasts a message to a list of clients.
func (hub *Hub) BroadcastToClients(res *Packet, ids ...string) {
	for _, id := range ids {
		c, ok := hub.clients[id]
		if !ok {
			continue
		}
		hub.sendOrDisconnect(c, res)
	}
}

// Broadcast broadcasts a message to all the clients that are connected to the hub.
func (hub *Hub) Broadcast(res *Packet) {
	for _, c := range hub.clients {
		hub.sendOrDisconnect(c, res)
	}
}

// BroadcastToChannels broadcasts a message to a list of channels.
func (hub *Hub) BroadcastToChannels(res *Packet, channels ...string) {
	for _, chn := range channels {
		ch, ok := hub.channels[chn]
		if !ok {
			continue
		}
		for sub := range ch.Subscribers() {
			c, ok := hub.clients[sub]
			if !ok {
				continue
			}
			hub.sendOrDisconnect(c, res)
		}
	}
}

// sendOrDisconnect sends a response message to a client. If the send channel of the
// client is blocked, it disconnects the client.
func (hub *Hub) sendOrDisconnect(c *Client, res *Packet) {
	select {
	case c.send <- res:
	default:
		hub.disconnect(c)
	}
}

func (hub *Hub) disconnect(c *Client) {
	_, ok := hub.clients[c.ID]
	if !ok {
		return
	}
	delete(hub.clients, c.ID)
	close(c.send)
	if hub.disconnectHandler != nil {
		hub.disconnectHandler(hub, c)
	}
}
