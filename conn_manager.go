package main

type WSClient struct {
}

type WSPacket struct {
}

// WSConnectionManager manages WebSocket connections from clients.
type WSConnectionManager struct {
}

func (cm *WSConnectionManager) Run() {

}

type WSHandler func(cm *WSConnectionManager, p *WSPacket)

type WSRouter struct {
	connectionManager *WSConnectionManager
	routes            map[string]WSHandler
}
