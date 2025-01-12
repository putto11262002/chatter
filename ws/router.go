package ws

import "fmt"

type PacketHandler func(HubActions, *InPacket) error

type Router struct {
	hub      *ConnHub
	handlers map[string]PacketHandler
}

func NewRouter(hub *ConnHub) *Router {
	return &Router{
		hub:      hub,
		handlers: make(map[string]PacketHandler),
	}
}

func (hub *Router) On(packetType string, h PacketHandler) {
	if _, ok := hub.handlers[packetType]; ok {
		panic(fmt.Sprintf("handler(%s): already exists", packetType))
	}
	hub.handlers[packetType] = h
}

func (r Router) Dispatch(packet *InPacket) {
	h, ok := r.handlers[packet.Type]
	if !ok {
		r.hub.logger.Error(fmt.Sprintf("handler for %s not found", packet.Type))
		return
	}
	func() {
		defer func() {
			if _r := recover(); _r != nil {
				r.hub.logger.Error("handler(%s): %v", packet.Type, _r)
			}

		}()
		err := h(r.hub, packet)
		if err != nil {
			r.hub.logger.Error(
				fmt.Sprintf("handler(%s): %v", packet.Type, err))
		}
	}()
}
