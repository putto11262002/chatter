package ws

import (
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/net/context"
)

type Event struct {
	context    context.Context
	dispatcher string
	Type       string           `json:"type"`
	Body       *json.RawMessage `json:"body"`
}

func decodeJsonEvent(r io.Reader) (*Event, error) {
	var event Event
	if err := json.NewDecoder(r).Decode(&event); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}
	return &event, nil
}

func encodeJsonEvent(w io.Writer, event *Event) error {
	if err := json.NewEncoder(w).Encode(event); err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	return nil
}
