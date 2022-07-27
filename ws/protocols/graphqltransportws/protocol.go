package graphqltransportws

import (
	"encoding/json"
	"time"

	"github.com/graphql-go/graphql/gqlerrors"
)

// MessageType is a message type
type MessageType string

// CloseCode a closing code
type CloseCode int

const (
	// Subprotocol
	Subprotocol = "graphql-transport-ws"

	// Message types
	MsgConnectionInit MessageType = "connection_init"
	MsgConnectionAck  MessageType = "connection_ack"
	MsgPing           MessageType = "ping"
	MsgPong           MessageType = "pong"
	MsgSubscribe      MessageType = "subscribe"
	MsgNext           MessageType = "next"
	MsgError          MessageType = "error"
	MsgComplete       MessageType = "complete"

	// Close codes
	Noop                             CloseCode = -1
	NormalClosure                    CloseCode = 1000
	InternalServerError              CloseCode = 4500
	InternalClientError              CloseCode = 4005
	BadRequest                       CloseCode = 4400
	BadResponse                      CloseCode = 4004
	Unauthorized                     CloseCode = 4401
	Forbidden                        CloseCode = 4403
	SubprotocolNotAcceptable         CloseCode = 4406
	ConnectionInitialisationTimeout  CloseCode = 4408
	ConnectionAcknowledgementTimeout CloseCode = 4504
	SubscriberAlreadyExists          CloseCode = 4409
	TooManyInitialisationRequests    CloseCode = 4429

	// Thresholds
	ReadLimit    = 4096
	WriteTimeout = 10 * time.Second
)

// OperationMessage
type OperationMessage struct {
	ID      string      `json:"id,omitempty"`
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type CompleteMessage struct {
	ID   string      `json:"id"`
	Type MessageType `json:"type"`
}

// ErrorMessage
type ErrorMessage struct {
	ID      string                    `json:"id"`
	Type    MessageType               `json:"type"`
	Payload gqlerrors.FormattedErrors `json:"payload"`
}

type SubscribeMessage struct {
	ID      string           `json:"id"`
	Type    MessageType      `json:"type"`
	Payload SubscribePayload `json:"payload"`
}

// SubscribePayload payload for a subscribe operation
type SubscribePayload struct {
	OperationName *string                `json:"operationName"`
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	Extensions    map[string]interface{} `json:"extensions"`
}

type NextMessage struct {
	ID      string          `json:"id"`
	Type    MessageType     `json:"type"`
	Payload ExecutionResult `json:"payload"`
}

// ExecutionResult result of an execution
type ExecutionResult struct {
	Errors     gqlerrors.FormattedErrors `json:"errors,omitempty"`
	Data       interface{}               `json:"data,omitempty"`
	Path       []interface{}             `json:"path,omitempty"`  // patch result
	Label      *string                   `json:"label,omitempty"` // patch result
	HasNext    *bool                     `json:"hasNext,omitempty"`
	Extensions map[string]interface{}    `json:"extensions,omitempty"`
}

// ReMarshal converts one type to another
func ReMarshal(in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
