package graphqlws

import (
	"time"

	"github.com/graphql-go/graphql/gqlerrors"
)

type CloseCode int

const (
	// Subprotocol - https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md
	Subprotocol = "graphql-ws"

	// CloseCodes
	NormalClosure       CloseCode = 1000
	ProtocolError       CloseCode = 1002
	UnexpectedCondition CloseCode = 1011

	// Thresholds
	WriteTimeout = 10 * time.Second
)

// ExecutionResult result of an execution
type ExecutionResult struct {
	Errors     gqlerrors.FormattedErrors `json:"errors,omitempty"`
	Data       interface{}               `json:"data,omitempty"`
	Extensions map[string]interface{}    `json:"extensions,omitempty"`
}
