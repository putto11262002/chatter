package core

import "bytes"

type WSMessageWithSender struct {
	Sender string
	*WSMessage
}

func NewWSMessageWithSender(sender string, m *WSMessage) *WSMessageWithSender {
	return &WSMessageWithSender{
		Sender:    sender,
		WSMessage: m,
	}
}

type WSMessage struct {
	id     int
	Format int
	*bytes.Buffer
}

func NewWSMessage(format int) *WSMessage {
	return &WSMessage{
		Format: format,
		Buffer: bytes.NewBuffer(make([]byte, 0)),
	}
}
