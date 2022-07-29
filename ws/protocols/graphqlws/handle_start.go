package graphqlws

import (
	"context"
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/metadata"
	"github.com/bhoriuchi/graphql-go-server/utils"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/bhoriuchi/graphql-go-server/ws/protocols"
	"github.com/graphql-go/graphql"
)

func (c *wsConnection) handleStart(msg *protocols.OperationMessage) {
	var err error
	c.log.Debugf("received START message")

	if !c.ConnectionInitReceived() {
		err := fmt.Errorf("attempted start operation on uninitialized connection")
		c.sendError(msg.ID, protocols.MsgConnectionError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	if msg.ID == "" {
		msgtxt := "message contains no ID"
		c.log.Errorf(msgtxt)
		c.sendError("", protocols.MsgError, map[string]interface{}{
			"message": msgtxt,
		})
		return
	}

	// if we already have a subscription with this id, unsubscribe from it first
	if c.mgr.HasSubscription(msg.ID) {
		c.mgr.Unsubscribe(msg.ID)
	}

	payload := &StartMessagePayload{}
	if err := utils.ReMarshal(msg.Payload, payload); err != nil {
		msgtxt := fmt.Sprintf("failed to parse start payload: %s", err)
		c.log.WithError(err).Errorf("failed to parse start payload")
		c.sendError(msg.ID, protocols.MsgError, map[string]interface{}{
			"message": msgtxt,
		})
		return
	}

	if err := payload.Validate(); err != nil {
		c.log.WithError(err).Errorf("start payload validation error")
		c.sendError(msg.ID, protocols.MsgError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	// add the connection to the metadata context
	metaCtx := metadata.NewWithContext(context.Background())
	metadata.Set(metaCtx, "connection", c)
	ctx, cancelFunc := context.WithCancel(metaCtx)

	rootObject := map[string]interface{}{}
	if c.config.RootValueFunc != nil {
		rootObject = c.config.RootValueFunc(c.ctx, c.config.Request)
	}

	args := graphql.Params{
		Schema:         *c.schema,
		RequestString:  payload.Query,
		VariableValues: payload.Variables,
		Context:        ctx,
		OperationName:  payload.OperationName,
		RootObject:     rootObject,
	}

	// handle onOperation hook if set
	if c.config.OnOperation != nil {
		parsedMessage := StartMessage{
			ID:      msg.ID,
			Type:    msg.Type,
			Payload: *payload,
		}

		if args, err = c.config.OnOperation(c, parsedMessage, args); err != nil {
			c.log.WithError(err).Errorf("onOperation hook failed")
			c.sendError(msg.ID, protocols.MsgError, map[string]interface{}{
				"message": err.Error(),
			})
			cancelFunc()
			return
		}
	}

	subName := args.OperationName
	if subName == "" {
		subName = "Unnamed Subscription"
	}

	resultChannel := graphql.Subscribe(args)
	if err := c.mgr.Subscribe(&manager.Subscription{
		Channel:       resultChannel,
		ConnectionID:  c.ID(),
		OperationID:   msg.ID,
		OperationName: subName,
		Context:       ctx,
		CancelFunc:    cancelFunc,
	}); err != nil {
		c.log.WithError(err).Errorf("subscribe operation failed")
		c.sendError(msg.ID, protocols.MsgError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	c.log.Tracef("subscription %q SUBSCRIBED", subName)
	go func() {
		// ensure subscription is always unsubscribed when finished
		defer func() {
			c.mgr.Unsubscribe(msg.ID)
			c.log.Debugf("subscription %q UNSUBSCRIBED", subName)

			if c.config.OnOperationComplete != nil {
				c.config.OnOperationComplete(c, msg.ID)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				c.log.Tracef("exiting subscription %q", subName)
				return

			case res, more := <-resultChannel:
				if !more {
					c.log.Tracef("subscription %q has no more messages, unsubscribing", subName)

					// send the complete message if the subscription is active
					if c.mgr.HasSubscription(msg.ID) {
						c.sendMessage(protocols.OperationMessage{
							ID:   msg.ID,
							Type: protocols.MsgComplete,
						})
					}

					return
				}

				// if the response is all errors, close the result and send errors
				if len(res.Errors) == 1 && res.Data == nil {
					err := res.Errors[1]
					c.log.WithError(fmt.Errorf(err.Message)).Errorf("subscription encountered an error")
					c.sendMessage(protocols.OperationMessage{
						ID:   msg.ID,
						Type: protocols.MsgError,
						Payload: map[string]interface{}{
							"message": err.Message,
						},
					})
				} else {
					c.sendMessage(protocols.OperationMessage{
						ID:   msg.ID,
						Type: protocols.MsgData,
						Payload: ExecutionResult{
							Errors:     res.Errors,
							Data:       res.Data,
							Extensions: res.Extensions,
						},
					})
				}
			}
		}
	}()
}
