package connection

import (
	"context"

	"github.com/gorilla/websocket"
)

// Connection is an interface to represent GraphQL WebSocket connections.
// Each connection is associated with an ID that is unique to the server.
type Connection interface {
	// ID returns the unique ID of the connection.
	ID() string

	// Context returns the context for the connection
	Context() context.Context

	// WS the websocket
	WS() *websocket.Conn

	// SendData sends results of executing an operation (typically a
	// subscription) to the client.
	SendData(string, interface{})

	// SendError sends an error to the client.
	SendError(error)
}

// AuthenticateFunc is a function that resolves an auth token
// into a user (or returns an error if that isn't possible).
type AuthenticateFunc func(data map[string]interface{}, conn Connection) (context.Context, error)
