package core

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var (
	rng         = rand.New(rand.NewSource(42))
	connIDKey   = "id"
	usernameKey = "username"
)

type testMessage struct {
	ID int `json:"id"`
	N  int `json:"n"`
}

type wsFixture struct {
	server   *testWSServer
	clients  []*testWSClient
	t        *testing.T
	clientWg sync.WaitGroup
	mu       sync.Mutex
	logger   *slog.Logger
	cm       *Manager
}

func setUpWSFixture(t *testing.T, nClients int) *wsFixture {

	f := &wsFixture{t: t, logger: slog.New(slog.NewTextHandler(os.Stdout, nil))}

	f.cm = NewManager(&DefaultConfig)
	f.cm.idGenerator = &testWSIDGenerator{}
	f.cm.logger = f.logger.WithGroup("server")

	f.server = newTestWSServer(http.HandlerFunc(f.cm.Connect))

	// create clients
	clientLogger := f.logger.WithGroup("client")
	f.mu.Lock()
	for i := 0; i < nClients; i++ {
		f.clients = append(f.clients, NewTestWSClient(i, clientLogger.With(slog.Int("id", i))))
	}
	f.mu.Unlock()

	return f
}

func (f *wsFixture) connectClientsToServer() {
	url := getWSURLFromHTTPURL(f.server.URL)
	var connectWg sync.WaitGroup
	for _, client := range f.clients {
		connectWg.Add(1)
		go func(client *testWSClient) {
			defer connectWg.Done()
			err := client.Connect(url)
			require.NoErrorf(f.t, err, "client %d: failed to connect to server", client.id)
			f.clientWg.Add(1)
			go func() {
				defer f.clientWg.Done()
				client.readLoop()
			}()
		}(client)
	}

	waitOrTimeout(f.t, func() {
		connectWg.Wait()
	}, baseTimeout, "Timeout waiting for clients to open connection")
}

func (f *wsFixture) tearDown() {
	f.mu.Lock()
	for _, client := range f.clients {
		client.Close()
	}
	f.mu.Unlock()
	f.clientWg.Wait()

	f.server.Close()
	f.cm.Close()
}

type testWSServer struct {
	*httptest.Server
}

func newTestWSServer(h http.Handler) *testWSServer {
	ts := &testWSServer{}
	s := httptest.NewServer(h)
	ts.Server = s
	return ts
}

func (s *testWSServer) Close() {
	if s.Server != nil {
		return
	}
	s.Server.Close()
}

type closeEvent struct {
	code      int
	initialor WSPeerType
	err       error
}

type testWSClient struct {
	conn *websocket.Conn
	// closeMsgCode is the code message code the is received from the server.
	// When no close message is received it should be -1.
	onMsg   func(*WSMessage)
	id      int
	onClose func(closeEvent)
	state   atomic.Int64
	logger  *slog.Logger
}

func NewTestWSClient(id int, logger *slog.Logger) *testWSClient {
	return &testWSClient{
		id:      id,
		onClose: func(ce closeEvent) {},
		onMsg:   func(m *WSMessage) {},
		logger:  logger,
	}
}

func (c *testWSClient) OnClose(f func(closeEvent)) {
	c.onClose = f
}

func (c *testWSClient) UpdateState(state WSState) {
	c.state.Store(int64(state))
}

func (c *testWSClient) State() WSState {
	return WSState(c.state.Load())
}

// TODO: change to method name to sendJson
func (c *testWSClient) Send(format int, r io.Reader) error {
	w, err := c.conn.NextWriter(format)
	if err != nil {
		return fmt.Errorf("get writer: %w", err)
	}

	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("copying message to connection writer: %w", err)
	}
	w.Close()
	return nil
}

func (c *testWSClient) Connect(_url string) error {
	c.UpdateState(WSOpening)
	url, err := url.Parse(_url)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	query := url.Query()
	query.Set("id", fmt.Sprintf("%d", c.id))
	url.RawQuery = query.Encode()

	conn, res, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	c.conn = conn
	c.UpdateState(WSOpened)
	return nil
}

// ForceClose closes the underlying connection without sending a close message to the server.
func (c *testWSClient) ForceClose() {
	c.conn.Close()
	c.UpdateState(WSClosed)
}

func (c *testWSClient) readLoop() {
	defer func() {
		c.conn.Close()
		c.UpdateState(WSClosed)
		c.logger.Info("readLoop stopped")
	}()
	for {
		format, r, err := c.conn.NextReader()
		if err != nil {
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) {
				ce := closeEvent{code: closeErr.Code}
				if c.State() == WSClosing {
					ce.initialor = WSClient
				} else {
					ce.initialor = WSServer
				}

				c.logger.Info(fmt.Sprintf("received close message: %v", closeErr))
				c.onClose(ce)
				return
			}
			c.logger.Error(fmt.Sprintf("getting next reader from connection: %v", err))
			c.onClose(closeEvent{initialor: WSClient, err: err})
			return
		}

		msg := NewWSMessage(format)

		if _, err := io.Copy(msg, r); err != nil {
			c.logger.Error(fmt.Sprintf("reading from connection: %v", err))
		} else {
			c.onMsg(msg)
		}
	}
}

func (c *testWSClient) OnMsg(f func(*WSMessage)) {
	c.onMsg = f
}

// Close closes the webscoket connection gracefully.
// It sends a close message to the server and waits for the server to response with a close message.
// It is blocking
func (c *testWSClient) Close() error {
	if c.State() != WSOpened {
		return nil
	}
	c.UpdateState(WSClosing)

	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return fmt.Errorf("send close message: %w", err)
	}
	return nil
}

func getWSURLFromHTTPURL(url string) string {
	return strings.Replace(url, "http://", "ws://", 1)
}

// waitOrTimeout waits for fn to finish or times out.
// fn must close the done channel when it is done.
func waitOrTimeout(t *testing.T, fn func(), timeout time.Duration, s string, args ...interface{}) {
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(timeout):
		require.Failf(t, "timeout", s, args...)
	}

}

type testWSIDGenerator struct {
}

func (ig *testWSIDGenerator) Generate(r *http.Request, conn *websocket.Conn) (int, error) {
	rawID := r.URL.Query().Get("id")
	if rawID == "" {
		return 0, errors.New("id query is empty")
	}
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return 0, fmt.Errorf("parsing id: %w", err)
	}
	return id, nil
}
