package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorilla/websocket"
)

type InPacket struct {
	context context.Context `json:"-"`
	Sender  string          `json:"-"`
	Type    string          `json:"type"`
	Body    json.RawMessage `json:"body"`
}

type OutPacket struct {
	Receivers []string    `json:"-"`
	Type      string      `json:"type"`
	Body      interface{} `json:"body"`
}

type Packet struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	// Body is the body of the packet.
	// The body is later decode into a specific type in the handler.
	Body json.RawMessage `json:"body"`
}

func partiallyDecodeInPacket(t int, r io.Reader) (*InPacket, error) {
	if t != websocket.TextMessage {
		return nil, fmt.Errorf("unexpected message type: %d", t)
	}

	var packet InPacket
	if err := json.NewDecoder(r).Decode(&packet); err != nil {
		return nil, fmt.Errorf("json.Decoder.Decode: %w", err)
	}
	return &packet, nil
}

func encodeOutPacket(f func(t int) (io.WriteCloser, error), packet *OutPacket) error {
	w, err := f(websocket.TextMessage)
	if err != nil {
		return fmt.Errorf("NextWriter: %w", err)
	}
	defer w.Close()

	if err := json.NewEncoder(w).Encode(packet); err != nil {
		return fmt.Errorf("json.Encoder.Encode: %w", err)
	}

	return nil
}

func encodeVarInt(w *bytes.Buffer, n int) error {
	for {
		// get the last 7 bits
		v := n & 0x7F

		// shift the number to the right by 7 bits
		n >>= 7

		// check if there are any remaining set bits
		hasMore := n != 0

		// if there are more bits to encode, set the most significant bit
		if hasMore {
			v |= 0x80
		}

		// write the byte
		if err := w.WriteByte(byte(v)); err != nil {
			return err
		}

		// if there are no more bits to encode, break the loop
		if !hasMore {
			break
		}
	}
	return nil
}

func decodeVarInt(buf *bytes.Buffer) (int, error) {
	// the amount of bits to shift the value by when add the next byte
	shift := 0
	n := 0
	for {
		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}

		// get the last 7 bits and shift them to the correct position
		v := int(b&0x7F) << shift
		n |= v

		// check the most significant bit if there are more bytes to read
		hasMore := b&0x80 != 0

		if !hasMore {
			break
		}

		// increment the shift by 7 bits
		shift += 7

	}
	return n, nil
}
