package ws

import (
	"errors"
)

var (
	ErrNoMatch = errors.New("no handler match")
)

type HubHandlerFunc func(req *Request, res chan<- *Response)

type SimpleHubRouter struct {
	handlers map[int]HubHandlerFunc
}

func NewSimpleHubRouter() *SimpleHubRouter {
	return &SimpleHubRouter{
		handlers: make(map[int]HubHandlerFunc),
	}
}

func (r *SimpleHubRouter) Handle(k int, h HubHandlerFunc) {
	r.handlers[k] = h
}

func (r *SimpleHubRouter) GetHandler(req *Request) HubHandlerFunc {
	h, ok := r.handlers[req.Type]
	if !ok {
		return nil
	}
	return h
}
