package graphqltransportws

import (
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/utils"
	"github.com/bhoriuchi/graphql-go-server/ws/protocols"
	"github.com/graphql-go/graphql/gqlerrors"
)

// RawMessage is the raw message data
type RawMessage map[string]interface{}

// string field extracts a string value from a raw message
func (m RawMessage) stringField(name string) (string, error) {
	rawField, ok := m[name]
	if !ok || rawField == nil {
		return "", fmt.Errorf("message is missing the '%s' property", name)
	}

	strField, ok := rawField.(string)
	if !ok {
		return "", fmt.Errorf("message expects the '%s' property to be a string but got %T", name, rawField)
	}

	if strField == "" {
		return "", fmt.Errorf("message is missing the '%s' property", name)
	}

	return strField, nil
}

// Type validates and extracts the type field value from a raw message
func (m RawMessage) Type() (protocols.MessageType, error) {
	str, err := m.stringField("type")
	if err != nil {
		return "", err
	}

	return protocols.MessageType(str), nil
}

// ID validates and extracts the id field value from a raw message
func (m RawMessage) ID() (string, error) {
	return m.stringField("id")
}

// HasPayload returns true if the payload field exists and is not null
func (m RawMessage) HasPayload() bool {
	p, ok := m["payload"]
	return ok && p != nil
}

// Payload returns the raw payload
func (m RawMessage) Payload() interface{} {
	return m["payload"]
}

// PayloadRecord converts the payload to a record
func (m RawMessage) RecordPayload() (map[string]interface{}, error) {
	payload, ok := m["payload"]
	if !ok || payload == nil {
		return nil, fmt.Errorf("message is missing the 'payload' property")
	}

	r := map[string]interface{}{}
	err := utils.ReMarshal(payload, &r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload")
	}

	return r, nil
}

// SubscribePayload converts the payload to a subscribe payload
func (m RawMessage) SubscribePayload() (*SubscribePayload, error) {
	payload, ok := m["payload"]
	if !ok || payload == nil {
		return nil, fmt.Errorf("message is missing the 'payload' property")
	}

	r := &SubscribePayload{}
	err := utils.ReMarshal(payload, r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload")
	}

	return r, nil
}

// NewNextMessage creates a new next message
func NewNextMessage(id string, payload *ExecutionResult) protocols.OperationMessage {
	return protocols.OperationMessage{
		ID:      id,
		Type:    protocols.MsgNext,
		Payload: payload,
	}
}

// NewAckMesage creates a new ack message
func NewAckMessage(payload interface{}) protocols.OperationMessage {
	if payload != nil {
		return protocols.OperationMessage{
			Type: protocols.MsgConnectionAck,
		}
	}

	return protocols.OperationMessage{
		Type:    protocols.MsgConnectionAck,
		Payload: payload,
	}
}

// NewPingMesage creates a new ping message
func NewPingMessage(payload interface{}) protocols.OperationMessage {
	if payload != nil {
		return protocols.OperationMessage{
			Type: protocols.MsgPing,
		}
	}

	return protocols.OperationMessage{
		Type:    protocols.MsgPing,
		Payload: payload,
	}
}

// NewPongMesage creates a new pong message
func NewPongMessage(payload interface{}) protocols.OperationMessage {
	if payload != nil {
		return protocols.OperationMessage{
			Type: protocols.MsgPong,
		}
	}

	return protocols.OperationMessage{
		Type:    protocols.MsgPong,
		Payload: payload,
	}
}

func NewSubscribeMessage(id string, payload *SubscribePayload) protocols.OperationMessage {
	return protocols.OperationMessage{
		ID:      id,
		Type:    protocols.MsgSubscribe,
		Payload: payload,
	}
}

func NewErrorMessage(id string, errs gqlerrors.FormattedErrors) protocols.OperationMessage {
	return protocols.OperationMessage{
		ID:      id,
		Type:    protocols.MsgError,
		Payload: errs,
	}
}
