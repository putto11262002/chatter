package ws

// HubActions define a collection of actions that a handler can perform on the hub.
// It is used to prevent the handler from directly accessing the hub.
type HubActions interface {
}

// BroadcastToClients broadcasts a message to a list of clients.
func (hub *ConnHub) BroadcastToClients(res *OutPacket, ids ...string) {
	for _, id := range ids {
		conns, ok := hub.conns[id]
		if !ok {
			continue
		}
		for _, conn := range conns {
			hub.sendOrDisconnect(conn, res)
		}
	}
}

// Broadcast broadcasts a message to all the clients that are connected to the hub.
func (hub *ConnHub) Broadcast(packet *OutPacket) {
	for _, conns := range hub.conns {
		for _, conn := range conns {
			hub.sendOrDisconnect(conn, packet)
		}
	}
}
