package core

import (
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

type WSState int
type WSPeerType int

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	WSOpened WSState = iota
	WSOpening
	WSClosed
	WSClosing

	WSServer WSPeerType = iota
	WSClient
)

type ConnIDGenerator interface {
	Generate(r *http.Request, conn *websocket.Conn) (int, error)
}

type AutoIncrementConnIDGenerator struct {
	counter int64
	mu      sync.Mutex
}

func (g *AutoIncrementConnIDGenerator) Generate(_ *http.Request, _ *websocket.Conn) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter++
	return int(g.counter), nil
}

type OnConnect func(string, int)

type OnDisconnect func(string, int)

type ConnManager struct {
	conns   map[string][]*Conn
	mu      sync.RWMutex
	connWg  *sync.WaitGroup
	context context.Context
	logger  *slog.Logger

	onConnect    OnConnect
	onDisconnect OnDisconnect

	receivedEvent chan *Event

	upgrader        websocket.Upgrader
	ReadStreamSize  int
	WriteStreamSize int
}

var defaultUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ManagerOption func(*ConnManager)

func WithCheckOrigin(f func(r *http.Request) bool) ManagerOption {
	return func(m *ConnManager) {
		m.upgrader.CheckOrigin = f
	}
}

func WithLogger(l *slog.Logger) ManagerOption {
	return func(m *ConnManager) {
		m.logger = l
	}
}

func NewConnManager(context context.Context, wg *sync.WaitGroup, logger *slog.Logger, opts ...ManagerOption) *ConnManager {

	m := &ConnManager{
		connWg:          wg,
		conns:           make(map[string][]*Conn),
		logger:          logger,
		context:         context,
		upgrader:        defaultUpgrader,
		ReadStreamSize:  100,
		WriteStreamSize: 100,
		onConnect:       func(string, int) {},
		onDisconnect:    func(string, int) {},
	}

	for _, opt := range opts {
		opt(m)
	}

	m.receivedEvent = make(chan *Event, m.ReadStreamSize)

	return m
}

func (m *ConnManager) Receive() <-chan *Event {
	return m.receivedEvent
}

func (m *ConnManager) OnConnect(f OnConnect) {
	m.onConnect = f
}

func (m *ConnManager) OnDisconnect(f OnDisconnect) {
	m.onDisconnect = f
}

func (m *ConnManager) Connect(username string, w http.ResponseWriter, r *http.Request) error {

	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	m.mu.Lock()
	conns, _ := m.conns[username]
	id := len(conns) + 1
	wsConn := &Conn{
		username:    username,
		id:          id,
		conn:        conn,
		context:     m.context,
		writeStream: make(chan *Event, m.WriteStreamSize),
		readStream:  m.receivedEvent,
		ticker:      time.NewTicker(pingPeriod),
		logger:      m.logger.With(slog.String("connection", fmt.Sprintf("%s:%d", username, id))),
		notifyDisconnect: func() {
			m.disconnect(username, id)
		},
	}
	m.conns[username] = append(conns, wsConn)
	m.mu.Unlock()
	m.connWg.Add(1)
	go func() {
		defer m.connWg.Done()
		wsConn.readLoop()
	}()
	m.connWg.Add(1)
	go func() {
		defer m.connWg.Done()
		wsConn.writeLoop()
	}()

	if m.onConnect != nil {
		m.onConnect(username, id)
	}

	return nil
}

func (m *ConnManager) disconnect(username string, ids ...int) {
	m.mu.Lock()
	conns, ok := m.conns[username]
	if !ok {
		m.mu.Unlock()
		return
	}

	indices := make([]int, 0, len(ids))
	if len(ids) == 0 {
		// disconnect all connections
		for _, c := range conns {
			c.close()
			indices = append(indices, c.id)
		}
		delete(m.conns, username)
	} else {
		// remove specific connections
		// remove from the end to avoid index shifting
		indices := make([]int, 0, len(ids))
		for i := len(conns) - 1; i >= 0; i-- {
			if slices.Contains(ids, conns[i].id) {
				conns[i].close()
				indices = append(indices, i)
			}
		}
		for _, idx := range indices {
			conns = slices.Delete(conns, idx, idx+1)
		}
		if len(conns) == 0 {
			delete(m.conns, username)
		} else {
			m.conns[username] = conns
		}
	}
	m.mu.Unlock()
	for _, id := range indices {
		if m.onDisconnect != nil {
			m.onDisconnect(username, id)
		}
	}
}

func (m *ConnManager) Send(e *Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for conns := range maps.Values(m.conns) {
		for _, conn := range conns {
			conn.writeStream <- e
		}
	}
}

func (m *ConnManager) SendToUsers(e *Event, username ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range username {
		conns, ok := m.conns[u]
		if !ok {
			continue
		}
		for _, conn := range conns {
			conn.writeStream <- e
		}
	}
}

func (m *ConnManager) SendToConn(e *Event, username string, id int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conns := m.conns[username]
	for _, conn := range conns {
		if conn.id == id {
			conn.writeStream <- e
		}
	}
}
