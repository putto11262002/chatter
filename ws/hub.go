package ws

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type ConnHub struct {
	conns map[string][]Conn

	connectChan chan Conn

	disconnectChan chan Conn
	// in is used to send incoming packets to the manager
	in chan *InPacket
	// exit is used to signal that the manager should stop accepting new connections and exit
	exit chan struct{}

	logger *slog.Logger

	OnConnect func(HubActions, Conn) error

	OnDisconnect func(HubActions, Conn) error

	baseCtx context.Context

	wg sync.WaitGroup

	OnPacketIn func(*InPacket)

	connFactory ConnFactory

	authenticator Authenticator

	closeTimeout time.Duration

	ready atomic.Bool
	mu    sync.Mutex
}

func New(cf ConnFactory, a Authenticator, opts ...HubOption) *ConnHub {
	hub := &ConnHub{
		conns:          make(map[string][]Conn),
		connectChan:    make(chan Conn),
		disconnectChan: make(chan Conn),
		in:             make(chan *InPacket),
		exit:           make(chan struct{}),
		logger: slog.New(slog.NewJSONHandler(os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug})),
		baseCtx:       context.TODO(),
		closeTimeout:  time.Second * 10,
		authenticator: a,
		connFactory:   cf,
	}

	for _, opt := range opts {
		opt(hub)
	}

	return hub
}

type HubOption func(*ConnHub)

func WithLogger(logger *slog.Logger) HubOption {
	return func(h *ConnHub) {
		h.logger = logger
	}
}

func WithBaseContext(ctx context.Context) HubOption {
	return func(h *ConnHub) {
		h.baseCtx = ctx
	}
}

func (hub *ConnHub) Start() {
	hub.wg.Add(1)
	go func() {
		defer func() {
			hub.wg.Done()
			hub.logger.Info("hub stopped")
		}()
		hub.start()
	}()
	hub.logger.Info("hub started")
}

func (hub *ConnHub) start() {
	hub.ready.Store(true)
	defer hub.ready.Store(false)
	for {

		select {
		case <-hub.exit:
			return
		case newC := <-hub.connectChan:
			hub.mu.Lock()
			hub.addConn(newC)
			hub.mu.Unlock()
		case c := <-hub.disconnectChan:
			hub.mu.Lock()
			hub.removeConn(c)
			hub.mu.Unlock()
		case packetIn := <-hub.in:
			if hub.OnPacketIn != nil {
				hub.OnPacketIn(packetIn)
			}
		}

	}
}

// Close start closing the hub.
// It does not wait for the clean up to complete.
// The closing sequence is as following:
//  1. Deregister connection from the hub then signal connection handler goroutine to close the connection then exit.
//  2. Signal the hub main goroutine to exit.
func (hub *ConnHub) Close() {
	hub.logger.Info("closing connections...")
	hub.mu.Lock()
	for _, conns := range hub.conns {
		for _, conn := range conns {
			hub.removeConn(conn)
		}
	}
	hub.mu.Unlock()
	hub.logger.Info("exiting hub...")
	close(hub.exit)
	timer := time.NewTimer(hub.closeTimeout)
	defer timer.Stop()
	done := make(chan struct{})
	go func() {
		hub.wg.Wait()
		close(done)
	}()

	select {
	case <-timer.C:
		hub.logger.Info("hub closed with timeout")
	case <-done:
		hub.logger.Info("hub closed gracefully")
	}
}

func (hub *ConnHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, ok := hub.authenticator.Authenticate(w, r)
	if !ok {
		log.Println("authenticator failed")
		return
	}
	conn, ok := hub.connFactory.NewConn(w, r, hub, id)
	if !ok {
		return
	}
	hub.handleConn(conn)
}

func (hub *ConnHub) handleConn(conn Conn) {
	hub.wg.Add(1)
	go func() {
		defer hub.wg.Done()
		conn.readLoop()
	}()

	hub.wg.Add(1)
	go func() {
		defer hub.wg.Done()
		conn.writeLoop()
	}()
	hub.connect(conn)
}

// sendOrDisconnect sends a response message to a client. If the send channel of the
// client is blocked, it disconnects the client.
func (hub *ConnHub) sendOrDisconnect(c Conn, p *OutPacket) {
	select {
	case c.pass() <- p:
	default:
		hub.disconnect(c)
	}
}

func (hub *ConnHub) connect(c Conn) {
	hub.connectChan <- c
}

func (hub *ConnHub) disconnect(c Conn) {
	hub.disconnectChan <- c
}

func (hub *ConnHub) pass(packet *InPacket) {
	hub.in <- packet
}

func (hub *ConnHub) removeConn(c Conn) {
	_, ok := hub.conns[c.ID()]
	if !ok {
		return
	}
	delete(hub.conns, c.ID())
	c.close()

	if hub.OnDisconnect != nil {
		hub.OnDisconnect(hub, c)
	}
}

func (hub *ConnHub) addConn(c Conn) {

	if _, ok := hub.conns[c.ID()]; !ok {
		hub.conns[c.ID()] = make([]Conn, 0, 1)
	}
	hub.conns[c.ID()] = append(hub.conns[c.ID()], c)
	hub.logger.Info("new connection", slog.String("id", c.ID()))

	if hub.OnConnect != nil {
		hub.OnConnect(hub, c)
	}
}
