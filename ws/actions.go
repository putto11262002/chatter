package ws

// HubActions define a collection of actions that a handler can perform on the hub.
// It is used to prevent the handler from directly accessing the hub.
type HubActions interface {
}

// BroadcastToClients broadcasts a message to a list of clients.
func (hub *ConnHub) BroadcastToClients(res *Packet, ids ...string) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for _, id := range ids {
		_, ok := hub.conns[id]
		if !ok {
			continue
		}
		for i := len(hub.conns[id]) - 1; i >= 0; i-- {
			hub.sendOrDisconnect(hub.conns[id][i], res)
		}
	}
}

// Broadcast broadcasts a message to all the clients that are connected to the hub.
func (hub *ConnHub) Broadcast(packet *Packet) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for id := range hub.conns {
		for i := len(hub.conns[id]) - 1; i >= 0; i-- {
			hub.sendOrDisconnect(hub.conns[id][i], packet)
		}
	}
}
