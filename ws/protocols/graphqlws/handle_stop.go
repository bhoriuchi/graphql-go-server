package graphqlws

import "github.com/bhoriuchi/graphql-go-server/ws/protocols"

func (c *wsConnection) handleStop(msg *protocols.OperationMessage) {
	c.log.Debugf("received STOP message")

	if sub := c.mgr.Unsubscribe(msg.ID); sub != nil {
		sub.CancelFunc()
	}
}
