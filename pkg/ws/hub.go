package ws

import (
	"log"
	"sync"
)

type WSHub struct {
	// clients contains all the connected clients
	clients    map[string]Client
	register   chan Client
	unregister chan Client
	request    chan *Request
	res        chan *Response
	done       chan struct{}
	mu         sync.RWMutex
	router     HubRouter
}

func NewChatterHub(router HubRouter) *WSHub {
	return &WSHub{
		router:     router,
		register:   make(chan Client, 1),
		unregister: make(chan Client, 1),
		request:    make(chan *Request, 1),
		res:        make(chan *Response, 1),
		clients:    make(map[string]Client),
		done:       make(chan struct{}, 1),
	}
}

func (hub *WSHub) Register(client Client) {
	if hub == nil {
		return
	}
	hub.register <- client
}

func (hub *WSHub) Unregister(client Client) {
	if hub == nil {
		return
	}
	hub.unregister <- client
}

func (hub *WSHub) Broadcast(packet *Request) {
	if hub == nil {
		return
	}
	hub.request <- packet
}

func (hub *WSHub) Close() error {
	if hub == nil {
		return nil
	}
	hub.done <- struct{}{}

	for _, client := range hub.clients {
		client.Close()
	}

	return nil

}

func (hub *WSHub) Start() {
	for {

		select {
		case client := <-hub.register:
			hub.clients[client.ID()] = client
			log.Printf("client registered: %v", client.ID())

		case client := <-hub.unregister:
			if c, ok := hub.clients[client.ID()]; ok {
				delete(hub.clients, client.ID())
				c.Close()
			}
			log.Printf("client unregistered: %v", client.ID())

		case req := <-hub.request:
			handler := hub.router.GetHandler(req)
			if handler == nil {
				log.Printf("no handler found for request: %v", req.Type)
			}
			go handler(req, hub.res)

		case res := <-hub.res:
			if res.Dest == nil {
				for _, c := range hub.clients {
					c.Send(res)
				}
				continue
			}
			for _, dest := range res.Dest {
				if c, ok := hub.clients[dest]; ok {
					c.Send(res)
				}
			}
		case <-hub.done:
			return

		}

	}
}
