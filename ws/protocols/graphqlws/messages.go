package graphqlws

import (
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/ws/protocols"
)

// StartMessagePayload defines the parameters of an operation that
// a client requests to be started.
type StartMessagePayload struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

func (s *StartMessagePayload) Validate() error {
	if s.Query == "" {
		return fmt.Errorf("no query specified in START message payload")
	}

	return nil
}

type StartMessage struct {
	ID      string                `json:"id,omitempty"`
	Type    protocols.MessageType `json:"type"`
	Payload StartMessagePayload   `json:"payload,omitempty"`
}
