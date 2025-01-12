package ws

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockConn struct {
	outPackets []*OutPacket
	out        chan *OutPacket
	id         string
	done       chan struct{}
	inPackets  []*InPacket
	hub        Hub
	reading    atomic.Bool
	writing    atomic.Bool
	closeDelay time.Duration
}

func NewMockConn(id string, hub Hub) *MockConn {
	return &MockConn{
		id:   id,
		out:  make(chan *OutPacket),
		done: make(chan struct{}),
		hub:  hub,
	}
}

func (c *MockConn) pass() chan<- *OutPacket {
	return c.out
}

func (c *MockConn) close() {

	close(c.done)
}

func (c *MockConn) ID() string {
	return c.id
}

func (c *MockConn) readLoop() {
	c.reading.Store(true)
	defer func() {
		c.reading.Store(false)
	}()
	for {

		select {
		case <-c.done:
			return
		default:
			if len(c.inPackets) > 0 {
				c.hub.pass(c.inPackets[0])
				c.inPackets = c.inPackets[1:]
			}

		}
	}
}

func (c *MockConn) writeLoop() {
	c.writing.Store(true)
	defer func() {
		c.writing.Store(false)
	}()
	for {
		select {
		case p := <-c.out:
			c.outPackets = append(c.outPackets, p)
		case <-c.done:
			if c.closeDelay > 0 {
				time.Sleep(c.closeDelay)
			}
			return
		}
	}
}

type MockConnFactory struct {
	shouldSucceed bool
}

func (f *MockConnFactory) NewConn(w http.ResponseWriter, r *http.Request,
	hub Hub, id string) (Conn, bool) {
	if !f.shouldSucceed {
		return nil, false
	}
	return NewMockConn(id, hub), true
}

type MockAuthenticator struct {
	id            atomic.Int64
	shouldSucceed bool
}

func (a *MockAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) (string, bool) {
	if !a.shouldSucceed {
		return "", false
	}
	id := fmt.Sprintf("%d", a.id.Load())
	a.id.Add(1)
	return id, true
}
func Test_connect(t *testing.T) {

}

func Test_disconnect(t *testing.T) {

}

func TestClose(t *testing.T) {
	t.Parallel()
	t.Run("Close clean up all resources", func(t *testing.T) {
		h := New(&MockConnFactory{shouldSucceed: true}, &MockAuthenticator{shouldSucceed: true})
		// wait for conn to connect
		done := make(chan struct{})
		h.OnConnect = func(ha HubActions, c Conn) error {
			close(done)
			return nil
		}
		h.Start()

		c1 := NewMockConn("1", h)
		h.handleConn(c1)

		<-done
		h.Close()

		// conn readLoop and writeLoop should exit
		assert.False(t, c1.reading.Load())
		assert.False(t, c1.writing.Load())
		// hub should be closed
		assert.False(t, h.ready.Load())
		// conn should be removed from hub
		assert.Len(t, h.conns, 0)
	})

	t.Run("Close with timeout", func(t *testing.T) {
		h := New(&MockConnFactory{shouldSucceed: true}, &MockAuthenticator{shouldSucceed: true})
		h.closeTimeout = time.Millisecond * 100 // Set a short timeout for testing
		// wait for conn to connect
		done := make(chan struct{})
		h.OnConnect = func(ha HubActions, c Conn) error {
			close(done)
			return nil
		}

		h.Start()

		// Configure mock connection with a delay
		c1 := NewMockConn("1", h)
		c1.closeDelay = time.Second // Simulate long close
		h.handleConn(c1)
		<-done

		start := time.Now()
		h.Close()
		elapsed := time.Since(start)

		// Assert that the hub closed within the timeout
		assert.LessOrEqual(t, elapsed, h.closeTimeout+time.Millisecond*50)
	})
}
