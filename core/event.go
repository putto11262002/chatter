package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

type Event struct {
	ID         int             `json:"-"`
	Dispatcher string          `json:"-"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
}

func (e Event) String() string {
	return fmt.Sprintf("Event{ID: %d, Dispatcher: %s, Type: %s, Payload.Size: %d}", e.ID, e.Dispatcher, e.Type, len(e.Payload))
}

func EncodeEvent(w io.Writer, e *Event) error {
	if err := json.NewEncoder(w).Encode(e); err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	return nil
}

func DecodeEvent(r io.Reader, e *Event) error {
	if err := json.NewDecoder(r).Decode(e); err != nil {
		return fmt.Errorf("decode event: %w", err)
	}
	return nil
}

type EventTransport interface {
	Send(event *Event)
	SendToUsers(event *Event, usernames ...string)
	Receive() <-chan *Event
}

type EventHandler func(context.Context, *Event) error

type EventRouter struct {
	listeners map[string]EventHandler
	ctx       context.Context
	transport EventTransport
	logger    *slog.Logger
}

func NewEventRouter(ctx context.Context, logger *slog.Logger, transport EventTransport) *EventRouter {
	return &EventRouter{
		listeners: make(map[string]EventHandler),
		ctx:       ctx,
		transport: transport,
		logger:    logger,
	}
}

func (em *EventRouter) Listen(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case e := <-em.transport.Receive():
			em.logger.Debug(fmt.Sprintf("received: %v", e))
			if handlers, ok := em.listeners[e.Type]; ok {
				go func() {
					if err := handlers(em.ctx, e); err != nil {
						em.logger.Error(fmt.Sprintf("%s handler: %s", e.Type, err))
					}
				}()
			}

		}

	}
}

func (em *EventRouter) On(eventName string, handler EventHandler) {
	em.listeners[eventName] = handler
}

// Emit sends an event to the specified targets.
func (em *EventRouter) Emit(t string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	e := &Event{
		Type:    t,
		Payload: b,
	}
	em.transport.Send(e)
	return nil
}

func (em *EventRouter) EmitTo(t string, payload interface{}, usernames ...string) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	e := &Event{
		Type:    t,
		Payload: b,
	}

	em.transport.SendToUsers(e, usernames...)
	return nil
}
