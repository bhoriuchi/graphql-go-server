package graphqltransportws

// handlePing handles a ping message
func (c *wsConnection) handlePing(msg *RawMessage) {
	payload := msg.Payload()

	if c.config.OnPing != nil {
		c.config.OnPing(c, payload)
		return
	}

	c.outgoing <- OperationMessage{
		Type:    MsgPong,
		Payload: payload,
	}
}
