package graphqlws

import (
	"time"
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
