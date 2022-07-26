package graphqltransportws

import (
	"encoding/json"
	"fmt"
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

// OperationMessage used for
// * ConnectionInit
// * ConnectionAck
// * Ping
// * Pong
// * Subscribe
// * Error
// * Complete
type OperationMessage struct {
	ID      string      `json:"id,omitempty"`
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Validate validates the message
func (m *OperationMessage) Validate() error {
	if m.Type == "" {
		return fmt.Errorf("message is missing the 'type' property")
	}

	switch m.Type {
	case MsgConnectionInit, MsgConnectionAck, MsgPing, MsgPong:
		if m.Payload != nil && !isObject(m) {
			return fmt.Errorf("%q message expects the 'payload' property to be an object or missing, but got %q", m.Type, m.Payload)
		}

	case MsgSubscribe:
		if m.ID == "" {
			return fmt.Errorf("%q message requires a non-empty 'id' property", m.Type)
		}

		// convert the payload and validate it
		payload := &SubscribePayload{}
		if err := ReMarshal(m.Payload, payload); err != nil {
			return fmt.Errorf("%q message expects the 'payload' property to be an object but got %T", m.Type, m.Payload)
		}

		if err := payload.Validate(); err != nil {
			return err
		}

	case MsgNext:
		if m.ID == "" {
			return fmt.Errorf("%q message requires a non-empty 'id' property", m.Type)
		}

		if !isObject(m.Payload) {
			return fmt.Errorf("%q message expects the 'payload' property to be an object, but got %T", m.Type, m.Payload)
		}

	case MsgError:
		if m.ID == "" {
			return fmt.Errorf("%q message requires a non-empty 'id' property", m.Type)
		}

		if !areGraphQLErrors(m.Payload) {
			j, _ := json.Marshal(m.Payload)
			return fmt.Errorf("%q message expects the 'payload' property to be an array of GraphQL errors, but got %s", m.Type, string(j))
		}

	case MsgComplete:
		if m.ID == "" {
			return fmt.Errorf("%q message requires a non-empty 'id' property", m.Type)
		}

	default:
		return fmt.Errorf("invalid message 'type' property %q", m.Type)
	}

	return nil
}

// SubscribePayload payload for a subscribe operation
type SubscribePayload struct {
	OperationName *string `json:"operationName"`
	Query         string  `json:"query"`
	Variables     Record  `json:"variables"`
	Extensions    Record  `json:"extensions"`
}

// Validates a subscribe payload
func (p SubscribePayload) Validate() error {
	return nil
}

// ExecutionResult result of an execution
type ExecutionResult struct {
	Errors     []*gqlerrors.Error     `json:"errors"`
	Data       interface{}            `json:"data"`
	HasNext    bool                   `json:"hasNext"`
	Extensions map[string]interface{} `json:"extensions"`
}

// ExecutionPatchResult result of an execution patch
type ExecutionPatchResult struct {
	Errors     []*gqlerrors.Error     `json:"errors"`
	Data       interface{}            `json:"data"`
	Path       []interface{}          `json:"path"`
	Label      string                 `json:"label"`
	HasNext    bool                   `json:"hasNext"`
	Extensions map[string]interface{} `json:"extensions"`
}
