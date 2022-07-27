package graphqltransportws

// handlePing handles a ping message
func (c *wsConnection) handlePing(msg *RawMessage) {
	var payload map[string]interface{}

	if msg.HasPayload() {
		payload, _ = msg.RecordPayload()
	}

	if c.config.OnPing != nil {
		c.config.OnPing(c, payload)
		return
	}

	c.Send(NewPingMessage(payload))
}
