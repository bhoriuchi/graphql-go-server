package graphqltransportws

import "fmt"

var (
	ConnectionInitReceived interface{} = "connectionInitReceived"
	ConnectionAckKey       interface{} = "ConnectionAcknowledged"
)

// handleConnectionInit handles the connection init
func (c *wsConnection) handleConnectionInit(msg *RawMessage) {
	c.initMx.Lock()

	// check for initialisation requests
	if c.connectionInitReceived {
		c.initMx.Unlock()
		err := fmt.Errorf("too many initialisation requests")
		c.log.Errorf("%d: %s", TooManyInitialisationRequests, err)
		c.setClose(TooManyInitialisationRequests, err.Error())
		return
	}

	// set the initialization
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
			c.log.Errorf("%d: %s", InternalServerError, err)
			c.setClose(InternalServerError, err.Error())
			return
		}
	}

	switch v := payloadOrPermitted.(type) {
	case bool:
		if !v {
			err := fmt.Errorf("Forbidden")
			c.log.Errorf("%d: %s", err)
			c.setClose(Forbidden, err.Error())
			return
		}
		payload = nil
	default:
		payload = v
	}

	c.Send(NewAckMessage(payload))

	c.ackMx.Lock()
	c.acknowledged = true
	c.ackMx.Unlock()
}
