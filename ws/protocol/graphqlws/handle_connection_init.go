package graphqlws

import (
	"fmt"
	"time"

	"github.com/bhoriuchi/graphql-go-server/ws/protocol"
)

func (c *wsConnection) handleConnectionInit(msg *protocol.OperationMessage) {
	c.log.Tracef("received CONNECTION_INIT message")

	c.initMx.Lock()

	// check for initialisation requests and ignore if already initialized
	if c.connectionInitReceived {
		c.initMx.Unlock()
		errmsg := "received multiple CONNECTION_INIT messages, ignoring duplicates"
		c.log.Errorf(errmsg)
		c.close(UnexpectedCondition, errmsg)
		return
	}

	// handle connection hook
	if c.config.OnConnect != nil {
		maybeContext, err := c.config.OnConnect(c, msg.Payload)
		if err != nil {
			c.log.WithError(err).Errorf("onConnect hook failed")
			c.close(UnexpectedCondition, err.Error())
			return
		}

		switch v := maybeContext.(type) {
		case bool:
			if !v {
				err := fmt.Errorf("prohibited connection")
				c.sendError(msg.ID, protocol.MsgConnectionError, map[string]interface{}{
					"message": err.Error(),
				})
				time.Sleep(10 * time.Millisecond)
				c.close(UnexpectedCondition, err.Error())
				c.initMx.Unlock()
				return
			}
		case map[string]interface{}:
			c.connectionParams = v
		}
	}

	// set the initialization
	c.log.Tracef("connection initialized")
	c.connectionInitReceived = true
	c.initMx.Unlock()

	// send an ack message
	c.sendMessage(protocol.OperationMessage{
		Type: protocol.MsgConnectionAck,
	})

	// setup keep-alives
	if c.config.KeepAlive > 0 {
		c.log.Tracef("sending KEEP_ALIVE message")
		c.sendMessage(protocol.OperationMessage{
			Type: protocol.MsgKeepAlive,
		})

		ticker := time.NewTicker(c.config.KeepAlive)
		go func() {
			for {
				select {
				case <-ticker.C:
					c.log.Tracef("sending KEEP_ALIVE message")
					c.sendMessage(protocol.OperationMessage{
						Type: protocol.MsgKeepAlive,
					})
				case <-c.ka:
					ticker.Stop()
					return
				}
			}
		}()
	}
}
