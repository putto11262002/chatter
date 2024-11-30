package ws

import (
	"context"
	"io"
	"net/http"
)

// Reguest and Response are just server-side abstractions for packets. They are
// used to distinguish between incoming and outgoing packets and to provide extra
// metadata like the context and the correlation ID.

// Request represents an incoming packet.
type Request struct {
	Payload       interface{}
	Type          int
	Context       context.Context
	CorrelationID int
	Src           string
}

func NewRequest(ctx context.Context, packet Packet, src string) *Request {
	return &Request{
		Payload:       packet.Payload,
		Type:          packet.Type,
		CorrelationID: packet.CorrelationID,
		Src:           src,
		Context:       ctx,
	}
}

// NewResponse creates a new response packet corresponding to the request.
// Multiple responses can be created from a single request.
func (r *Request) NewResponse(t int, payload interface{}, dest []string) *Response {
	return &Response{
		Payload:       payload,
		Type:          t,
		Context:       r.Context,
		CorrelationID: r.CorrelationID,
		Dest:          dest,
	}
}

// Response represents an outgoing packet.
type Response struct {
	Payload       interface{}
	Type          int
	Context       context.Context
	CorrelationID int
	Dest          []string
}

func (r *Response) Packet() *Packet {
	return &Packet{
		Type:          r.Type,
		Payload:       r.Payload,
		CorrelationID: r.CorrelationID,
	}
}

type Client interface {
	ID() string
	Send(*Response)
	Close() error
}

type Hub interface {
	Register(client Client)
	Unregister(client Client)
	Broadcast(*Request)
	Close() error
	Start()
}

type HubRouter interface {
	Handle(k int, h HubHandlerFunc)
	GetHandler(req *Request) HubHandlerFunc
}

// Packet represents a packet that is sent over the wire.
type Packet struct {
	Type          int         `json:"type"`
	Payload       interface{} `json:"payload"`
	CorrelationID int         `json:"correlationID"`
}

type PacketDecoder interface {
	Decode(r io.Reader, mt int) (*Packet, error)
}

type PacketEncoder interface {
	MessageType() int
	Encode(w io.Writer, packet *Packet) error
}

type AuthAdapter interface {
	Authenticate(r *http.Request) (string, error)
}

type ClientFactory interface {
	NewClient(w http.ResponseWriter, r *http.Request) (Client, error)
}
