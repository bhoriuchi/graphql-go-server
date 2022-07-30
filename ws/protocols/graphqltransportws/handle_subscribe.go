package graphqltransportws

import (
	"context"
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/utils"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/bhoriuchi/graphql-go-server/ws/protocols"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
)

// handleSubscribe manages a subscribe operation
func (c *wsConnection) handleSubscribe(msg *RawMessage) {
	var (
		execArgs        *graphql.Params
		maybeExecArgs   *graphql.Params
		operationResult interface{}
		formattedErrs   gqlerrors.FormattedErrors
	)

	// validate the message and create a structured one
	id, err := msg.ID()
	if err != nil {
		c.log.Tracef("received SUBSCRIBE message")
		c.log.WithField("error", err).Errorf("subscribe operation failed")
		c.close(BadRequest, err.Error())
		return
	}

	subLog := c.log.WithField("subscriptionId", id)
	subLog.Tracef("received SUBSCRIBE message")
	payload, err := msg.SubscribePayload()
	if err != nil {
		subLog.WithField("error", err).Errorf("invalid subscribe message payload")
		c.close(BadRequest, err.Error())
		return
	}

	subMsg := SubscribeMessage{
		ID:      id,
		Type:    protocols.MsgSubscribe,
		Payload: *payload,
	}

	// validate that the subscription has been acknowledged
	if !c.Acknowledged() {
		subLog.Errorf("attempted subscribe operation on unacknowledged connection")
		c.close(Unauthorized, "not authorized")
		return
	}

	// attempt to subscribe a placeholder
	// if the subscription exists, close the connection
	if err := c.mgr.Subscribe(&manager.Subscription{OperationID: id}); err != nil {
		subLog.WithError(err).Errorf("failed subscribe operation")
		err := fmt.Errorf("subscriber for %s already exists", id)
		c.close(SubscriberAlreadyExists, err.Error())
		return
	}
	subLog.Tracef("subscription count increased to: %d", c.mgr.SubscriptionCount())

	if c.config.OnSubscribe != nil {
		maybeExecArgs, formattedErrs = c.config.OnSubscribe(c, subMsg)
	}

	// evaluate the exec args
	if formattedErrs != nil {
		if len(formattedErrs) == 0 {
			err := fmt.Errorf("invalid return value from onSubscribe hook, expected an array of GraphQLError objects")
			subLog.WithError(err).Errorf("onSubscribe hook failed")
			c.sendError(id, utils.GQLErrors(err))
			c.mgr.Unsubscribe(id)
			return
		}

		subLog.WithError(formattedErrs[0].OriginalError()).Errorf("onSubscribe hook failed")
		c.sendError(id, formattedErrs)
		c.mgr.Unsubscribe(id)
		return
	} else if maybeExecArgs != nil {
		execArgs = maybeExecArgs
	} else {
		if c.schema == nil {
			err := fmt.Errorf("the GraphQL schema is not provided")
			subLog.WithError(err).Errorf("no schema provided")
			c.sendError(id, utils.GQLErrors(err))
			c.mgr.Unsubscribe(id)
			return
		}

		execArgs = &graphql.Params{
			Schema:         *c.schema,
			OperationName:  *payload.OperationName,
			RequestString:  payload.Query,
			VariableValues: payload.Variables,
		}
	}

	subName := execArgs.OperationName
	if subName == "" {
		subName = "Unnamed Subscription"
	}

	// get the operation
	document, err := utils.ParseQuery(execArgs.RequestString)
	if err != nil {
		subLog.WithError(err).Errorf("failed to parse query")
		err = fmt.Errorf("failed to parse query: %s", err)
		c.sendError(id, utils.GQLErrors(err))
		c.mgr.Unsubscribe(id)
		return
	}

	operation, err := utils.GetOperationAST(document, execArgs.OperationName)
	if err != nil {
		subLog.WithError(err).Errorf("failed to identify operation")
		err = fmt.Errorf("failed to identify operation: %s", err)
		c.sendError(id, utils.GQLErrors(err))
		c.mgr.Unsubscribe(id)
		return
	}

	// add root object
	if execArgs.RootObject == nil {
		if c.config.Roots != nil {
			switch operation.Kind {
			case ast.OperationTypeQuery:
				execArgs.RootObject = c.config.Roots.Query
			case ast.OperationTypeMutation:
				execArgs.RootObject = c.config.Roots.Mutation
			case ast.OperationTypeSubscription:
				execArgs.RootObject = c.config.Roots.Subscription
			}
		}
	}

	if execArgs.RootObject != nil {
		execArgs.RootObject = map[string]interface{}{}
	}

	// add context
	if execArgs.Context == nil {
		if c.config.ContextValueFunc != nil {
			execArgs.Context, formattedErrs = c.config.ContextValueFunc(c, protocols.OperationMessage{
				ID:      subMsg.ID,
				Type:    subMsg.Type,
				Payload: subMsg.Payload,
			}, *execArgs)

			if formattedErrs != nil {
				subLog.WithError(err).Errorf("failed to identify operation")
				err = fmt.Errorf("failed to identify operation: %s", err)
				c.sendError(id, utils.GQLErrors(err))
				c.mgr.Unsubscribe(id)
				return
			}

		} else {
			execArgs.Context = context.Background()
		}
	}

	// create a cancelable context
	ctx, cancelFunc := context.WithCancel(execArgs.Context)
	execArgs.Context = ctx

	if operation.Operation == ast.OperationTypeSubscription {
		operationResult = graphql.Subscribe(*execArgs)
	} else {
		operationResult = graphql.Do(*execArgs)
	}

	if c.config.OnOperation != nil {
		maybeResult, err := c.config.OnOperation(c, subMsg, *execArgs, operationResult)
		if err != nil {
			cancelFunc()
			subLog.WithError(err).Errorf("onOperation hook failed")
			err = fmt.Errorf("onOperation hook failed: %s", err)
			c.sendError(id, utils.GQLErrors(err))
			c.mgr.Unsubscribe(id)
			return
		}

		if maybeResult != nil {
			operationResult = maybeResult
		}
	}

	// handle the result
	switch result := operationResult.(type) {

	// operation was a subscription
	case chan *graphql.Result:
		// if the subscription has already been unsubscribed, exit silently
		if !c.mgr.HasSubscription(id) {
			cancelFunc()
			if err := c.sendComplete(id, false); err != nil {
				subLog.WithError(err).Errorf("failed to complete operation")
			}
			return
		}

		// subscribe the actual subscription
		if err := c.mgr.Subscribe(&manager.Subscription{
			IsSub:         true,
			Channel:       result,
			ConnectionID:  c.id,
			OperationID:   id,
			OperationName: subName,
			Context:       ctx,
			CancelFunc:    cancelFunc,
		}); err != nil {
			cancelFunc()
			err := fmt.Errorf("subscriber for %s already exists", id)
			subLog.WithError(err).Errorf("failed subscribe operation")
			c.close(SubscriberAlreadyExists, err.Error())
			return
		}

		// start the goroutine to handle graphql events
		go c.subscribe(ctx, id, subName, *execArgs, result, subLog)
		subLog.Tracef("subscription %q SUBSCRIBED", subName)

	// operation was a query or mutation
	case *graphql.Result:
		cancelFunc()
		notify := false
		if c.mgr.HasSubscription(id) {
			notify = true
			if err := c.sendNext(NextMessage{
				ID:   id,
				Type: protocols.MsgNext,
				Payload: ExecutionResult{
					Errors:     result.Errors,
					Data:       result.Data,
					Extensions: result.Extensions,
				},
			}, *execArgs, result); err != nil {
				subLog.WithError(err).Errorf("failed to send next")
				c.close(InternalServerError, err.Error())
				return
			}
		}
		c.sendComplete(id, notify)
		c.mgr.Unsubscribe(id)
		subLog.Debugf("subscription %q UNSUBSCRIBED", subName)

	// unknown operation type
	default:
		cancelFunc()
		err := fmt.Errorf("invalid operationResult type %T", operationResult)
		subLog.WithError(err).Errorf("failed subscribe operation")
		c.close(InternalServerError, err.Error())
	}
}

// subscribe performs the graphql subscription operation
func (c *wsConnection) subscribe(
	ctx context.Context,
	id string,
	subName string,
	args graphql.Params,
	resultChannel chan *graphql.Result,
	subLog *logger.LogWrapper,
) {
	// ensure subscription is always unsubscribed when finished
	defer func() {
		c.mgr.Unsubscribe(id)
		subLog.Tracef("subscription ended, current count: %d", c.mgr.SubscriptionCount())
		subLog.Debugf("subscription %q UNSUBSCRIBED", subName)
	}()

	for {
		select {
		case <-ctx.Done():
			subLog.Tracef("exiting subscription %q", subName)
			return

		case res, more := <-resultChannel:
			// if channel has no more messages, send a complete
			if !more {
				subLog.Tracef("subscription %q has no more messages, unsubscribing", subName)
				if err := c.sendComplete(id, c.mgr.HasSubscription(id)); err != nil {
					subLog.WithError(err).Errorf("failed to send complete")
				}
				return
			}

			// if the response is a single error, close the result and send errors
			if len(res.Errors) == 1 && res.Data == nil {
				if err := c.sendError(id, res.Errors); err != nil {
					subLog.WithError(err).Errorf("result channel responsed with a single error")
				}
			} else {
				if err := c.sendNext(NextMessage{
					ID:   id,
					Type: protocols.MsgNext,
					Payload: ExecutionResult{
						Errors:     res.Errors,
						Data:       res.Data,
						Extensions: res.Extensions,
					},
				}, args, res); err != nil {
					subLog.WithError(err).Errorf("failed to send next message")
					c.close(InternalServerError, err.Error())
					return
				}
			}
		}
	}
}
