package graphqltransportws

func (c *wsConnection) handleComplete(msg *RawMessage) {
	id, err := msg.ID()
	c.log.WithField("subscriptionId", id).Tracef("received COMPLETE message")

	if err != nil {
		c.setClose(BadRequest, err.Error())
		c.log.WithField("error", err).WithField("code", c.closeCode).Errorf("failed to handle COMPLETE message")
		return
	}

	c.mgr.Unsubscribe(id)
}
