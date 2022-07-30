package graphqltransportws

import "github.com/bhoriuchi/graphql-go-server/ws/protocols"

// handleConnectionInit handles the connection init
func (c *wsConnection) handleConnectionInit(msg *RawMessage) {
	var (
		err                error
		payloadOrPermitted interface{}
		payload            interface{}
	)

	c.initMx.Lock()
	c.log.Tracef("received CONNECTION_INIT message")

	// check for initialisation requests
	if c.connectionInitReceived {
		c.close(TooManyInitialisationRequests, "too many initialization requests")
		c.initMx.Unlock()
		return
	}

	// set the initialization
	c.log.Tracef("initialized connection")
	c.connectionInitReceived = true
	c.initMx.Unlock()

	// check for payload and add it to the connection params
	if msg.HasPayload() {
		payload, err := msg.RecordPayload()
		if err == nil && payload != nil {
			c.connectionParams = payload
		}
	}

	// onConnect hook
	if c.config.OnConnect != nil {
		payloadOrPermitted, err = c.config.OnConnect(c)
		if err != nil {
			c.log.WithError(err).Errorf("onConnect hook failed")
			c.close(InternalServerError, err.Error())
			return
		}
	}

	switch v := payloadOrPermitted.(type) {
	case bool:
		if !v {
			c.log.Errorf("onConnect hook returned false")
			c.close(Forbidden, "Forbidden")
			return
		}
		payload = nil
	default:
		payload = v
	}

	c.ackMx.Lock()
	defer c.ackMx.Unlock()

	c.sendMessage(protocols.OperationMessage{
		Type:    protocols.MsgConnectionAck,
		Payload: payload,
	})
	c.log.Debugf("acknowledged connection")
	c.acknowledged = true
}
