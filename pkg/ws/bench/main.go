package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/putto11262002/chatter/pkg/ws"
)

type HeaderAuth struct{}

type Timestamp struct {
	timestamp     time.Time
	correlationID int
}

func (a *HeaderAuth) Authenticate(r *http.Request) (string, error) {
	clientID := r.Header.Get("x-client-id")
	if clientID == "" {
		return "", fmt.Errorf("authenticated")
	}

	return clientID, nil
}

type client struct {
	id      string
	conn    *websocket.Conn
	encoder Encoder
	decoder Decoder
}

type clientFactory struct {
	serverURL string
	encoder   Encoder
	decoder   Decoder
}

func NewClientFactory(serverURL string) *clientFactory {
	return &clientFactory{
		serverURL: serverURL,
		encoder:   Encoder{},
		decoder:   Decoder{},
	}
}

func (cf *clientFactory) NewClient() (*client, error) {
	header := http.Header{}
	id := fmt.Sprintf("%d", rand.Int())
	header.Set("x-client-id", id)

	conn, _, err := websocket.DefaultDialer.Dial(cf.serverURL, header)
	if err != nil {
		return nil, fmt.Errorf("Dial: %w", err)
	}

	c := &client{
		conn:    conn,
		id:      id,
		decoder: cf.decoder,
		encoder: cf.encoder,
	}

	return c, nil
}

func generateSendMessageRequestPacket(n int, s int) []*ws.Packet {
	packets := make([]*ws.Packet, n)
	for i := 0; i < n; i++ {
		packets[i] = &ws.Packet{
			Type: SendMessageRequest,
			Payload: &SendMessageRequestPayload{
				Message: strings.Repeat("a", s),
			},
		}
	}

	return packets
}

func startReading(client *client, done <-chan struct{}, received chan<- Timestamp) {
	defer client.conn.Close()
	for {
		select {
		case <-done:
			return
		default:
			mt, r, err := client.conn.NextReader()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("client %s: connection closed gracefully", client.id)
					return
				}

				if websocket.IsUnexpectedCloseError(err) {
					log.Printf("client %s: connection closed unexpectedly", client.id)
					return
				}

				log.Printf("client %s: read error: %v", client.id, err)
				return
			}

			packet, err := client.decoder.Decode(r, mt)
			if err != nil {
				log.Printf("client %s: decoder.Decode error: %v", client.id, err)
				continue
			}
			received <- Timestamp{timestamp: time.Now(), correlationID: packet.CorrelationID}
		}
	}
}

func startWriting(client *client, messageSize int, interval time.Duration, done <-chan struct{}, sent chan<- Timestamp, timeout chan<- int) {
	ticker := time.NewTicker(interval)
	payload := SendMessageRequestPayload{
		Message: strings.Repeat("a", messageSize),
	}

	var lastReqCorrelationID int
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w, err := client.conn.NextWriter(client.encoder.MessageType())
			if err != nil {
				log.Printf("client %s: conn.NextWriter error: %v", client.id, err)
				return
			}

			packet := &ws.Packet{
				Type:          SendMessageRequest,
				Payload:       &payload,
				CorrelationID: rand.Int(),
			}
			lastReqCorrelationID = packet.CorrelationID

			if err := client.encoder.Encode(w, packet); err != nil {
				log.Printf("client %s: encoder.Encode error: %v", client.id, err)
				w.Close()
				return
			}
			w.Close()
			sent <- Timestamp{timestamp: time.Now(), correlationID: lastReqCorrelationID}

		case <-done:
			return
		}
	}
}

func main() {
	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Router setup
	router := ws.NewSimpleHubRouter()
	router.Handle(SendMessageRequest, func(req *ws.Request, res chan<- *ws.Response) {
		payload, ok := req.Payload.(*SendMessageRequestPayload)
		if !ok {
			log.Printf("Invalid payload received from %v", req.Src)
			return
		}
		res <- req.NewResponse(BroadcastMessage, &BroadcastMessagePayload{
			Message: payload.Message,
			From:    req.Src,
		}, nil)
	})

	// Hub and server setup
	hub := ws.NewChatterHub(router)
	go hub.Start()

	factory := ws.NewHubClientFactory(hub, &HeaderAuth{}, context, &Encoder{}, &Decoder{})

	server := http.Server{
		Addr: ":8081",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			factory.HandleFunc(w, r)
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server closed: %v", err)
		}
	}()

	// Parameterized client creation
	numberOfClients := 100
	messageSize := 100
	interval := time.Second / 2
	testDuration := 10 * time.Second

	// Create a client factory
	clientFactory := NewClientFactory("ws://localhost:8081")

	// Create done channel for coordinating shutdown
	done := make(chan struct{})

	// Slice to hold clients
	clients := make([]*client, numberOfClients)

	timeout := make(chan int, numberOfClients)
	received := make(chan Timestamp, numberOfClients)
	sent := make(chan Timestamp, numberOfClients)

	request := map[int]time.Time{}
	succesful := map[int]time.Time{}
	failed := map[int]struct{}{}

	// Create and run clients
	for i := 0; i < numberOfClients; i++ {
		client, err := clientFactory.NewClient()
		if err != nil {
			log.Fatalf("Failed to create client %d: %v", i, err)
		}
		clients[i] = client

		// Run each client with reading first
		go startReading(client, done, received)
	}

	// Run each client with writing
	for i := 0; i < numberOfClients; i++ {
		client := clients[i]
		go startWriting(client, messageSize, interval, done, sent, timeout)
	}

	go func() {
		for {
			select {
			case <-done:
				return

			case timestamp := <-sent:
				request[timestamp.correlationID] = timestamp.timestamp
			case timestamp := <-received:
				// check if request is not timed out
				if _, ok := request[timestamp.correlationID]; !ok {
					continue
				}

				succesful[timestamp.correlationID] = timestamp.timestamp

			case correlationID := <-timeout:
				if _, ok := succesful[correlationID]; ok {
					continue
				}
				failed[correlationID] = struct{}{}
			}

		}
	}()

	// Wait for test duration
	timer := time.NewTimer(testDuration)
	<-timer.C

	// Graceful shutdown
	cancel()
	hub.Close()
	close(done)
	server.Shutdown(nil)

	// calculate stats
	totalRequests := len(request)
	totalSuccesful := len(succesful)
	totalFailed := len(failed)
	fmt.Printf("Total requests: %d\n", totalRequests)
	fmt.Printf("Total succesful: %d\n", totalSuccesful)
	fmt.Printf("Total failed: %d\n", totalFailed)

	// 99th percentile latency
	latencies := make([]time.Duration, 0, totalSuccesful)
	for correlationID, requestTime := range request {
		endTime, ok := succesful[correlationID]
		if !ok {
			continue
		}
		latencies = append(latencies, endTime.Sub(requestTime))
	}

	// sort latencies
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	percentile99 := latencies[int(float64(len(latencies))*0.99)]
	fmt.Printf("99th percentile latency: %v\n", percentile99)

}

const (
	SendMessageRequest  int = 1
	SendMessageResponse int = 2
	BroadcastMessage    int = 3
)

type SendMessageRequestPayload struct {
	Message string `json:"message"`
}

type SendMessageResponsePayload struct {
	ID string `json:"id"`
}

type BroadcastMessagePayload struct {
	Message string `json:"message"`
	ID      string `json:"id"`
	From    string `json:"from"`
}

type Encoder struct{}

func (e *Encoder) Encode(w io.Writer, packet *ws.Packet) error {
	payload, err := json.Marshal(packet.Payload)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(struct {
		Type          int             `json:"type"`
		Payload       json.RawMessage `json:"payload"`
		CorrelationID int             `json:"correlationID"`
	}{
		Type:          packet.Type,
		Payload:       json.RawMessage(payload),
		CorrelationID: packet.CorrelationID,
	}); err != nil {
		return err
	}
	return nil
}

func (e *Encoder) MessageType() int {
	return websocket.TextMessage
}

type Decoder struct{}

func (d *Decoder) Decode(r io.Reader, mt int) (*ws.Packet, error) {
	decoder := json.NewDecoder(r)
	var packet struct {
		Type          int             `json:"type"`
		Payload       json.RawMessage `json:"payload"`
		CorrelationID int             `json:"correlationID"`
	}
	if err := decoder.Decode(&packet); err != nil {
		return nil, err
	}

	var payload interface{}
	switch packet.Type {
	case SendMessageRequest:
		payload = &SendMessageRequestPayload{}
	case SendMessageResponse:
		payload = &SendMessageResponsePayload{}
	case BroadcastMessage:
		payload = &BroadcastMessagePayload{}
	default:
		return nil, fmt.Errorf("unknown packet type: %d", packet.Type)
	}

	if err := json.Unmarshal(packet.Payload, payload); err != nil {
		return nil, err
	}

	return &ws.Packet{
		Type:          packet.Type,
		Payload:       payload,
		CorrelationID: packet.CorrelationID,
	}, nil
}

func (d *Decoder) MessageType() int {
	return websocket.TextMessage
}
