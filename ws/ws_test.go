package ws

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const (
	closing = 1
	closed  = 2
	opening = 3
	opened  = 4

	wsServer = 1
	wsClient = 2
)

type closeEvent struct {
	code int
	by   int
	err  error
}

type wsFixture struct {
	server   *TestWSServer
	clients  []*TestWSClient
	conns    map[string]Conn
	t        *testing.T
	serverWg sync.WaitGroup
	clientWg sync.WaitGroup
	mu       sync.Mutex
}

func setUpWSFixture(t *testing.T, h Hub, cf ConnFactory, nClients int) *wsFixture {
	f := &wsFixture{conns: map[string]Conn{}, t: t}
	var clientWG sync.WaitGroup // WaitGroup to track client-side connection callbacks
	var connsWg sync.WaitGroup  // WaitGroup to track server side connection callbacks

	// define server ws handler
	connsWg.Add(nClients)
	f.server = NewTestWSServer(h, cf, func(c Conn) {
		defer connsWg.Done()
		f.mu.Lock()
		f.conns[c.ID()] = c
		f.mu.Unlock()
		f.serverWg.Add(1)
		go func() {
			defer f.serverWg.Done()
			go c.readLoop()
		}()
		f.serverWg.Add(1)
		go func() {
			defer f.serverWg.Done()
			go c.writeLoop()
		}()
	})

	// create clients
	f.clients = make([]*TestWSClient, nClients)
	for i := 0; i < nClients; i++ {
		f.clients[i] = NewTestWSClient(fmt.Sprintf("%d", i))
	}

	url := getWSURLFromHTTPURL(f.server.URL)
	clientWG.Add(nClients) // Wait for all clients to connect
	for i, client := range f.clients {
		go func(client *TestWSClient, i int) {
			defer clientWG.Done()
			err := client.Connect(url)
			require.NoErrorf(t, err, "client %d: failed to connect to server", i)
			f.serverWg.Add(1)
			go func() {
				defer f.serverWg.Done()
				client.readLoop()
			}()
		}(client, i)
	}
	// Wait for all clients to finish connecting
	ok := waitOrTimeout(baseTimeout, func() {
		clientWG.Wait()
	})
	if !ok {
		require.Fail(f.t, "timeout waiting clients to establish connections")
	}

	// Wait for all server-side connection callbacks to complete
	ok = waitOrTimeout(baseTimeout, func() {
		connsWg.Wait()
	})
	if !ok {
		require.Fail(f.t, "timeout waiting for server to accept connections")
	}
	// Assert that all connections have been added to f.conns
	require.Equal(t, nClients, len(f.conns), "not all connections were added to f.conns")

	return f
}

func (f *wsFixture) tearDown() {
	// TODO: check if there is any opening connections if there are opening connections close then
	for _, client := range f.clients {
		client.Close()
	}
	f.clientWg.Wait()
	f.serverWg.Wait()

	f.server.Close()
}

type TestWSServer struct {
	*httptest.Server
	cf     ConnFactory
	onConn func(c Conn)
	h      Hub
}

func NewTestWSServer(h Hub, cf ConnFactory, onConn func(c Conn)) *TestWSServer {
	ts := &TestWSServer{cf: cf, h: h, onConn: onConn}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		conn, ok := cf.NewConn(w, r, ts.h, id)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ts.onConn(conn)

	}))
	ts.Server = s
	return ts
}

func (s *TestWSServer) Close() {
	if s.Server != nil {
		return
	}
	s.Server.Close()
}

type TestWSClient struct {
	conn *websocket.Conn
	// closeMsgCode is the code message code the is received from the server.
	// When no close message is received it should be -1.
	mode     int
	onPacket func(p *Packet)
	id       string
	onClose  func(closeEvent)
}

func NewTestWSClient(id string) *TestWSClient {
	return &TestWSClient{
		id:   id,
		mode: closed,
	}
}

func (c *TestWSClient) OnClose(f func(closeEvent)) {
	c.onClose = f
}

func (c *TestWSClient) Send(p *Packet) error {
	if c.mode != opened {
		return errors.New("connection is not opened")
	}

	err := encodeOutPacket(c.conn.NextWriter, p)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (c *TestWSClient) Connect(_url string) error {
	c.mode = opening
	url, err := url.Parse(_url)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	query := url.Query()
	query.Set("id", c.id)
	url.RawQuery = query.Encode()
	conn, res, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	c.conn = conn
	c.mode = opened
	return nil
}

// ForceClose closes the underlying connection without sending a close message to the server.
func (c *TestWSClient) ForceClose() {
	c.conn.Close()
	c.mode = closed
}

func (c *TestWSClient) readLoop() {
	defer func() {
		c.conn.Close()
		c.mode = closed
	}()
	for {
		mt, r, err := c.conn.NextReader()
		if err != nil {
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) {
				// check close error code
				ce := closeEvent{code: closeErr.Code}
				if c.mode == closing {
					ce.by = wsClient
				} else {
					ce.by = wsServer
				}
				if c.onClose != nil {
					c.onClose(ce)
				}
				return
			}
			if c.onClose != nil {
				c.onClose(closeEvent{by: wsClient, err: err})
			}
			return
		}

		packet, err := partiallyDecodeInPacket(mt, r)
		if err != nil {
			slog.Error(fmt.Sprintf("DecodePacket: %v", err))
		}
		if c.onPacket != nil {
			c.onPacket(packet)
		}
	}
}

func (c *TestWSClient) OnPacket(f func(p *Packet)) {
	c.onPacket = f
}

// Close closes the webscoket connection gracefully.
// It sends a close message to the server and waits for the server to response with a close message.
// It is blocking
func (c *TestWSClient) Close() error {
	c.mode = closing

	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return fmt.Errorf("send close message: %w", err)
	}
	return nil
}
