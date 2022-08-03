package graphqltransportws

import (
	"time"

	"github.com/bhoriuchi/graphql-go-server/ws/protocol"
	"github.com/graphql-go/graphql/gqlerrors"
)

// CloseCode a closing code
type CloseCode int

const (
	// Subprotocol - https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md
	Subprotocol = "graphql-transport-ws"

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
	WriteTimeout = 10 * time.Second
)

type CompleteMessage struct {
	ID   string               `json:"id"`
	Type protocol.MessageType `json:"type"`
}

// ErrorMessage
type ErrorMessage struct {
	ID      string                    `json:"id"`
	Type    protocol.MessageType      `json:"type"`
	Payload gqlerrors.FormattedErrors `json:"payload"`
}

type SubscribeMessage struct {
	ID      string               `json:"id"`
	Type    protocol.MessageType `json:"type"`
	Payload SubscribePayload     `json:"payload"`
}

// SubscribePayload payload for a subscribe operation
type SubscribePayload struct {
	OperationName string                 `json:"operationName"`
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	Extensions    map[string]interface{} `json:"extensions"`
}

type NextMessage struct {
	ID      string                   `json:"id"`
	Type    protocol.MessageType     `json:"type"`
	Payload protocol.ExecutionResult `json:"payload"`
}
