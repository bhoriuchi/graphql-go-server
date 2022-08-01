package graphqlws

import (
	"context"
	"fmt"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/utils"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

func (c *wsConnection) handleStart(msg *protocol.OperationMessage) {
	var (
		err             error
		operationResult interface{}
	)
	id := msg.ID

	if id == "" {
		c.log.Debugf("received START message")
		msgtxt := "message contains no ID"
		c.log.Errorf(msgtxt)
		c.sendError("", protocol.MsgError, map[string]interface{}{
			"message": msgtxt,
		})
		return
	}

	subLog := c.log.WithField("subscriptionId", id)
	subLog.Debugf("received START message")

	if !c.ConnectionInitReceived() {
		err := fmt.Errorf("attempted start operation on uninitialized connection")
		c.sendError(id, protocol.MsgConnectionError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	// if we already have a subscription with this id, unsubscribe from it first
	if c.mgr.HasSubscription(id) {
		c.mgr.Unsubscribe(id)
	}

	payload := &StartMessagePayload{}
	if err := utils.ReMarshal(msg.Payload, payload); err != nil {
		msgtxt := fmt.Sprintf("failed to parse start payload: %s", err)
		subLog.WithError(err).Errorf("failed to parse start payload")
		c.sendError(id, protocol.MsgError, map[string]interface{}{
			"message": msgtxt,
		})
		return
	}

	if err := payload.Validate(); err != nil {
		subLog.WithError(err).Errorf("start payload validation error")
		c.sendError(id, protocol.MsgError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	execArgs := &graphql.Params{
		Schema:         *c.schema,
		RequestString:  payload.Query,
		VariableValues: payload.Variables,
		OperationName:  payload.OperationName,
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
		c.sendError(id, protocol.MsgError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	operation, err := utils.GetOperationAST(document, execArgs.OperationName)
	if err != nil {
		subLog.WithError(err).Errorf("failed to identify operation")
		err = fmt.Errorf("failed to identify operation: %s", err)
		c.sendError(id, protocol.MsgError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	rctx := context.Background()
	if c.config.ContextValueFunc != nil {
		rctx, _ = c.config.ContextValueFunc(c, *msg, *execArgs)
	}

	// add the connection to the metadata context
	ctx, cancelFunc := context.WithCancel(rctx)
	execArgs.Context = ctx

	// set the root value
	if execArgs.RootObject == nil {
		if c.config.RootValueFunc != nil {
			execArgs.RootObject = c.config.RootValueFunc(execArgs.Context, c.config.Request, operation)
		}
	}

	if execArgs.RootObject != nil {
		execArgs.RootObject = map[string]interface{}{}
	}

	// handle onOperation hook if set
	if c.config.OnOperation != nil {
		parsedMessage := StartMessage{
			ID:      id,
			Type:    msg.Type,
			Payload: *payload,
		}

		if execArgs, err = c.config.OnOperation(c, parsedMessage, execArgs); err != nil {
			c.log.WithError(err).Errorf("onOperation hook failed")
			c.sendError(id, protocol.MsgError, map[string]interface{}{
				"message": err.Error(),
			})
			cancelFunc()
			return
		}
	}

	if operation.Operation == ast.OperationTypeSubscription {
		operationResult = graphql.Subscribe(*execArgs)
	} else {
		operationResult = graphql.Do(*execArgs)
	}

	switch result := operationResult.(type) {
	case chan *graphql.Result:
		if err := c.mgr.Subscribe(&manager.Subscription{
			IsSub:         true,
			Channel:       result,
			ConnectionID:  c.id,
			OperationID:   id,
			OperationName: subName,
			Context:       ctx,
			CancelFunc:    cancelFunc,
		}); err != nil {
			c.log.WithError(err).Errorf("subscribe operation failed")
			c.sendError(id, protocol.MsgError, map[string]interface{}{
				"message": err.Error(),
			})
			return
		}

		c.log.Tracef("subscription %q SUBSCRIBED", subName)
		subLog.Tracef("subscription count increased to: %d", c.mgr.SubscriptionCount())
		go c.subscribe(ctx, id, subName, *execArgs, result, subLog)

	case *graphql.Result:
		cancelFunc()
		c.sendMessage(protocol.OperationMessage{
			ID:   id,
			Type: protocol.MsgData,
			Payload: protocol.ExecutionResult{
				Errors:     result.Errors,
				Data:       result.Data,
				Extensions: result.Extensions,
			},
		})

		c.sendMessage(protocol.OperationMessage{
			ID:   id,
			Type: protocol.MsgComplete,
		})

		c.mgr.Unsubscribe(id)

	default:
		cancelFunc()
		err := fmt.Errorf("invalid operationResult type %T", operationResult)
		subLog.WithError(err).Errorf("failed subscribe operation")
		c.close(UnexpectedCondition, err.Error())
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
		c.log.Debugf("subscription %q UNSUBSCRIBED", subName)

		if c.config.OnOperationComplete != nil {
			c.config.OnOperationComplete(c, id)
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
				if c.mgr.HasSubscription(id) {
					c.sendMessage(protocol.OperationMessage{
						ID:   id,
						Type: protocol.MsgComplete,
					})
				}

				return
			}

			// if the response is all errors, close the result and send errors
			if len(res.Errors) == 1 && res.Data == nil {
				err := res.Errors[1]
				c.log.WithError(fmt.Errorf(err.Message)).Errorf("subscription encountered an error")
				c.sendMessage(protocol.OperationMessage{
					ID:   id,
					Type: protocol.MsgError,
					Payload: map[string]interface{}{
						"message": err.Message,
					},
				})
			} else {
				c.sendMessage(protocol.OperationMessage{
					ID:   id,
					Type: protocol.MsgData,
					Payload: protocol.ExecutionResult{
						Errors:     res.Errors,
						Data:       res.Data,
						Extensions: res.Extensions,
					},
				})
			}
		}
	}
}
