package graphqlws

import (
	"fmt"
	"time"

	"github.com/bhoriuchi/graphql-go-server/ws/protocols"
)

func (c *wsConnection) handleConnectionInit(msg *protocols.OperationMessage) {
	c.log.Tracef("received CONNECTION_INIT message")

	c.initMx.Lock()

	// check for initialisation requests and ignore if already initialized
	if c.connectionInitReceived {
		c.initMx.Unlock()
		c.log.Warnf("received multiple CONNECTION_INIT messages, ignoring duplicates")
		return
	}

	// handle connection hook
	if c.config.OnConnect != nil {
		maybeContext := c.config.OnConnect(c, msg.Payload)
		switch v := maybeContext.(type) {
		case bool:
			if !v {
				err := fmt.Errorf("prohibited connection")
				c.sendError(msg.ID, protocols.MsgConnectionError, map[string]interface{}{
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
	c.sendMessage(protocols.OperationMessage{
		Type: protocols.MsgConnectionAck,
	})

	// setup keep-alives
	if c.config.KeepAlive > 0 {
		c.log.Tracef("sending KEEP_ALIVE message")
		c.sendMessage(protocols.OperationMessage{
			Type: protocols.MsgKeepAlive,
		})

		ticker := time.NewTicker(c.config.KeepAlive)
		go func() {
			for {
				select {
				case <-ticker.C:
					c.log.Tracef("sending KEEP_ALIVE message")
					c.sendMessage(protocols.OperationMessage{
						Type: protocols.MsgKeepAlive,
					})
				case <-c.ka:
					ticker.Stop()
					return
				}
			}
		}()
	}
}
