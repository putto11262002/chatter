package ws

import (
	"context"
	"log"
)

const (
	ChatMessage MessageType = iota
)

type MessageType uint8

type StoreAapter interface {
	GetRoomMembers(string) ([]string, error)
	NewMessage(Message) error
}

type Hub struct {
	Register   chan *WSClient
	Unregister chan *WSClient
	Incoming   chan Message
	// id to client
	clients map[string]*WSClient
	ctx     context.Context
	store   StoreAapter
}

type Message struct {
	Type MessageType
	Data string
	To   string
	From string
}

func NewHub(ctx context.Context, store StoreAapter) *Hub {
	return &Hub{
		Register:   make(chan *WSClient),
		Unregister: make(chan *WSClient),
		Incoming:   make(chan Message),
		clients:    make(map[string]*WSClient),
		ctx:        ctx,
		store:      store,
	}
}

func (hub *Hub) Start() {
	for {
		select {

		case client := <-hub.Register:
			hub.clients[client.id] = client
			log.Printf("Client %s connected", client.id)

			// Broadcast client connected
		case client := <-hub.Unregister:
			delete(hub.clients, client.id)
			// TODO: do i need to call client.Close() here	?

			log.Printf("Client %s disconnected", client.id)

		case message := <-hub.Incoming:

			targetClients, err := hub.store.GetRoomMembers(message.To)
			// for now just drop the message
			if err != nil {
				log.Printf("store.GetRoomMembers: %v", err)
				continue
			}

			// save the message to the database to guarantee delivery
			err = hub.store.NewMessage(message)
			// for now just drop the message
			if err != nil {
				log.Printf("store.NewMessage: %v", err)
				continue
			}

			for _, c := range targetClients {
				if c == message.From {
					continue
				}

				target, ok := hub.clients[c]
				if ok {
					target.send <- message
				}

			}

			// client is not connected, save message to database

		case <-hub.ctx.Done():
			for _, client := range hub.clients {
				client.Close()
			}

			// some additional cleanup

			return

		}
	}
}
