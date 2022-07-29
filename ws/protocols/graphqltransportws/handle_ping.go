package graphqltransportws

// handlePing handles a ping message
func (c *wsConnection) handlePing(msg *RawMessage) {
	c.log.Tracef("received PING message")

	var payload map[string]interface{}

	if msg.HasPayload() {
		payload, _ = msg.RecordPayload()
	}

	if c.config.OnPing != nil {
		c.config.OnPing(c, payload)
		return
	}

	c.log.Tracef("replying to PING message with PONG")
	c.Send(NewPongMessage(payload))
}
