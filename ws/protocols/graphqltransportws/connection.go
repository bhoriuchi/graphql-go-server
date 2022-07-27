package graphqltransportws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// ConnectionConfig defines the configuration parameters of a
// GraphQL WebSocket connection.
type Config struct {
	WS                        *websocket.Conn
	Schema                    *graphql.Schema
	Logger                    logger.Logger
	Request                   *http.Request
	RootValueFunc             func(ctx context.Context, r *http.Request) map[string]interface{}
	ConnectionInitWaitTimeout time.Duration
	OnConnect                 func(c *wsConnection) (interface{}, error)
	OnPing                    func(c *wsConnection, payload map[string]interface{})
	OnPong                    func(c *wsConnection, payload map[string]interface{})
	OnDisconnect              func(c *wsConnection, code CloseCode, reason string)
	OnClose                   func(c *wsConnection, code CloseCode, reason string)
	OnSubscribe               func(c *wsConnection, msg SubscribeMessage) (interface{}, error)
	OnNext                    func(c *wsConnection, msg NextMessage, Args graphql.Params, Result *graphql.Result) (interface{}, error)
	OnError                   func(c *wsConnection, msg ErrorMessage, errs gqlerrors.FormattedErrors) (gqlerrors.FormattedErrors, error)
	OnComplete                func(c *wsConnection, msg CompleteMessage) error
	// OnOperation               func()
}

type wsConnection struct {
	id                     string
	ctx                    context.Context
	ws                     *websocket.Conn
	schema                 *graphql.Schema
	initMx                 sync.RWMutex
	ackMx                  sync.RWMutex
	closeMx                sync.RWMutex
	config                 Config
	log                    logger.Logger
	outgoing               chan OperationMessage
	closed                 bool
	mgr                    *manager.Manager
	connectionInitReceived bool
	acknowledged           bool
	connectionParams       map[string]interface{}
	closeCode              CloseCode
	closeReason            string
}

// NewConnection establishes a GraphQL WebSocket connection. It implements
// the GraphQL WebSocket protocol by managing its internal state and handling
// the client-server communication.
func NewConnection(ctx context.Context, config Config) (*wsConnection, error) {
	c := &wsConnection{
		ctx:                    ctx,
		schema:                 config.Schema,
		id:                     uuid.NewString(),
		ws:                     config.WS,
		config:                 config,
		log:                    config.Logger,
		closed:                 false,
		outgoing:               make(chan OperationMessage),
		connectionInitReceived: false,
		acknowledged:           false,
		closeCode:              Noop,
		closeReason:            "",
	}

	// validate the subprotocol
	if c.ws.Subprotocol() != Subprotocol {
		c.setClose(SubprotocolNotAcceptable, "Subprotocol not acceptable")
		c.Close(c.closeCode, c.closeReason)

		if c.config.OnClose != nil {
			c.config.OnClose(c, c.closeCode, c.closeReason)
		}

		return nil, fmt.Errorf("subprotocol not acceptable")
	}

	// start the read and write loops
	go c.writeLoop()
	go c.readLoop()

	if config.ConnectionInitWaitTimeout == 0 {
		config.ConnectionInitWaitTimeout = 3 * time.Second
	}

	time.AfterFunc(config.ConnectionInitWaitTimeout, func() {
		if !c.ConnectionInitReceived() {
			c.setClose(ConnectionInitialisationTimeout, "Connection initialisation timeout")
			c.Close(c.closeCode, c.closeReason)
		}
	})

	return c, nil
}

// ID returns the connection id
func (c *wsConnection) ID() string {
	return c.id
}

// WS returns the websocket
func (c *wsConnection) WS() *websocket.Conn {
	return c.ws
}

// Context returns the original connection request context
func (c *wsConnection) Context() context.Context {
	return c.ctx
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
func (c *wsConnection) ConnectionParams() interface{} {
	return c.connectionParams
}

// Send sends a message
func (c *wsConnection) Send(msg OperationMessage) {
	if !c.isClosed() {
		c.outgoing <- msg
	}
}

func (c *wsConnection) Close(code CloseCode, msg string) {
	// Close the write loop by closing the outgoing messages channels
	c.closeMx.Lock()
	c.closed = true

	c.ws.WriteMessage(int(code), []byte(msg))
	close(c.outgoing)
	c.closeMx.Unlock()

	c.log.Infof("closed connection %s", c.id)
}

func (c *wsConnection) writeLoop() {
	// Close the WebSocket connection when leaving the write loop;
	// this ensures the read loop is also terminated and the connection
	// closed cleanly
	defer c.ws.Close()

	for {
		if c.isClosed() {
			break
		}

		msg, ok := <-c.outgoing
		// Close the write loop when the outgoing messages channel is closed;
		// this will close the connection
		if !ok {
			return
		}

		// conn.logger.Debugf("send message: %s", msg.String())
		c.ws.SetWriteDeadline(time.Now().Add(WriteTimeout))

		// Send the message to the client; if this times out, the WebSocket
		// connection will be corrupt, hence we need to close the write loop
		// and the connection immediately
		if err := c.ws.WriteJSON(msg); err != nil {
			c.log.Warnf("sending message failed: %s", err)
			return
		}
	}
}

func (c *wsConnection) readLoop() {
	// Close the WebSocket connection when leaving the read loop
	defer c.ws.Close()
	c.ws.SetReadLimit(ReadLimit)

	for {
		if c.isClosed() {
			break
		}

		msg := new(RawMessage)
		err := c.ws.ReadJSON(&msg)

		if err != nil {
			// look for a normal closure and exit
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				c.setClose(NormalClosure, "Client requested normal closure")
				break
			}

			c.log.Errorf("force closing connection: %s", err)
			c.setClose(BadRequest, err.Error())
			break
		}

		msgType, err := msg.Type()
		if err != nil {
			c.log.Errorf(err.Error())
			c.setClose(BadRequest, err.Error())
			break
		}

		switch msgType {
		// When the GraphQL WS connection is initiated, send an ACK back
		case MsgConnectionInit:
			c.handleConnectionInit(msg)

		case MsgPing:
			c.handlePing(msg)

		case MsgPong:
			c.handlePong(msg)

		case MsgSubscribe:
			c.handleSubscribe(msg)

		case MsgComplete:
			c.handleComplete(msg)

		// GraphQL WS protocol messages that are not handled represent
		// a bug in our implementation; make this very obvious by logging
		// an error
		default:
			err := fmt.Errorf("unexpected message of type %s received", msgType)
			c.log.Errorf("%d: %s", BadRequest, err)
			c.setClose(BadRequest, err.Error())
		}
	}

	c.mgr.UnsubscribeAll()

	if c.Acknowledged() && c.config.OnDisconnect != nil {
		c.config.OnDisconnect(c, c.closeCode, c.closeReason)
	}

	c.Close(c.closeCode, c.closeReason)
}

// handleGQLErrors handles graphql errors
func (c *wsConnection) handleGQLErrors(id string, errs gqlerrors.FormattedErrors) error {
	if c.config.OnError != nil {
		maybeErrors, err := c.config.OnError(c, ErrorMessage{
			ID:      id,
			Type:    MsgError,
			Payload: errs,
		}, errs)

		if err != nil {
			return err
		}

		if maybeErrors != nil {
			errs = maybeErrors
		}
	}

	c.Send(NewErrorMessage(id, errs))
	return nil
}

// setClose sets the close code and reason using the mutex
func (c *wsConnection) setClose(code CloseCode, reason string) {
	c.closeMx.Lock()
	defer c.closeMx.Unlock()

	// only set the close code if it is unset upon obtaining the lock
	if c.closeCode == Noop {
		c.closeCode = code
		c.closeReason = reason
	}
}

// isClosed returns true if the connection is closed
func (c *wsConnection) isClosed() bool {
	c.closeMx.RLock()
	defer c.closeMx.RUnlock()
	return c.closeCode != Noop || c.closed
}
