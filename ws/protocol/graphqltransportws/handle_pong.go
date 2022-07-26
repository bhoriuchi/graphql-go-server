package graphqltransportws

// handlePong handles a pong message
func (c *wsConnection) handlePong(msg *RawMessage) {
	c.log.Tracef("received PONG message")

	var payload map[string]interface{}

	if msg.HasPayload() {
		payload, _ = msg.RecordPayload()
	}

	if c.config.OnPong != nil {
		c.config.OnPong(c, payload)
	}
}
