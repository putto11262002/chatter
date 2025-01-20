package ws

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rng = rand.New(rand.NewSource(42))

type mockConnFixtureConfig struct {
	// n is the total number of connections to create
	n                 int
	duplicationFactor float64 // Must be in the range [0.0, 1.0]
}

type mockConnFixture struct {
	hub    *ConnHub
	conns  []*MockConn
	config mockConnFixtureConfig
	t      *testing.T
}

func (f *mockConnFixture) Teardown() {
	f.hub.Close()
}

func newMockConnFixture(t *testing.T, config mockConnFixtureConfig) *mockConnFixture {
	// Validate configuration
	if config.duplicationFactor < 0.0 || config.duplicationFactor > 1.0 {
		t.Fatalf("duplicationFactor must be between 0.0 and 1.0, got %f", config.duplicationFactor)
	}

	f := &mockConnFixture{
		conns:  make([]*MockConn, 0, config.n),
		config: config,
		t:      t,
		hub:    New(&MockConnFactory{}, &MockAuthenticator{}),
	}

	// Calculate unique and duplicate counts
	numDuplicates := int(float64(config.n) * config.duplicationFactor)
	numUnique := config.n - numDuplicates

	// Create unique connections
	for i := 0; i < numUnique; i++ {
		id := fmt.Sprintf("%d", i)
		f.conns = append(f.conns, NewMockConn(id, f.hub))
	}

	// Create duplicated connections
	for i := 0; i < numDuplicates; i++ {
		id := fmt.Sprintf("%d", rng.Intn(numUnique))
		f.conns = append(f.conns, NewMockConn(id, f.hub))
	}

	f.hub.Start()

	return f
}

func TestConnect(t *testing.T) {
	f := newMockConnFixture(t, mockConnFixtureConfig{n: 10, duplicationFactor: 0.4})
	defer f.Teardown()
	var onConnectCbWG sync.WaitGroup
	onConnectCbWG.Add(f.config.n)
	onConnectCb := make([]Conn, 0, f.config.n)
	f.hub.OnConnect(func(ha HubActions, c Conn) {
		onConnectCb = append(onConnectCb, c)
		onConnectCbWG.Done()
	})

	var onReadLoopCalledWG sync.WaitGroup
	onReadLoopCalledWG.Add(f.config.n)
	onReadLoopCalled := make([]Conn, 0, f.config.n)
	for _, conn := range f.conns {
		conn.OnReadLoopCalled(func() {
			onReadLoopCalled = append(onReadLoopCalled, conn)
			onReadLoopCalledWG.Done()
		})
	}

	var onWriteLoopCalledWg sync.WaitGroup
	onWriteLoopCalledWg.Add(f.config.n)
	onWriteLoopCalled := make([]Conn, 0, f.config.n)
	for _, conn := range f.conns {
		conn.OnWriteLoopCalled(func() {
			onWriteLoopCalled = append(onWriteLoopCalled, conn)
			onWriteLoopCalledWg.Done()
		})
	}

	for _, c := range f.conns {
		f.hub.Connect(c)
	}

	ok := waitOrTimeout(baseTimeout, func() {
		onReadLoopCalledWG.Wait()
	})
	require.True(t, ok, "timeout waiting for readLoops to start")
	assert.ElementsMatch(t, f.conns, onReadLoopCalled, "not all readLoops were started")

	ok = waitOrTimeout(baseTimeout, func() {
		onWriteLoopCalledWg.Wait()
	})
	require.True(t, ok, "timeout waiting for writeLoops to start")
	assert.ElementsMatch(t, f.conns, onWriteLoopCalled, "not all writeLoops were started")

	ok = waitOrTimeout(baseTimeout, func() {
		onConnectCbWG.Wait()
	})
	require.True(t, ok, "timeout waiting for OnConnect callbacks")
	connectedConns := make([]Conn, 0, f.config.n)
	for _, conns := range f.hub.conns {
		connectedConns = append(connectedConns, conns...)
	}
	assert.ElementsMatch(t, f.conns, connectedConns, "not all connections were added to the hub by OnConnect")
	assert.ElementsMatch(t, f.conns, onConnectCb, "not all OnConnect callbacks were called")

}

func TestDisconnect(t *testing.T) {
	f := newMockConnFixture(t, mockConnFixtureConfig{n: 10, duplicationFactor: 0.4})
	defer f.Teardown()

	var onConnectCbWG sync.WaitGroup
	onConnectCbWG.Add(f.config.n)
	onConnectCb := make([]Conn, 0, f.config.n)
	f.hub.onConnect = func(ha HubActions, c Conn) {
		onConnectCb = append(onConnectCb, c)
		onConnectCbWG.Done()
	}

	var onDisconnectCbWG sync.WaitGroup
	onDisconnectCbWG.Add(f.config.n)
	onDisconnectCb := make([]Conn, 0, f.config.n)
	f.hub.onDisconnect = func(ha HubActions, c Conn) {
		onDisconnectCb = append(onDisconnectCb, c)
		onDisconnectCbWG.Done()
	}

	var onCloseCalledWG sync.WaitGroup
	onCloseCalledWG.Add(f.config.n)
	onCloseCalled := make([]Conn, 0, f.config.n)
	for _, conn := range f.conns {
		conn.OnCloseCalled(func() {
			onCloseCalled = append(onCloseCalled, conn)
			onCloseCalledWG.Done()
		})
	}

	for _, conn := range f.conns {
		f.hub.Connect(conn)
	}

	// wait for conns to connect to the hub
	ok := waitOrTimeout(baseTimeout, func() {
		onConnectCbWG.Wait()
	})
	require.True(t, ok, "timeout waiting for OnConnect callbacks")

	// disconnect all connections
	for _, conn := range f.conns {
		f.hub.Disconnect(conn)
	}

	// wait for close callbacks
	ok = waitOrTimeout(baseTimeout, func() {
		onCloseCalledWG.Wait()
	})
	require.True(t, ok, "timeout waiting for OnClose callbacks")
	assert.ElementsMatch(t, f.conns, onCloseCalled, "not all OnClose callbacks were called")

	// wait for disconnect callbacks
	ok = waitOrTimeout(baseTimeout, func() {
		onDisconnectCbWG.Wait()
	})
	require.True(t, ok, "timeout waiting for OnDisconnect callbacks")
	assert.ElementsMatch(t, f.conns, onDisconnectCb, "not all OnDisconnect callbacks were called")
	assert.Len(t, f.hub.conns, 0, "not all connections were removed from the hub by OnDisconnect")
}

func TestClose(t *testing.T) {
	t.Run("Close gracefuly", func(t *testing.T) {
		t.Parallel()
		f := newMockConnFixture(t, mockConnFixtureConfig{n: 10, duplicationFactor: 0.4})
		defer f.Teardown()

		var onConnectCbCallsWG sync.WaitGroup
		onConnectCbCallsWG.Add(f.config.n)
		onConnectCbCalls := make([]Conn, 0, f.config.n)
		f.hub.OnConnect(func(ha HubActions, c Conn) {
			onConnectCbCalls = append(onConnectCbCalls, c)
			onConnectCbCallsWG.Done()
		})

		onDisconnectCbCalls := make([]Conn, 0, f.config.n)
		f.hub.OnDisconnect(func(ha HubActions, c Conn) {
			onDisconnectCbCalls = append(onDisconnectCbCalls, c)
		})

		onCloseCalledCalls := make([]Conn, 0, f.config.n)
		for _, conn := range f.conns {
			conn.OnCloseCalled(func() {
				onCloseCalledCalls = append(onCloseCalledCalls, conn)
			})
		}

		for _, conn := range f.conns {
			f.hub.Connect(conn)
		}

		// wait for conns to connect to the hub
		ok := waitOrTimeout(baseTimeout, func() {
			onConnectCbCallsWG.Wait()
		})
		require.True(t, ok, "timeout waiting for OnConnect callbacks")

		// Close should block until all connections are closed
		f.hub.Close()

		assert.ElementsMatch(t, f.conns, onCloseCalledCalls, "not all OnClose callbacks were called")
		assert.ElementsMatch(t, f.conns, onDisconnectCbCalls, "not all OnDisconnect callbacks were called")
		assert.Len(t, f.hub.conns, 0, "not all connections were removed from the hub")
	})

	t.Run("Close with timeout", func(t *testing.T) {
		t.Parallel()
		f := newMockConnFixture(t, mockConnFixtureConfig{n: 10, duplicationFactor: 0.4})
		defer f.Teardown()
		for _, conn := range f.conns {
			conn.closeDelay = f.hub.closeTimeout * 2
		}

		done := make(chan struct{})
		go func() {
			f.hub.Close()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(f.hub.closeTimeout * 2):
			require.Fail(t, "timeout waiting for hub to close")
		}
	})
}

func generateTestPacket(n int, _type string, sender string) []*Packet {
	packets := make([]*Packet, 0, n)
	for i := 0; i < n; i++ {
		packets = append(packets, &Packet{
			Type:   _type,
			Sender: sender,
			Body:   []byte(fmt.Sprintf("test %d", i)),
		})
	}
	return packets
}

func Test_pass(t *testing.T) {
	nPackets := 10
	f := newMockConnFixture(t, mockConnFixtureConfig{n: 10, duplicationFactor: 0.4})
	defer f.Teardown()

	received := make([]*Packet, 0, 10)
	var receivedWg sync.WaitGroup
	receivedWg.Add(nPackets)
	f.hub.OnPacket(func(ha HubActions, ip *Packet) {
		received = append(received, ip)
		receivedWg.Done()
	})

	sent := generateTestPacket(nPackets, "test", "1")
	var sentWg sync.WaitGroup
	for _, p := range sent {
		sentWg.Add(1)
		go func(p *Packet) {
			f.hub.pass(p)
			sentWg.Done()
		}(p)
	}

	ok := waitOrTimeout(baseTimeout, func() {
		sentWg.Wait()
	})
	require.True(t, ok, "timeout waiting for all packets to be sent")

	ok = waitOrTimeout(baseTimeout, func() {
		receivedWg.Wait()
	})
	require.True(t, ok, "timeout waiting for all packets to be received")

	assert.ElementsMatch(t, sent, received)
}

func Test_sendOrDisconnect(t *testing.T) {
	nPacket := 10
	t.Run("send successful", func(t *testing.T) {
		f := newMockConnFixture(t, mockConnFixtureConfig{n: 10, duplicationFactor: 0.4})
		defer f.Teardown()
		var receivedWg sync.WaitGroup
		received := make(map[string][]*Packet)
		var receivedMu sync.Mutex
		for _, c := range f.conns {
			// use buffered channel with the same length as the packets to avoid blocking.
			c.in = make(chan *Packet, nPacket)
			receivedWg.Add(1)
			go func(c *MockConn) {
				ps := make([]*Packet, 0, nPacket)
				for {
					p := <-c.in
					ps = append(ps, p)
					if len(ps) == nPacket {
						break
					}
				}
				receivedMu.Lock()
				received[c.ID()] = ps
				receivedMu.Unlock()
				receivedWg.Done()
			}(c)
		}

		var sentWg sync.WaitGroup
		sent := make(map[string][]*Packet)
		var sentMu sync.Mutex
		for _, c := range f.conns {
			sentWg.Add(1)
			go func(c *MockConn) {
				packets := generateTestPacket(nPacket, fmt.Sprintf("test %s", c.ID()), "")
				for _, p := range packets {
					f.hub.sendOrDisconnect(c, p)
				}
				sentMu.Lock()
				sent[c.ID()] = packets
				sentMu.Unlock()
				sentWg.Done()
			}(c)
		}

		ok := waitOrTimeout(baseTimeout, func() {
			sentWg.Wait()
		})
		require.True(t, ok, "timeout waiting for all packets to be sent")

		ok = waitOrTimeout(baseTimeout, func() {
			receivedWg.Wait()
		})
		require.True(t, ok, "timeout waiting for all packets to be received")

		for id, sent := range sent {
			received := received[id]
			assert.ElementsMatchf(t, sent, received, "not all packets were received for conn %s", id)
		}

	})

	t.Run("send blocked", func(t *testing.T) {
		f := newMockConnFixture(t, mockConnFixtureConfig{n: 10, duplicationFactor: 0.4})
		defer f.Teardown()

		var sentWg sync.WaitGroup
		sent := make(map[string][]*Packet)
		var sentMu sync.Mutex
		for _, c := range f.conns {
			sentWg.Add(1)
			go func(c *MockConn) {
				packets := generateTestPacket(nPacket, fmt.Sprintf("test %s", c.ID()), "")
				for _, p := range packets {
					f.hub.sendOrDisconnect(c, p)
				}
				sentMu.Lock()
				sent[c.ID()] = packets
				sentMu.Unlock()
				sentWg.Done()
			}(c)
		}

		ok := waitOrTimeout(baseTimeout, func() {
			sentWg.Wait()
		})
		require.True(t, ok, "should not block when sending packets")

	})
}
