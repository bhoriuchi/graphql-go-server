package protocol

import (
	"context"

	"github.com/gorilla/websocket"
)

type Context interface {
	// ConnectionID returns the connection id
	ConnectionID() string

	// Context returns the original connection request context
	Context() context.Context

	// WS returns the websocket
	WS() *websocket.Conn

	C() chan OperationMessage

	// ConnectionInitReceived
	ConnectionInitReceived() bool

	// Acknowledged
	Acknowledged() bool

	// ConnectionParams
	ConnectionParams() map[string]interface{}
}
