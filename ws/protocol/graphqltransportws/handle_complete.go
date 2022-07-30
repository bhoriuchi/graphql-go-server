package graphqltransportws

func (c *wsConnection) handleComplete(msg *RawMessage) {
	id, err := msg.ID()
	c.log.WithField("subscriptionId", id).Tracef("received COMPLETE message")

	if err != nil {
		c.log.WithError(err).Errorf("failed to handle COMPLETE message")
		c.close(BadRequest, err.Error())
		return
	}

	c.mgr.Unsubscribe(id)
}
