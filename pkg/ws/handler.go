package hub

import "context"

type Request struct {
	Packet  *Packet
	Sender  *Client
	Hub     *Hub
	context context.Context
}

func NewContext(packet *Packet, sender *Client, hub *Hub, context context.Context) *Request {
	return &Request{
		Packet:  packet,
		Sender:  sender,
		Hub:     hub,
		context: context,
	}
}

func (ctx *Request) Context() context.Context {
	return ctx.context
}

func (ctx *Request) WithContext(context context.Context) {
	ctx.context = context
}

type Handler func(*Request) error
