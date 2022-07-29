package graphqltransportws

import (
	"context"
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/metadata"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/bhoriuchi/graphql-go-server/ws/protocols"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// handleSubscribe manages a subscribe operation
func (c *wsConnection) handleSubscribe(msg *RawMessage) {
	c.log.Tracef("received SUBSCRIBE message")

	id, err := msg.ID()
	if err != nil {
		c.setClose(BadRequest, err.Error())
		c.log.WithField("error", err).WithField("code", c.closeCode).Errorf("subscribe operation failed")
		return
	}

	subLog := c.log.WithField("subscriptionId", id)

	if !c.Acknowledged() {
		c.setClose(Unauthorized, err.Error())
		subLog.WithField("code", Unauthorized).Warnf("attempted subscribe operation before connection is acknowledged")
		return
	}

	if c.mgr.HasSubscription(id) {
		err := fmt.Errorf("subscriber for %s already exists", id)
		c.setClose(SubscriberAlreadyExists, err.Error())
		subLog.WithField("code", c.closeCode).Errorf("failed subscribe operation")
		return
	}

	payload, err := msg.SubscribePayload()
	if err != nil {
		c.setClose(BadRequest, err.Error())
		c.log.WithField("error", err).WithField("code", c.closeCode).Errorf("invalid subscribe message payload")
		return
	}

	params := &graphql.Params{}

	if c.config.OnSubscribe != nil {
		maybeExecArgsOrErrors, err := c.config.OnSubscribe(c, SubscribeMessage{
			ID:      id,
			Type:    protocols.MsgSubscribe,
			Payload: *payload,
		})
		if err != nil {
			c.log.Errorf("%d: %s", InternalServerError, err)
			c.setClose(InternalServerError, err.Error())
			return
		}

		if maybeExecArgsOrErrors != nil {
			switch val := maybeExecArgsOrErrors.(type) {
			case gqlerrors.FormattedErrors:
				if err := c.handleGQLErrors(id, val); err != nil {
					c.log.Errorf("%d: %s", InternalServerError, err)
					c.setClose(InternalServerError, err.Error())
					return
				}
			case graphql.Params:
				params = &val
			default:
				err := fmt.Errorf("invalid return value from onSubscribe hook, expected an array of GraphQLError objects or ExecutionArgs")
				c.log.Errorf("%d: %s", InternalServerError, err)
				c.setClose(InternalServerError, err.Error())
				return
			}
		}
	} else {
		if c.schema == nil {
			err := fmt.Errorf("the GraphQL schema is not provided")
			c.log.Errorf("%d: %s", InternalServerError, err)
			c.setClose(InternalServerError, err.Error())
			return
		}

		params.Schema = *c.schema
		if payload.OperationName != nil {
			params.OperationName = *payload.OperationName
		}
		params.RequestString = payload.Query
		params.VariableValues = payload.Variables
	}

	// add root object
	if params.RootObject == nil {
		if c.config.RootValueFunc != nil {
			params.RootObject = c.config.RootValueFunc(c.ctx, c.config.Request)
		} else {
			params.RootObject = map[string]interface{}{}
		}
	}

	if params.Context == nil {
		params.Context = context.Background()
	}

	// add the connection to the metadata context
	metaCtx := metadata.NewWithContext(params.Context)
	metadata.Set(metaCtx, "connection", c)
	ctx, cancelFunc := context.WithCancel(metaCtx)

	args := graphql.Params{
		Schema:         params.Schema,
		RequestString:  params.RequestString,
		VariableValues: params.VariableValues,
		OperationName:  params.OperationName,
		Context:        ctx,
		RootObject:     params.RootObject,
	}

	subName := args.OperationName
	if subName == "" {
		subName = "Unnamed Subscription"
	}

	resultChannel := graphql.Subscribe(args)
	if err := c.mgr.Subscribe(&manager.Subscription{
		Channel:       resultChannel,
		ConnectionID:  c.ID(),
		OperationID:   id,
		OperationName: subName,
		Context:       ctx,
		CancelFunc:    cancelFunc,
	}); err != nil {
		c.log.Errorf("%d: %s", InternalServerError, err)
		c.setClose(InternalServerError, err.Error())
		return
	}

	c.log.Tracef("subscription %q SUBSCRIBED", subName)
	go func() {
		// ensure subscription is always unsubscribed when finished
		defer func() {
			c.mgr.Unsubscribe(id)
			c.log.Debugf("subscription %q UNSUBSCRIBED", subName)
		}()

		for {
			select {
			case <-ctx.Done():
				c.log.Tracef("exiting subscription %q", subName)
				return

			case res, more := <-resultChannel:
				if !more {
					c.log.Tracef("subscription %q has no more messages, unsubscribing", subName)
					// if the result channel is finished, send a complete message
					out := protocols.OperationMessage{
						ID:   id,
						Type: protocols.MsgComplete,
					}

					if c.config.OnComplete != nil {
						c.config.OnComplete(c, CompleteMessage{
							ID:   id,
							Type: protocols.MsgComplete,
						})
					}

					// send the complete message if the subscription is active
					if c.mgr.HasSubscription(id) {
						c.Send(out)
					}

					return
				}

				// if the response is all errors, close the result and send errors
				if len(res.Errors) == 1 && res.Data == nil {
					if err := c.handleGQLErrors(id, res.Errors); err != nil {
						c.log.WithError(err).Errorf("unsubscribing")
						c.setClose(InternalServerError, err.Error())
						return
					}
				} else {
					execResult := ExecutionResult{
						Errors:     res.Errors,
						Data:       res.Data,
						Extensions: res.Extensions,
					}
					nextMessage := NewNextMessage(id, &execResult)

					if c.config.OnNext != nil {
						maybeResult, err := c.config.OnNext(
							c,
							NextMessage{
								ID:      id,
								Type:    protocols.MsgNext,
								Payload: execResult,
							},
							args,
							res,
						)
						if err != nil {
							c.log.WithError(err).Errorf("unsubscribing")
							c.setClose(InternalServerError, err.Error())
							return
						}
						if maybeResult != nil {
							execResult, ok := maybeResult.(*ExecutionResult)
							if !ok {
								err := fmt.Errorf("onNext hook expected return type of ExecutionResult but got %T", maybeResult)
								c.log.WithError(err).Errorf("unsubscribing")
								c.setClose(InternalServerError, err.Error())
								return
							}

							nextMessage = NewNextMessage(id, execResult)
						}
					}

					c.Send(nextMessage)
				}
			}
		}
	}()
}
