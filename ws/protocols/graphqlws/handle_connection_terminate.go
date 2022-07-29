package graphqlws

import "github.com/bhoriuchi/graphql-go-server/ws/protocols"

func (c *wsConnection) handleConnectionTerminate(msg *protocols.OperationMessage) {
	c.log.Debugf("received CONNECTION_TERMINATE message")
	c.close(NormalClosure, "")
}
