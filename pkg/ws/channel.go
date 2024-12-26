package hub

import (
	"iter"
	"maps"
	"sync"
)

type Channel struct {
	mu      sync.RWMutex
	ID      string
	clients map[string]bool
}

func NewChannel(id string) *Channel {
	return &Channel{
		ID:      id,
		clients: make(map[string]bool),
	}
}

func (c *Channel) subscribe(cID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[cID] = true
}

func (c *Channel) unsubscribe(client *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.clients, client.ID)
}

func (c *Channel) isSubscribed(client *Client) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.clients[client.ID]
	return ok
}

func (c *Channel) Subscribers() iter.Seq[string] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return maps.Keys(c.clients)
}
