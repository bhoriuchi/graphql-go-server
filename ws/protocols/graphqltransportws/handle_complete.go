package graphqltransportws

func (c *wsConnection) handleComplete(msg *RawMessage) {
	id, err := msg.ID()
	if err != nil {
		c.log.Errorf("%d: %s", BadRequest, err)
		c.setClose(BadRequest, err.Error())
		return
	}

	c.mgr.Unsubscribe(id)
}
