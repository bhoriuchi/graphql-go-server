package graphqltransportws

func (c *wsConnection) handleComplete(msg *RawMessage) {
	id, err := msg.ID()
	if err != nil {
		c.logger.Errorf(err.Error())
		c.Close(BadRequest, err.Error())
		return
	}

	c.mgr.Unsubscribe(id)
}
