package graphqlws

import (
	"encoding/json"
	"time"
)

// Message is a message type
type Message string

const (
	// Subprotocol
	Subprotocol = "graphql-ws"

	// Enhanced messages
	MsgConnectionAuth Message = "connection_auth"

	// Message types
	MsgConnectionInit      Message = "connection_init"
	MsgConnectionAck       Message = "connection_ack"
	MsgKeepAlive           Message = "ka"
	MsgConnectionError     Message = "connection_error"
	MsgConnectionTerminate Message = "connection_terminate"
	MsgStart               Message = "start"
	MsgData                Message = "data"
	MsgError               Message = "error"
	MsgComplete            Message = "complete"
	MsgStop                Message = "stop"

	// Thresholds
	ReadLimit    = 4096
	WriteTimeout = 10 * time.Second
)

// InitMessagePayload defines the parameters of a connection
// init message.
type InitMessagePayload struct {
	AuthToken     string `json:"authToken"`
	Authorization string `json:"Authorization"`
}

// StartMessagePayload defines the parameters of an operation that
// a client requests to be started.
type StartMessagePayload struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// DataMessagePayload defines the result data of an operation.
type DataMessagePayload struct {
	Data   interface{} `json:"data"`
	Errors []error     `json:"errors"`
}

// OperationMessage represents a GraphQL WebSocket message.
type OperationMessage struct {
	ID      string      `json:"id"`
	Type    Message     `json:"type"`
	Payload interface{} `json:"payload"`
}

func (msg OperationMessage) String() string {
	s, _ := json.Marshal(msg)
	if s != nil {
		return string(s)
	}
	return "<invalid>"
}

func operationMessageForType(messageType Message) OperationMessage {
	return OperationMessage{
		Type: messageType,
	}
}
