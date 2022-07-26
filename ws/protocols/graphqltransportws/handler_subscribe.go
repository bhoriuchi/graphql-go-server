package graphqltransportws

import (
	"context"
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/graphql-go/graphql"
)

// handleSubscribe manages a subscribe operation
func (c *wsConnection) handleSubscribe(msg *RawMessage) {
	if !c.ctx.Acknowledged() {
		c.logger.Errorf("Unauthorized subscribe operation")
		c.Close(Unauthorized, "Unauthorized")
		return
	}

	id, err := msg.ID()
	if err != nil {
		c.logger.Errorf(err.Error())
		c.Close(BadRequest, err.Error())
		return
	}

	if c.mgr.HasSubscription(id) {
		msgtxt := fmt.Sprintf("Subscriber for %s already exists", id)
		c.logger.Errorf(msgtxt)
		c.Close(SubscriberAlreadyExists, msgtxt)
		return
	}

	payload, err := msg.SubscribePayload()
	if err != nil {
		c.logger.Errorf(err.Error())
		c.Close(BadRequest, err.Error())
		return
	}

	rootObject := map[string]interface{}{}
	/*
		if s.options.RootValueFunc != nil {
			rootObject = s.options.RootValueFunc(ctx, r)
		}
	*/

	opname := ""
	if payload.OperationName != nil {
		opname = *payload.OperationName
	}

	ctx, cancelFunc := context.WithCancel(context.WithValue(context.Background(), ConnKey, c))
	resultChannel := graphql.Subscribe(graphql.Params{
		Schema:         c.schema,
		RequestString:  payload.Query,
		VariableValues: payload.Variables,
		OperationName:  opname,
		Context:        ctx,
		RootObject:     rootObject,
	})

	c.mgr.Subscribe(&manager.Subscription{
		Channel:      resultChannel,
		ConnectionID: c.ID(),
		OperationID:  id,
		Context:      ctx,
		CancelFunc:   cancelFunc,
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.mgr.UnsubscribeAll()
				return

			case res, more := <-resultChannel:
				if !more {
					out := &OperationMessage{
						ID:   id,
						Type: MsgComplete,
					}

					if c.config.OnComplete != nil {
						c.config.OnComplete(c, c.ctx, out)
					}

					if c.mgr.HasSubscription(id) {
						c.outgoing <- *out
					}

					return
				}

				if res.HasErrors() && res.Data == nil {
					c.outgoing <- OperationMessage{
						ID:      id,
						Type:    MsgError,
						Payload: res.Errors,
					}
				} else {
					c.outgoing <- OperationMessage{
						ID:      id,
						Type:    MsgNext,
						Payload: res,
					}
				}
			}
		}
	}()
}
