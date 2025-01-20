package ws

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type HubState int

const (
	StateClosed HubState = iota
	StateClosing
	StateRunning
)

type ConnHub struct {
	conns map[string][]Conn

	connectChan chan Conn

	disconnectChan chan Conn
	// in is used to send incoming packets to the manager
	in chan *Packet
	// exit is used to signal that the manager should stop accepting new connections and exit
	exit chan struct{}

	logger *slog.Logger

	onConnect func(HubActions, Conn)

	onDisconnect func(HubActions, Conn)

	baseCtx context.Context

	wg sync.WaitGroup

	onPacket func(HubActions, *Packet)

	connFactory ConnFactory

	authenticator Authenticator

	closeTimeout time.Duration
	// ready indicates whether the hub is ready to accept new connections.
	// Passing packets to the hub when the hub is not ready will block.
	ready atomic.Bool
	// close indicates whether the hub is closed.
	// The hub is considered close when the main goroutine is stopped.
	state HubState
	mu    sync.RWMutex
}

func New(cf ConnFactory, a Authenticator, opts ...HubOption) *ConnHub {
	hub := &ConnHub{
		conns:          make(map[string][]Conn),
		connectChan:    make(chan Conn),
		disconnectChan: make(chan Conn),
		in:             make(chan *Packet),
		exit:           make(chan struct{}),
		logger: slog.New(slog.NewTextHandler(os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})),
		baseCtx:       context.TODO(),
		closeTimeout:  time.Second * 10,
		authenticator: a,
		connFactory:   cf,
		state:         StateClosed,
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
	hub.mu.Lock()
	hub.state = StateRunning
	hub.mu.Unlock()
	defer func() {
		hub.mu.Lock()
		hub.state = StateClosed
		hub.mu.Unlock()
	}()
	for {

		select {
		case <-hub.exit:
			return
		case newC := <-hub.connectChan:
			hub.connect(newC)
		case c := <-hub.disconnectChan:
			hub.disconnect(c)
		case packetIn := <-hub.in:
			packetIn.context = hub.baseCtx
			if hub.onPacket != nil {
				hub.onPacket(hub, packetIn)
			}
		}

	}
}

func (hub *ConnHub) OnPacket(f func(HubActions, *Packet)) {
	hub.onPacket = f
}

func (hub *ConnHub) OnConnect(f func(HubActions, Conn)) {
	hub.onConnect = f
}

func (hub *ConnHub) OnDisconnect(f func(HubActions, Conn)) {
	hub.onDisconnect = f
}

// Close start closing the hub.
// It does not wait for the clean up to complete.
// The closing sequence is as following:
//  1. Deregister connection from the hub then signal connection handler goroutine to close the connection then exit.
//  2. Signal the hub main goroutine to exit.
func (hub *ConnHub) Close() {
	hub.logger.Info("closing connections...")
	if hub.state != StateRunning {
		return
	}
	hub.mu.Lock()
	hub.state = StateClosing
	hub.mu.Unlock()
	for _, conns := range hub.conns {
		for i := len(conns) - 1; i >= 0; i-- {
			hub.disconnect((conns)[i])
		}
	}
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
	hub.Connect(conn)
}

func (hub *ConnHub) startConn(conn Conn) {
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
}

// sendOrDisconnect sends a response message to a client. If the send channel of the
// client is blocked, it disconnects the client.
func (hub *ConnHub) sendOrDisconnect(c Conn, p *Packet) {
	select {
	case c.pass() <- p:
	default:
		hub.disconnect(c)
	}
}

func (hub *ConnHub) Connect(c Conn) {
	hub.connectChan <- c
}

func (hub *ConnHub) Disconnect(c Conn) {
	hub.disconnectChan <- c
}

func (hub *ConnHub) pass(packet *Packet) {
	hub.in <- packet
}

func (hub *ConnHub) connect(c Conn) {
	hub.startConn(c)
	hub.mu.Lock()
	hub.addConn(c)
	hub.mu.Unlock()
	hub.logger.Info("new connection", slog.String("id", c.ID()))
	if hub.onConnect != nil {
		hub.onConnect(hub, c)
	}
}

func (hub *ConnHub) disconnect(c Conn) {
	hub.mu.Lock()
	ok := hub.removeConn(c)
	hub.mu.Unlock()
	if !ok {
		return
	}
	c.close()
	if hub.onDisconnect != nil {
		hub.onDisconnect(hub, c)
	}
}

func (hub *ConnHub) removeConn(c Conn) bool {
	conns, ok := hub.conns[c.ID()]
	if !ok {
		return false
	}
	if conns == nil {
		return false
	}

	idx := slices.Index(conns, c)
	if idx == -1 {
		return false
	}
	conns = slices.Delete(conns, idx, idx+1)
	if len(conns) == 0 {
		delete(hub.conns, c.ID())
	} else {
		hub.conns[c.ID()] = conns
	}
	return true

}

func (hub *ConnHub) addConn(c Conn) {
	conns := hub.conns[c.ID()]
	conns = append(conns, c)
	hub.conns[c.ID()] = conns
}
