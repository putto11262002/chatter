package core

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var baseTimeout = time.Second

func waitForallClientsToConnect(f *wsFixture) {
	require.Eventually(f.t, func() bool {
		f.cm.mu.RLock()
		defer f.cm.mu.RUnlock()
		return len(f.clients) == len(f.cm.conns)
	}, baseTimeout, baseTimeout/20, "Timeout waiting for connection to be added to the manager")
}

func TestClientConnectToServer(t *testing.T) {
	f := setUpWSFixture(t, 5)
	defer f.tearDown()

	var numOnConnectCalled int
	f.cm.OnConnect(func(conn *Conn) {
		numOnConnectCalled++
	})

	f.connectClientsToServer()

	require.Eventually(t, func() bool {
		f.cm.mu.RLock()
		defer f.cm.mu.RUnlock()
		return len(f.clients) == len(f.cm.conns)
	}, baseTimeout, baseTimeout/20, "Timeout waiting for connection to be added to the manager")

	assert.Equal(t, len(f.clients), numOnConnectCalled, "OnConnected was not called for all clients.")

	f.cm.mu.Lock()
	assert.Equal(t, len(f.clients), len(f.cm.conns), "Number of clients does not match the connections")

	isConnected := make(map[int]bool)
	for _, client := range f.clients {
		_, ok := f.cm.conns[client.id]
		if ok {
			isConnected[client.id] = true
		}
	}
	assert.Equal(t, len(f.clients), len(isConnected), "Not all clients is connected to the manager")
	f.cm.mu.Unlock()
}

func TestServerDisconnectFromClients(t *testing.T) {
	f := setUpWSFixture(t, 5)
	defer f.tearDown()

	f.connectClientsToServer()

	waitForallClientsToConnect(f)

	disconnected := make(chan int, 1)
	f.cm.OnDisconnect(func(conn *Conn) {
		disconnected <- conn.ID
	})

	// disconnect one connection at a time then check the manager's state
	for _, client := range f.clients {
		f.cm.disconnect(client.id)

		// assert that OnDisconnected should be called
		require.Eventually(t, func() bool {
			select {
			case <-disconnected:
				return true
			default:
				return false
			}
		}, baseTimeout, baseTimeout/20,
			"Timeout waiting for OnDisconnected to be called.")

		// assert that the connection has been removed
		f.cm.mu.RLock()
		require.NotContains(t, f.cm.conns, client.id,
			"connection should be removed from manager")
		f.cm.mu.RUnlock()
	}

	// assert that the clients have been closed
	for _, client := range f.clients {
		require.Equalf(t, WSClosed, client.State(),
			"client: %d has not been close", client.id)
	}
}

func TestClientSendMessage(t *testing.T) {

	nEventsPerConn := 5
	nClients := 5
	f := setUpWSFixture(t, nClients)
	defer f.tearDown()

	// make the receive buffer large enough to hold all events from all connections
	// so write doesn't block
	f.cm.MsgStream = make(chan *WSMessageWithSender, nEventsPerConn*nClients)

	f.connectClientsToServer()

	waitForallClientsToConnect(f)

	sentMsgs := make(map[int][]*testMessage)
	var buf bytes.Buffer
	for _, client := range f.clients {
		for i := 0; i < nEventsPerConn; i++ {
			msg := testMessage{ID: client.id, N: i}
			err := json.NewEncoder(&buf).Encode(msg)
			require.NoError(t, err, "failed to encode message")
			err = client.Send(websocket.TextMessage, &buf)
			require.NoError(t, err, "client failed to sent message")
			sentMsgs[client.id] = append(sentMsgs[client.id], &msg)
			buf.Reset()
		}
	}

	// events that are received on the server
	receivedMsgs := make(map[int][]*testMessage)
	waitOrTimeout(t, func() {
		var receivedMsgsCount int
		for {
			msg := <-f.cm.MsgStream
			receivedMsg := &testMessage{}
			err := json.Unmarshal(msg.Bytes(), receivedMsg)
			require.NoError(t, err, "failed to decode message")

			receivedMsgs[msg.Sender] = append(receivedMsgs[msg.Sender], receivedMsg)
			receivedMsgsCount++
			if receivedMsgsCount == nEventsPerConn*nClients {
				return
			}
		}
	}, baseTimeout, "Timeout waiting for all messages to be received")

	for _, sentMsgs := range sentMsgs {
		receivedMsgs, ok := receivedMsgs[sentMsgs[0].ID]
		require.True(t, ok, "client did not receive any message")
		require.Len(t, receivedMsgs, nEventsPerConn, "client did not receive all messages")
		require.ElementsMatch(t, sentMsgs, receivedMsgs, "received messages do not match sent messages")
	}
}

func TestServerBroadcastMsg(t *testing.T) {
	nMsgs := 5
	nClients := 5
	f := setUpWSFixture(t, nClients)
	defer f.tearDown()

	f.connectClientsToServer()

	waitForallClientsToConnect(f)

	receviedMsgs := NewSyncMap[int, []*testMessage]()
	waitReceiveMsgs := sync.WaitGroup{}
	waitReceiveMsgs.Add(nClients)
	for _, client := range f.clients {
		client.OnMsg(func(m *WSMessage) {
			receviedMsgs.LoadAndStore(client.id, func(value []*testMessage, ok bool) []*testMessage {
				msg := &testMessage{}
				if m.Format != websocket.TextMessage {
					return value
				}
				err := json.Unmarshal(m.Bytes(), msg)
				require.NoError(t, err, "failed to decode message")
				value = append(value, msg)
				if len(value) == nMsgs {
					waitReceiveMsgs.Done()
				}
				return value
			})
		})

	}

	sent := make([]*testMessage, 0, nMsgs)
	for i := 0; i < nMsgs; i++ {
		testMsg := &testMessage{ID: 42, N: i}
		msg := NewWSMessage(websocket.TextMessage)
		err := json.NewEncoder(msg).Encode(testMsg)
		require.NoError(t, err, "failed to encode message")
		t.Logf("Calling emit with: %s", msg.Bytes())
		f.cm.Emit(msg)
		sent = append(sent, testMsg)
	}

	waitOrTimeout(t, func() {
		waitReceiveMsgs.Wait()
	}, baseTimeout, "Timeout waiting for all messages to be received")

	for _, client := range f.clients {
		receivedMsgs, ok := receviedMsgs.Load(client.id)
		require.True(t, ok, "client did not receive any message")
		require.Len(t, receivedMsgs, nMsgs, "client did not receive all messages")
		require.ElementsMatch(t, sent, receivedMsgs, "received messages do not match sent messages")
	}

}
