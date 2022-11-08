package graphqlws

import "github.com/bhoriuchi/graphql-go-server/ws/protocol"

func (c *wsConnection) handleConnectionTerminate(msg *protocol.OperationMessage) {
	c.log.Debugf("received CONNECTION_TERMINATE message")
	c.close(NormalClosure, "Client requested normal closure: terminate request")
}
