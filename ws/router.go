package ws

import (
	"fmt"
	"log/slog"
)

type PacketHandler func(HubActions, *Packet) error

type Router struct {
	hub      Hub
	handlers map[string]PacketHandler
	logger   *slog.Logger
}

func NewRouter(hub Hub) *Router {
	r := &Router{
		hub:      hub,
		handlers: make(map[string]PacketHandler),
	}
	hub.OnPacket(r.Dispatch)
	return r
}

func (hub *Router) On(packetType string, h PacketHandler) {
	if _, ok := hub.handlers[packetType]; ok {
		panic(fmt.Sprintf("handler(%s): already exists", packetType))
	}
	hub.handlers[packetType] = h
}

func (r Router) Dispatch(actions HubActions, packet *Packet) {
	h, ok := r.handlers[packet.Type]
	if !ok {
		r.logger.Error(fmt.Sprintf("handler for %s not found", packet.Type))
		return
	}
	func() {
		defer func() {
			if _r := recover(); _r != nil {
				r.logger.Error("handler(%s): %v", packet.Type, _r)
			}

		}()
		err := h(r.hub, packet)
		if err != nil {
			r.logger.Error(
				fmt.Sprintf("handler(%s): %v", packet.Type, err))
		}
	}()
}
