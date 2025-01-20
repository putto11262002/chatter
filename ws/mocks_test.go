package ws

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type MockConn struct {
	in                chan *Packet
	id                string
	done              chan struct{}
	inPackets         []*Packet
	hub               Hub
	readingExit       chan struct{}
	readingStart      chan struct{}
	writingStart      chan struct{}
	writingExit       chan struct{}
	closeDelay        time.Duration
	connected         chan struct{}
	disconnected      chan struct{}
	onClose           func()
	onReadLoopCalled  func()
	onWriteLoopCalled func()
}

func NewMockConn(id string, hub Hub) *MockConn {
	return &MockConn{
		id:           id,
		in:           make(chan *Packet),
		done:         make(chan struct{}),
		hub:          hub,
		connected:    make(chan struct{}),
		disconnected: make(chan struct{}),
		readingExit:  make(chan struct{}),
		readingStart: make(chan struct{}),
		writingExit:  make(chan struct{}),
		writingStart: make(chan struct{}),
	}
}

func (c *MockConn) OnReadLoopCalled(f func()) {
	c.onReadLoopCalled = f
}

func (c *MockConn) OnWriteLoopCalled(f func()) {
	c.onWriteLoopCalled = f
}

// OnClose
func (c *MockConn) OnCloseCalled(f func()) {
	c.onClose = f
}

func (c *MockConn) pass() chan<- *Packet {
	return c.in
}

func (c *MockConn) close() {
	if c.closeDelay > 0 {
		time.Sleep(c.closeDelay)
	}
	close(c.done)
	if c.onClose != nil {
		c.onClose()
	}
}

func (c *MockConn) ID() string {
	return c.id
}

type MockConnFactory struct {
	shouldFail bool
}

func (f *MockConnFactory) NewConn(w http.ResponseWriter, r *http.Request,
	hub Hub, id string) (Conn, bool) {
	if f.shouldFail {
		return nil, false
	}
	return NewMockConn(id, hub), true
}

func (c *MockConn) readLoop() {
	if c.onReadLoopCalled != nil {
		c.onReadLoopCalled()
	}
	<-c.done
}

func (c *MockConn) writeLoop() {
	if c.onWriteLoopCalled != nil {
		c.onWriteLoopCalled()
	}
	<-c.done
}

type MockAuthenticator struct {
	id         atomic.Int64
	shouldFail bool
}

func (a *MockAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) (string, bool) {
	if a.shouldFail {
		return "", false
	}
	id := fmt.Sprintf("%d", a.id.Load())
	a.id.Add(1)
	return id, true
}

type MockHub struct {
	Hub
	conns                 map[string][]Conn
	disconnectAsyncCalled chan Conn
	connectAsyncCalled    chan Conn
	packetsReceived       map[string][]*Packet
	onPacketReceived      chan *Packet
	mu                    sync.Mutex
	connsWg               sync.WaitGroup
	onDisconnect          func(a HubActions, c Conn)
	onPacket              func(a HubActions, p *Packet)
}

func (h *MockHub) OnPacket(f func(a HubActions, p *Packet)) {
	h.onPacket = f
}

func (h *MockHub) OnDisconnect(f func(a HubActions, c Conn)) {
	h.onDisconnect = f
}

func NewMockHub() *MockHub {
	return &MockHub{
		disconnectAsyncCalled: make(chan Conn, 1),
		connectAsyncCalled:    make(chan Conn, 1),
		packetsReceived:       make(map[string][]*Packet),
		onPacketReceived:      make(chan *Packet, 1),
		conns:                 make(map[string][]Conn),
	}
}

func (h *MockHub) Disconnect(c Conn) {
	if h.onDisconnect != nil {
		h.onDisconnect(h, c)
	}
	// h.mu.Lock()
	// defer h.mu.Unlock()
	// conns, ok := h.conns[c.ID()]
	// if ok {
	// 	idx := slices.Index(conns, c)
	// 	if idx == -1 {
	// 		return
	// 	}
	// 	h.conns[c.ID()] = slices.Delete(conns, idx, idx+1)
	// }
	// c.close()
	// h.disconnectAsyncCalled <- c
}

func (h *MockHub) ConnectAsync(c Conn) {

	h.connsWg.Add(1)
	go func() {
		defer h.connsWg.Done()
		c.readLoop()
	}()
	h.connsWg.Add(1)
	go func() {
		defer h.connsWg.Done()
		c.writeLoop()
	}()
	h.mu.Lock()
	h.conns[c.ID()] = append(h.conns[c.ID()], c)
	h.mu.Unlock()
	h.connectAsyncCalled <- c
}

func (h *MockHub) pass(p *Packet) {
	h.onPacket(h, p)
}

func (h *MockHub) WaitConnsExit() {
	h.connsWg.Wait()
}
