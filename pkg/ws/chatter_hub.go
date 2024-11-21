package ws

import (
	"context"
	"fmt"
	"sync"
)

// TODO: wrap base context with hub owns context so we can cancel it in close
// also for each transaction i may need to create a new timeout context
// and perhaps pass the context to client send. In the client.Send()
// unregister the client if the context times out

type ChatterHub struct {
	// clients contains all the connected clients
	clients map[string]Client

	register   chan Client
	unregister chan Client
	broadcast  chan Message
	baseCtx    context.Context
	store      HubStore
	done       chan struct{}
	mu         sync.RWMutex
}

func NewChatterHub(ctx context.Context, store HubStore, hub Hub) *ChatterHub {
	return &ChatterHub{
		register:   make(chan Client, 2),
		unregister: make(chan Client, 2),
		broadcast:  make(chan Message, 2),
		clients:    make(map[string]Client),
		baseCtx:    ctx,
		store:      store,
		done:       make(chan struct{}, 1),
	}
}

func (hub *ChatterHub) Register(client Client) {
	if hub == nil {
		return
	}
	hub.register <- client
}

func (hub *ChatterHub) Unregister(client Client) {
	if hub == nil {
		return
	}
	hub.unregister <- client
}

func (hub *ChatterHub) Broadcast(message Message) {
	if hub == nil {
		return
	}
	fmt.Println("broadcasting")
	hub.broadcast <- message
}

func (hub *ChatterHub) Close() error {
	if hub == nil {
		return nil
	}
	hub.done <- struct{}{}

	for _, client := range hub.clients {
		client.Close()
	}

	return nil

}

func (hub *ChatterHub) Start() {
	for {

		select {
		case client := <-hub.register:
			hub.clients[client.ID()] = client

		case client := <-hub.unregister:
			if c, ok := hub.clients[client.ID()]; ok {
				delete(hub.clients, client.ID())
				c.Close()
			}

		case message := <-hub.broadcast:
			// if targetClients is nil, broadcast to all clients
			var targetClients []string
			var err error

			if message.To != "" {

				targetClients, err = hub.store.GetRoomMembers(hub.baseCtx, message.To)
				if err != nil {
					fmt.Printf("store.GetRoomMembers: %v\n", err)
					break
				}
			}

			err = hub.store.PersistMessage(hub.baseCtx, message)
			if err != nil {
				fmt.Printf("store.PersistMessage: %v\n", err)
				break
			}

			if targetClients == nil {
				for _, client := range hub.clients {
					if client.ID() == message.From {
						continue
					}

					client.Send(message)
				}
			} else {
				for _, clientID := range targetClients {
					if clientID == message.From {
						continue
					}

					client, ok := hub.clients[clientID]
					if ok {
						client.Send(message)
					}
				}
			}

		case <-hub.done:
			return

		}

	}
}
