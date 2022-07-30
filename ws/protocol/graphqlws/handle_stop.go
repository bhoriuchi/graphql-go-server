package graphqlws

import "github.com/bhoriuchi/graphql-go-server/ws/protocol"

func (c *wsConnection) handleStop(msg *protocol.OperationMessage) {
	c.log.WithField("subscriptionId", msg.ID).Debugf("received STOP message")

	if sub := c.mgr.Unsubscribe(msg.ID); sub != nil {
		sub.CancelFunc()
	}
}
