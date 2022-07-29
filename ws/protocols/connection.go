package protocols

import (
	"context"

	"github.com/gorilla/websocket"
)

type Connection interface {
	// ID returns the connection id
	ID() string

	// Context returns the original connection request context
	Context() context.Context

	// WS returns the websocket
	WS() *websocket.Conn

	C() chan OperationMessage

	// ConnectionInitReceived
	ConnectionInitReceived()

	// Acknowledged
	Acknowledged() bool

	// ConnectionParams
	ConnectionParams() interface{}
}

type OperationMessage struct {
	ID      string      `json:"id,omitempty"`
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}
