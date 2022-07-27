package graphqltransportws

import (
	"context"
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/metadata"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// handleSubscribe manages a subscribe operation
func (c *wsConnection) handleSubscribe(msg *RawMessage) {
	id, err := msg.ID()
	if err != nil {
		c.log.Errorf("%d: %s", BadRequest, err)
		c.setClose(BadRequest, err.Error())
		return
	}

	if !c.Acknowledged() {
		err := fmt.Errorf("Unauthorized")
		c.log.Errorf("%d: %s", Unauthorized, err)
		c.setClose(Unauthorized, err.Error())
		return
	}

	if c.mgr.HasSubscription(id) {
		err := fmt.Errorf("subscriber for %s already exists", id)
		c.log.Errorf("%d: %s", SubscriberAlreadyExists, err)
		c.setClose(SubscriberAlreadyExists, err.Error())
		return
	}

	payload, err := msg.SubscribePayload()
	if err != nil {
		c.log.Errorf("%d: %s", BadRequest, err)
		c.setClose(BadRequest, err.Error())
		return
	}

	params := &graphql.Params{}

	if c.config.OnSubscribe != nil {
		maybeExecArgsOrErrors, err := c.config.OnSubscribe(c, SubscribeMessage{
			ID:      id,
			Type:    MsgSubscribe,
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

	resultChannel := graphql.Subscribe(args)
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
				return

			case res, more := <-resultChannel:
				if !more {
					// if the result channel is finished, send a complete message
					out := OperationMessage{
						ID:   id,
						Type: MsgComplete,
					}

					if c.config.OnComplete != nil {
						c.config.OnComplete(c, CompleteMessage{
							ID:   id,
							Type: MsgComplete,
						})
					}

					if c.mgr.HasSubscription(id) {
						c.Send(out)
					}

					return
				}

				if res.HasErrors() && res.Data == nil {
					if err := c.handleGQLErrors(id, res.Errors); err != nil {
						c.log.Errorf("%d: %s", InternalServerError, err)
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
								Type:    MsgNext,
								Payload: execResult,
							},
							args,
							res,
						)
						if err != nil {
							c.log.Errorf("%d: %s", InternalServerError, err.Error())
							c.setClose(InternalServerError, err.Error())
							return
						}
						if maybeResult != nil {
							execResult, ok := maybeResult.(*ExecutionResult)
							if !ok {
								err := fmt.Errorf("onNext hook expected return type of ExecutionResult but got %T", maybeResult)
								c.log.Errorf("%d: %s", InternalServerError, err)
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
