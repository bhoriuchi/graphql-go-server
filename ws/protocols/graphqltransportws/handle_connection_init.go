package graphqltransportws

import "fmt"

// handleConnectionInit handles the connection init
func (c *wsConnection) handleConnectionInit(msg *RawMessage) {
	c.log.Tracef("received CONNECTION_INIT message")

	c.initMx.Lock()

	// check for initialisation requests
	if c.connectionInitReceived {
		c.initMx.Unlock()
		err := fmt.Errorf("too many initialization requests")
		c.setClose(TooManyInitialisationRequests, err.Error())
		c.log.WithField("error", err).WithField("code", c.closeCode).Errorf("failed to init connection")
		return
	}

	// set the initialization
	c.log.Tracef("initialized connection")
	c.connectionInitReceived = true
	c.initMx.Unlock()

	// check for payload
	if msg.HasPayload() {
		payload, err := msg.RecordPayload()
		if err == nil {
			c.connectionParams = payload
		}
	}

	// handle authorization
	var (
		err                error
		payloadOrPermitted interface{}
		payload            interface{}
	)

	if c.config.OnConnect != nil {
		payloadOrPermitted, err = c.config.OnConnect(c)
		if err != nil {
			c.setClose(InternalServerError, err.Error())
			c.log.WithField("error", err).WithField("code", c.closeCode).Errorf("onConnect hook failed")
			return
		}
	}

	switch v := payloadOrPermitted.(type) {
	case bool:
		if !v {
			c.setClose(Forbidden, "Forbidden")
			c.log.WithField("code", c.closeCode).Warnf("onConnect hook returned false")
			return
		}
		payload = nil
	default:
		payload = v
	}

	c.Send(NewAckMessage(payload))

	c.ackMx.Lock()
	c.log.Errorf("acknowledged connection")
	c.acknowledged = true
	c.ackMx.Unlock()
}
