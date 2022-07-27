package graphqltransportws

import (
	"fmt"

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
func (m RawMessage) Type() (MessageType, error) {
	str, err := m.stringField("type")
	if err != nil {
		return "", err
	}

	return MessageType(str), nil
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
	err := ReMarshal(payload, &r)
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
	err := ReMarshal(payload, r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload")
	}

	return r, nil
}

// NewNextMessage creates a new next message
func NewNextMessage(id string, payload *ExecutionResult) OperationMessage {
	return OperationMessage{
		ID:      id,
		Type:    MsgNext,
		Payload: payload,
	}
}

// NewAckMesage creates a new ack message
func NewAckMessage(payload interface{}) OperationMessage {
	if payload != nil {
		return OperationMessage{
			Type: MsgConnectionAck,
		}
	}

	return OperationMessage{
		Type:    MsgConnectionAck,
		Payload: payload,
	}
}

// NewPingMesage creates a new ping message
func NewPingMessage(payload interface{}) OperationMessage {
	if payload != nil {
		return OperationMessage{
			Type: MsgPing,
		}
	}

	return OperationMessage{
		Type:    MsgPing,
		Payload: payload,
	}
}

func NewSubscribeMessage(id string, payload *SubscribePayload) OperationMessage {
	return OperationMessage{
		ID:      id,
		Type:    MsgSubscribe,
		Payload: payload,
	}
}

func NewErrorMessage(id string, errs gqlerrors.FormattedErrors) OperationMessage {
	return OperationMessage{
		ID:      id,
		Type:    MsgError,
		Payload: errs,
	}
}
