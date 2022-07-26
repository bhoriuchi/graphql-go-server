package graphqltransportws

// handlePong handles a pong message
func (c *wsConnection) handlePong(msg *RawMessage) {
	if c.config.OnPong != nil {
		c.config.OnPong(c, msg.Payload())
	}
}
