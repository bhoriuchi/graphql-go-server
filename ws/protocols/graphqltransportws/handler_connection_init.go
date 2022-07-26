package graphqltransportws

var (
	ConnectionInitReceived interface{} = "connectionInitReceived"
	ConnectionAckKey       interface{} = "ConnectionAcknowledged"
)

// handleConnectionInit handles the connection init
func (c *wsConnection) handleConnectionInit(msg *RawMessage) {
	c.mx.Lock()

	// check for initialisation requests
	if c.ctx.ConnectionInitReceived() {
		c.mx.Unlock()
		c.logger.Errorf("%d: Too many initialisation requests", TooManyInitialisationRequests)
		c.Close(TooManyInitialisationRequests, "Too many initialisation requests")
		return
	}

	// set the initialization
	c.ctx["connectionInitReceived"] = true
	c.mx.Unlock()

	// check for payload
	if msg.HasPayload() {
		payload, err := msg.RecordPayload()
		if err == nil {
			c.ctx["connectionParams"] = payload
		}
	}

	// handle authorization
	var (
		err                error
		payloadOrPermitted interface{}
		payload            interface{}
	)

	if c.config.OnConnect != nil {
		payloadOrPermitted, err = c.config.OnConnect(c, c.ctx)
		if err != nil {
			c.logger.Errorf(err.Error())
			c.Close(InternalServerError, err.Error())
			return
		}
	}

	switch v := payloadOrPermitted.(type) {
	case bool:
		if !v {
			c.logger.Errorf("Forbidden")
			c.Close(Forbidden, "Forbidden")
			return
		}
		payload = nil
	default:
		payload = v
	}

	c.outgoing <- OperationMessage{
		Type:    MsgConnectionAck,
		Payload: payload,
	}

	c.ctx["acknowledged"] = true
}
