package graphqltransportws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
)

var (
	CloseDeadlineDuration time.Duration = 100 * time.Millisecond
)

// ConnectionConfig defines the configuration parameters of a
// GraphQL WebSocket connection.
type Config struct {
	WS                        *websocket.Conn
	Schema                    *graphql.Schema
	Logger                    *logger.LogWrapper
	Request                   *http.Request
	ConnectionInitWaitTimeout time.Duration
	RootValueFunc             func(ctx context.Context, r *http.Request, op *ast.OperationDefinition) map[string]interface{}
	ContextValueFunc          func(c protocol.Context, msg protocol.OperationMessage, execArgs graphql.Params) (context.Context, gqlerrors.FormattedErrors)
	OnConnect                 func(c protocol.Context) (interface{}, error)
	OnPing                    func(c protocol.Context, payload map[string]interface{})
	OnPong                    func(c protocol.Context, payload map[string]interface{})
	OnDisconnect              func(c protocol.Context, code CloseCode, reason string)
	OnClose                   func(c protocol.Context, code CloseCode, reason string)
	OnSubscribe               func(c protocol.Context, msg SubscribeMessage) (*graphql.Params, gqlerrors.FormattedErrors)
	OnNext                    func(c protocol.Context, msg NextMessage, args graphql.Params, Result *graphql.Result) (*protocol.ExecutionResult, error)
	OnError                   func(c protocol.Context, msg ErrorMessage, errs gqlerrors.FormattedErrors) (gqlerrors.FormattedErrors, error)
	OnComplete                func(c protocol.Context, msg CompleteMessage) error
	OnOperation               func(c protocol.Context, msg SubscribeMessage, args graphql.Params, result interface{}) (interface{}, error)
}

// wsConnection defines a connection context
type wsConnection struct {
	id                     string
	ctx                    context.Context
	ws                     *websocket.Conn
	schema                 *graphql.Schema
	config                 Config
	log                    *logger.LogWrapper
	outgoing               chan protocol.OperationMessage
	closed                 bool
	mgr                    *manager.Manager
	connectionInitReceived bool
	acknowledged           bool
	connectionParams       map[string]interface{}
	initMx                 sync.RWMutex
	ackMx                  sync.RWMutex
	closeMx                sync.RWMutex
}

// NewConnection establishes a GraphQL WebSocket connection. It implements
// the GraphQL WebSocket protocol by managing its internal state and handling
// the client-server communication.
func NewConnection(ctx context.Context, config Config) (*wsConnection, error) {
	id := uuid.NewString()
	l := config.Logger.
		WithField("connectionId", id).
		WithField("subprotocol", Subprotocol)

	c := &wsConnection{
		id:                     id,
		ctx:                    ctx,
		ws:                     config.WS,
		schema:                 config.Schema,
		config:                 config,
		log:                    l,
		closed:                 false,
		outgoing:               make(chan protocol.OperationMessage),
		connectionInitReceived: false,
		acknowledged:           false,
		mgr:                    manager.NewManager(),
	}

	// validate the subprotocol
	if c.ws.Subprotocol() != Subprotocol {
		err := fmt.Errorf("subprotocol not acceptable")
		c.log.WithError(err).Errorf("failed to create connection")
		c.close(SubprotocolNotAcceptable, err.Error())
		return nil, err
	}

	c.log.Debugf("server accepted graphql subprotocol")

	// start the read and write loops
	go c.writeLoop()
	go c.readLoop()

	if config.ConnectionInitWaitTimeout == 0 {
		config.ConnectionInitWaitTimeout = 3 * time.Second
	}

	time.AfterFunc(config.ConnectionInitWaitTimeout, func() {
		if !c.ConnectionInitReceived() {
			c.close(ConnectionInitialisationTimeout, "connection initialisation timeout")
		}
	})

	return c, nil
}

// ConnectionID returns the connection id
func (c *wsConnection) ConnectionID() string {
	return c.id
}

// Context returns the original connection request context
func (c *wsConnection) Context() context.Context {
	return c.ctx
}

// WS returns the websocket
func (c *wsConnection) WS() *websocket.Conn {
	return c.ws
}

func (c *wsConnection) C() chan protocol.OperationMessage {
	return c.outgoing
}

// ConnectionInitReceived
func (c *wsConnection) ConnectionInitReceived() bool {
	c.initMx.RLock()
	defer c.initMx.RUnlock()
	return c.connectionInitReceived
}

// Acknowledged
func (c *wsConnection) Acknowledged() bool {
	c.ackMx.RLock()
	defer c.ackMx.RUnlock()
	return c.acknowledged
}

// ConnectionParams
func (c *wsConnection) ConnectionParams() map[string]interface{} {
	return c.connectionParams
}

func (c *wsConnection) writeLoop() {
	// Close the WebSocket connection when leaving the write loop;
	// this ensures the read loop is also terminated and the connection
	// closed cleanly
	defer c.ws.Close()

	for {
		msg, ok := <-c.outgoing
		// Close the write loop when the outgoing messages channel is closed;
		// this will close the connection
		if !ok {
			return
		}

		if c.isClosed() {
			return
		}

		// conn.logger.Debugf("send message: %s", msg.String())
		c.ws.SetWriteDeadline(time.Now().Add(WriteTimeout))

		// Send the message to the client; if this times out, the WebSocket
		// connection will be corrupt, hence we need to close the write loop
		// and the connection immediately
		if err := c.ws.WriteJSON(msg); err != nil {
			c.log.WithError(err).Warnf("sending message failed")
			return
		}
	}
}

func (c *wsConnection) readLoop() {
	// Close the WebSocket connection when leaving the read loop
	defer c.ws.Close()

	for {

		msg := new(RawMessage)
		err := c.ws.ReadJSON(&msg)

		if err != nil {
			// look for a normal closure and exit
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) || c.isClosed() {
				c.close(NormalClosure, "Client requested normal closure: close error")
				break
			}

			c.log.WithError(err).Errorf("graphql-transport-ws: force closing connection")
			c.close(BadRequest, err.Error())
			break
		}

		msgType, err := msg.Type()
		if err != nil {
			c.log.WithError(err).Errorf("failed to read message type")
			c.close(BadRequest, err.Error())
			break
		}

		switch msgType {

		case protocol.MsgConnectionInit:
			c.handleConnectionInit(msg)

		case protocol.MsgConnectionTerminate:
			c.close(NormalClosure, "client requested connection termination")

		case protocol.MsgPing:
			c.handlePing(msg)

		case protocol.MsgPong:
			c.handlePong(msg)

		case protocol.MsgSubscribe:
			c.handleSubscribe(msg)

		case protocol.MsgComplete:
			c.handleComplete(msg)

		// GraphQL WS protocol messages that are not handled represent
		// a bug in our implementation; make this very obvious by logging
		// an error
		default:
			err := fmt.Errorf("unexpected message of type %q received", msgType)
			c.log.Errorf("%d: %s", BadRequest, err)
			c.close(BadRequest, err.Error())
		}
	}

	c.log.Tracef("exiting read loop")
}

// send error sends an error
func (c *wsConnection) sendError(id string, errs gqlerrors.FormattedErrors) error {
	if c.config.OnError != nil {
		maybeErrors, err := c.config.OnError(c, ErrorMessage{
			ID:      id,
			Type:    protocol.MsgError,
			Payload: errs,
		}, errs)

		if err != nil {
			return err
		}

		if maybeErrors != nil {
			errs = maybeErrors
		}
	}

	c.sendMessage(protocol.OperationMessage{
		ID:      id,
		Type:    protocol.MsgError,
		Payload: errs,
	})

	return nil
}

// Send sends a message
func (c *wsConnection) sendMessage(msg protocol.OperationMessage) {
	if !c.isClosed() {
		c.outgoing <- msg
	}
}

// close closes the socket with a control message
func (c *wsConnection) close(code CloseCode, msg string) {
	// Close the write loop by closing the outgoing messages channels
	c.closeMx.Lock()
	defer c.closeMx.Unlock()

	if c.closed {
		return
	}

	// mark as closed and stop outbound messages
	c.closed = true
	close(c.outgoing)

	// close the websocket connection
	closeMsg := websocket.FormatCloseMessage(int(code), msg)
	deadline := time.Now().Add(CloseDeadlineDuration)

	// close the connection
	closedWS := true
	if err := c.ws.WriteControl(websocket.CloseMessage, closeMsg, deadline); err != nil {
		if err != websocket.ErrCloseSent {
			c.log.WithError(err).Errorf("failed to write close control message to websocket, trying force close")
			if err := c.ws.Close(); err != nil {
				c.log.WithError(err).Errorf("failed to close websocket")
				closedWS = false
			}
		}
	}

	if closedWS {
		c.log.WithField("code", code).Infof("CLOSED connection with %q", msg)
	}

	// clean up subscriptions
	c.mgr.UnsubscribeAll()

	// onDisconnect hook
	if c.Acknowledged() && c.config.OnDisconnect != nil {
		c.config.OnDisconnect(c, code, msg)
	}

	// onClose hook
	if c.config.OnClose != nil {
		c.config.OnClose(c, code, msg)
	}
}

// sendComplete sends a complete message
func (c *wsConnection) sendComplete(id string, notify bool) error {
	msg := CompleteMessage{
		ID:   id,
		Type: protocol.MsgComplete,
	}

	if c.config.OnComplete != nil {
		if err := c.config.OnComplete(c, msg); err != nil {
			return err
		}
	}

	if notify {
		c.sendMessage(protocol.OperationMessage{
			ID:   id,
			Type: protocol.MsgComplete,
		})
	}

	return nil
}

// sendNext sends a next message
func (c *wsConnection) sendNext(msg NextMessage, args graphql.Params, result *graphql.Result) error {
	var (
		err         error
		maybeResult *protocol.ExecutionResult
	)

	if c.config.OnNext != nil {
		maybeResult, err = c.config.OnNext(c, msg, args, result)

		if err != nil {
			return err
		}

		if maybeResult != nil {
			msg = NextMessage{
				ID:      msg.ID,
				Type:    msg.Type,
				Payload: *maybeResult,
			}
		}
	}

	c.sendMessage(protocol.OperationMessage{
		ID:      msg.ID,
		Type:    msg.Type,
		Payload: msg.Payload,
	})

	return nil
}

// isClosed returns true if the connection is closed
func (c *wsConnection) isClosed() bool {
	c.closeMx.RLock()
	defer c.closeMx.RUnlock()
	return c.closed
}
