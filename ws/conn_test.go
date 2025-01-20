package ws

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseTimeout = time.Second * 5

var (
	validCloseCodes = []int{websocket.CloseNormalClosure, websocket.CloseGoingAway}
)

func TestClientCloseConnection(t *testing.T) {
	nClients := 3
	h := NewMockHub()
	cf := NewWSConnFactory()
	f := setUpWSFixture(t, h, cf, nClients)

	var disconnectWg sync.WaitGroup
	disconnectWg.Add(nClients)
	disconnectedConn := make(map[string]struct{})
	disconnectedConnMutex := sync.Mutex{}
	h.OnDisconnect(func(a HubActions, c Conn) {
		go func() {
			c.close()
			disconnectedConnMutex.Lock()
			disconnectedConn[c.ID()] = struct{}{}
			disconnectedConnMutex.Unlock()
			disconnectWg.Done()
		}()
	})
	defer f.tearDown()

	// close all connections from the client
	go func() {
		for _, client := range f.clients {
			client.Close()
		}
	}()

	// wait for all the hub to the nofitfy via hub.Disconnect
	ok := waitOrTimeout(baseTimeout, func() {
		disconnectWg.Wait()

	})
	require.True(t, ok, "timeout waiting for hub.Disconnect callbacks")
	assert.Equal(t, nClients, len(disconnectedConn), "unexpected number of connections closed")
	// wait for all the readLoop and writeLoop to exit
	ok = waitOrTimeout(baseTimeout, func() {
		f.serverWg.Wait()
	})
	require.True(t, ok, "timeout waiting for readLoop and writeLoop to exit")
}

func TestHubClosesConnection(t *testing.T) {
	nClients := 3

	h := NewMockHub()
	cf := NewWSConnFactory()
	f := setUpWSFixture(t, h, cf, nClients)
	defer f.tearDown()

	var clientCloseWg sync.WaitGroup
	clientCloseWg.Add(nClients)
	clientCloseEvents := make(map[string]closeEvent)
	clientCloseEventsMutex := sync.Mutex{}
	for _, client := range f.clients {
		client.OnClose(func(ce closeEvent) {
			defer clientCloseWg.Done()
			clientCloseEventsMutex.Lock()
			clientCloseEvents[client.id] = ce
			clientCloseEventsMutex.Unlock()
		})
	}

	// simulate the hub closing the connection
	for _, conn := range f.conns {
		conn.close()
	}

	ok := waitOrTimeout(baseTimeout, func() {
		f.serverWg.Wait()
	})
	require.True(t, ok, "timeout waiting for server-side readLoop and writeLoop to exit")

	// wait for all clients to close
	ok = waitOrTimeout(baseTimeout, func() {
		clientCloseWg.Wait()
	})
	require.True(t, ok, "timeout waiting for all clients to close")
	// assert that clients are closed gracefully
	assert.Len(t, clientCloseEvents, nClients)
	for id, ce := range clientCloseEvents {
		assert.NoErrorf(t, ce.err, "conn %s: unexpected closing error: %v", id, ce.err)
		assert.Equalf(t, wsServer, ce.by, "conn %s: was not closed by the server", id)
		assert.Equalf(t, websocket.CloseNormalClosure, ce.code,
			"conn %s: unexpected close code: expected %d got %d",
			id, websocket.CloseNormalClosure, ce.code)
	}
}

var (
	testPacketType = "test"
)

type testPacketBody struct {
	Sender string `json:"sender"`
	N      int    `json:"n"`
}

func TestClientSendPacket(t *testing.T) {
	// number of packets to send from each client
	nPackets := 10
	nClients := 3

	h := NewMockHub()
	cf := NewWSConnFactory()
	f := setUpWSFixture(t, h, cf, nClients)
	defer f.tearDown()

	var receivedWg sync.WaitGroup
	receivedWg.Add(nClients)
	received := make(map[string][]*Packet)
	receivedMutex := sync.Mutex{}
	h.OnPacket(func(a HubActions, p *Packet) {
		receivedMutex.Lock()
		defer receivedMutex.Unlock()
		received[p.Sender] = append(received[p.Sender], p)
		if len(received[p.Sender]) == nPackets {
			receivedWg.Done()
		}
	})

	// send packets from each client to the hub concurrently
	var sentWg sync.WaitGroup
	sent := make(map[string][]*Packet)
	sentMutex := sync.Mutex{}
	for _, client := range f.clients {
		sentWg.Add(1)
		go func() {
			defer sentWg.Done()
			for i := 0; i < nPackets; i++ {
				b, err := json.Marshal(testPacketBody{Sender: client.id, N: i})
				require.NoError(t, err, "client %s: marshaling packet", client.id)
				packet := &Packet{
					Type:   testPacketType,
					Body:   b,
					Sender: client.id,
				}
				err = client.Send(packet)
				require.NoError(t, err, "client %s: sending message", client.id)
				sentMutex.Lock()
				sent[client.id] = append(sent[client.id], packet)
				sentMutex.Unlock()
			}

		}()
	}

	ok := waitOrTimeout(baseTimeout, func() {
		sentWg.Wait()
	})
	require.True(t, ok, "timeout waiting for all packets to be sent")

	ok = waitOrTimeout(baseTimeout, func() {
		receivedWg.Wait()
	})
	require.True(t, ok, "timeout waiting for all packets to be received")

	for _, client := range f.clients {
		sent := sent[client.id]
		received := received[client.id]

		assert.Len(t, sent, nPackets, "client %s: unexpected number of packets sent", client.id)
		assert.Equal(t, len(sent), len(received),
			"client %s: number of packets sent and received mismatch", client.id)
		assert.ElementsMatch(t, sent, received,
			"client %s: packets sent and received mismatch", client.id)
	}
}

func TestHubSendPacketToClient(t *testing.T) {
	// number of packets to send from each client
	nPackets := 5
	nClients := 3
	h := NewMockHub()
	cf := NewWSConnFactory()
	f := setUpWSFixture(t, h, cf, nClients)
	defer f.tearDown()

	// collect packets received by the clients
	receivedWg := sync.WaitGroup{}
	receivedMutex := sync.Mutex{}
	received := make(map[string][]*Packet)
	for _, client := range f.clients {
		receivedWg.Add(1)
		client.OnPacket(func(p *Packet) {
			receivedMutex.Lock()
			defer receivedMutex.Unlock()
			received[client.id] = append(received[client.id], p)
			if len(received[client.id]) == nPackets {
				receivedWg.Done()
			}
		})
	}

	// simulate sending packet from the hub to the client
	var sentWg sync.WaitGroup
	sent := make(map[string][]*Packet)
	sentMutex := sync.Mutex{}
	for _, conn := range f.conns {
		sentWg.Add(1)
		go func() {
			defer sentWg.Done()
			for i := 0; i < nPackets; i++ {
				b, err := json.Marshal(&testPacketBody{
					N: i,
				})
				require.NoError(t, err, "conn %s: marshaling packet", conn.ID())
				p := &Packet{
					Type: testPacketType,
					Body: b,
				}
				conn.pass() <- p
				sentMutex.Lock()
				sent[conn.ID()] = append(sent[conn.ID()], p)
				sentMutex.Unlock()
			}
		}()
	}

	ok := waitOrTimeout(baseTimeout, func() {
		sentWg.Wait()
	})
	require.True(t, ok, "timeout waiting for all packets to be sent")

	ok = waitOrTimeout(baseTimeout, func() {
		receivedWg.Wait()
	})
	require.True(t, ok, "timeout waiting for all packets to be received")

	// compare packets sent by the hub with packets received by the client
	for _, client := range f.clients {
		sent := sent[client.id]
		received := received[client.id]

		assert.Len(t, sent, nPackets,
			"client %s: unexpected number of packets sent", client.id)
		assert.Equalf(t, len(sent), len(received),
			"client %s: number of packets sent and received mismatch", client.id)
		assert.ElementsMatch(t, sent, received,
			"client %s: packets sent and received mismatch", client.id)
	}
}
