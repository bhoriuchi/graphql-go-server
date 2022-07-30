package protocol

import "github.com/graphql-go/graphql/gqlerrors"

// MessageType is a message type
type MessageType string

const (
	// Common
	MsgConnectionInit MessageType = "connection_init"
	MsgConnectionAck  MessageType = "connection_ack"
	MsgError          MessageType = "error"
	MsgComplete       MessageType = "complete"

	// graphql-ws protocol specific
	MsgPing      MessageType = "ping"
	MsgPong      MessageType = "pong"
	MsgSubscribe MessageType = "subscribe"
	MsgNext      MessageType = "next"

	// graphql-transport-ws specific - deprecated protocol
	MsgKeepAlive           MessageType = "ka"
	MsgConnectionError     MessageType = "connection_error"
	MsgConnectionTerminate MessageType = "connection_terminate"
	MsgStart               MessageType = "start"
	MsgData                MessageType = "data"
	MsgStop                MessageType = "stop"
)

// ExecutionResult result of an execution
type ExecutionResult struct {
	Errors     gqlerrors.FormattedErrors `json:"errors,omitempty"`
	Data       interface{}               `json:"data,omitempty"`
	Path       []interface{}             `json:"path,omitempty"`  // patch result
	Label      *string                   `json:"label,omitempty"` // patch result
	HasNext    *bool                     `json:"hasNext,omitempty"`
	Extensions map[string]interface{}    `json:"extensions,omitempty"`
}

type OperationMessage struct {
	ID      string      `json:"id,omitempty"`
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}
