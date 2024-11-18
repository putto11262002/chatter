package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	clientIDHeader = "x-client-id"
)

type TestWSClientAdapter struct {
}

func (*TestWSClientAdapter) Authenticate(r *http.Request) (string, error) {
	id := r.Header.Get("x-client-id")
	if id == "" {
		return "", errors.New("missing x-client-id header")
	}
	return id, nil

}

func wsURL(ts *httptest.Server) string {
	return "ws" + strings.TrimPrefix(ts.URL, "http")
}

type TestHubStoreAdapter struct {
	rooms    map[string][]string
	messages []Message
}

func NewTestHubStoreAdapter() *TestHubStoreAdapter {
	return &TestHubStoreAdapter{
		rooms: make(map[string][]string),
	}
}

func (t *TestHubStoreAdapter) GetRoomMembers(room string) ([]string, error) {
	if members, ok := t.rooms[room]; ok {
		return members, nil
	}
	return nil, errors.New("room not found")
}

func (t *TestHubStoreAdapter) NewMessage(m Message) error {
	t.messages = append(t.messages, m)
	return nil
}

func setUp(t *testing.T, ctx context.Context) (*httptest.Server, *Hub, *TestHubStoreAdapter) {
	storeAdapter := NewTestHubStoreAdapter()
	hub := NewHub(ctx, storeAdapter)
	go hub.Start()

	wsFactory := NewWSClientFactory(hub, &TestWSClientAdapter{})
	mux := http.NewServeMux()
	mux.HandleFunc("/", wsFactory.HandleFunc)

	ts := httptest.NewServer(mux)

	go func() {
		<-ctx.Done()
		ts.Close()
	}()

	return ts, hub, storeAdapter

}

type userClient struct {
	mu   *sync.Mutex
	conn *websocket.Conn
	id   string
}

func (uc *userClient) SendMessageToRoom(message JsonMessage) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("could not marshal message: %v", err)
	}

	return uc.conn.WriteMessage(websocket.TextMessage, jsonMessage)

}

func newClient(ctx context.Context, id string, ts *httptest.Server) (*userClient, *http.Response, error) {
	header := http.Header{}
	header.Set(clientIDHeader, id)

	conn, res, err := websocket.DefaultDialer.DialContext(ctx, wsURL(ts), header)
	if err != nil {
		return nil, res, err
	}

	return &userClient{
		conn: conn,
		mu:   &sync.Mutex{},
		id:   id,
	}, res, nil

}

func Test_Connnect(t *testing.T) {
	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts, hub, _ := setUp(t, context)

	t.Run("connect success", func(t *testing.T) {
		uc, _, err := newClient(context, "1", ts)
		if err != nil {
			t.Fatalf("could not connect to ws: %v", err)
		}
		defer func() {
			uc.conn.Close()
		}()

		assert.Len(t, hub.clients, 1)
		assert.Contains(t, hub.clients, "1")
	})

	t.Run("invalid auth", func(t *testing.T) {
		_, res, err := websocket.DefaultDialer.DialContext(context, wsURL(ts), nil)

		assert.Equal(t, websocket.ErrBadHandshake, err)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		assert.Len(t, hub.clients, 0)

	})

}

// NOTE: not a proper test, there are race conditions
func Test_Disconnect(t *testing.T) {

	context, cancel := context.WithCancel(context.Background())
	defer cancel()
	ts, hub, _ := setUp(t, context)
	t.Run("disconnect via hub unregister ", func(t *testing.T) {
		uc, _, err := newClient(context, "1", ts)
		if err != nil {
			t.Fatalf("could not connect to ws: %v", err)
		}

		assert.Equal(t, 1, len(hub.clients))
		assert.Contains(t, hub.clients, "1")

		go func() {
			defer func() {
				// the hub should have removed the client before
				assert.NotContains(t, hub.clients,
					uc.id,
					"the client should have been removed from the hub before closing the connection")
				uc.conn.Close()
			}()

			for {
				uc.mu.Lock()
				_, _, err := uc.conn.ReadMessage()
				uc.mu.Unlock()

				if err != nil {
					assert.True(t,
						websocket.IsCloseError(err, websocket.CloseNormalClosure))
					return
				}

			}
		}()

		hub.Unregister <- hub.clients[uc.id]

	})

	t.Run("disconnect via client close", func(t *testing.T) {
		uc, _, err := newClient(context, "1", ts)
		if err != nil {
			t.Fatalf("could not connect to ws: %v", err)
		}

		assert.Equal(t, 1, len(hub.clients))
		assert.Contains(t, hub.clients, uc.id)

		go func() {
			defer func() {
				assert.NotContains(t, hub.clients,
					uc.id,
					"the client should have been removed from the hub before closing the connection")

				uc.conn.Close()
			}()

			for {
				uc.mu.Lock()
				_, _, err := uc.conn.ReadMessage()
				uc.mu.Unlock()

				if err != nil {
					assert.True(t, websocket.IsCloseError(err, websocket.CloseNormalClosure))
					return
				}
			}
		}()

		err = uc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

		if err != nil {
			t.Fatalf("could not write close message: %v", err)
		}
	})

	t.Run("disconnect ungracefully from client", func(t *testing.T) {

		uc, _, err := newClient(context, "1", ts)
		if err != nil {
			t.Fatalf("could not connect to ws: %v", err)
		}

		assert.Equal(t, 1, len(hub.clients))
		assert.Contains(t, hub.clients, uc.id)

		go func() {
			defer func() {
				fmt.Printf("clients: %v\n", hub.clients)
				uc.conn.Close()
			}()

			for {
				uc.mu.Lock()
				_, _, err := uc.conn.ReadMessage()
				uc.mu.Unlock()

				if err != nil {
					return
				}
			}
		}()

		uc.conn.Close()

	})
}

func Test_SendMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ts, _, store := setUp(t, ctx)

	uc1, _, err := newClient(ctx, "1", ts)
	if err != nil {
		t.Fatalf("client 1 could not connect to ws: %v", err)
	}

	uc2, _, err := newClient(ctx, "2", ts)
	if err != nil {
		t.Fatalf("client 2 could not connect to ws: %v", err)
	}

	roomID := "room1"
	store.rooms[roomID] = []string{uc1.id, uc2.id}

	var uc2MessageReceived []JsonMessage

	done := make(chan struct{})

	uc1MessageSent := []JsonMessage{
		{
			Data: "hello",
			To:   roomID,
			Type: 1,
		},
		{
			Data: "world",
			To:   roomID,
			Type: 1,
		},
		{
			Data: "!",
			To:   roomID,
			Type: 1,
		},
	}

	go func() {
		defer func() {
			fmt.Printf("closing uc2\n")
			uc2.conn.Close()

			done <- struct{}{}
		}()

		for {
			uc2.mu.Lock()
			var jsonMessage JsonMessage
			err := uc2.conn.ReadJSON(&jsonMessage)
			uc2.mu.Unlock()

			if err != nil {
				return
			}
			fmt.Println("uc2 received message: ", jsonMessage)

			uc2MessageReceived = append(uc2MessageReceived, jsonMessage)
			if len(uc2MessageReceived) == 3 {
				return
			}
		}
	}()

	go func() {
		defer func() {
			fmt.Printf("closing uc1\n")
			uc1.conn.Close()
		}()

		for {
			uc1.mu.Lock()
			var jsonMessage JsonMessage
			err := uc1.conn.ReadJSON(&jsonMessage)
			uc1.mu.Unlock()

			if err != nil {
				return
			}

		}
	}()

	for _, msg := range uc1MessageSent {
		err := uc1.SendMessageToRoom(msg)
		if err != nil {
			t.Fatalf("could not send message: %v", err)
		}
	}

	<-done

	assert.ElementsMatch(t, uc1MessageSent, uc2MessageReceived)

}
