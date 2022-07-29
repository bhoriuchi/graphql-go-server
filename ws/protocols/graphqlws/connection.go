package graphqlws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/bhoriuchi/graphql-go-server/ws/protocols"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

// ConnectionConfig defines the configuration parameters of a
// GraphQL WebSocket connection.
type Config struct {
	WS                  *websocket.Conn
	Schema              *graphql.Schema
	Logger              *logger.LogWrapper
	Request             *http.Request
	RootValueFunc       func(ctx context.Context, r *http.Request) map[string]interface{}
	KeepAlive           time.Duration
	OnConnect           func(c *wsConnection, payload interface{}) interface{}
	OnDisconnect        func(c *wsConnection)
	OnOperation         func(c *wsConnection, msg StartMessage, params graphql.Params) (graphql.Params, error)
	OnOperationComplete func(c *wsConnection, id string)
}

// wsConnection defines a connection context
type wsConnection struct {
	id                     string
	ctx                    context.Context
	ws                     *websocket.Conn
	schema                 *graphql.Schema
	config                 Config
	log                    *logger.LogWrapper
	outgoing               chan protocols.OperationMessage
	ka                     chan struct{}
	closeMx                sync.RWMutex
	initMx                 sync.RWMutex
	closed                 bool
	mgr                    *manager.Manager
	connectionParams       map[string]interface{}
	connectionInitReceived bool
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
		id:       id,
		ctx:      ctx,
		schema:   config.Schema,
		ws:       config.WS,
		config:   config,
		log:      l,
		closed:   false,
		outgoing: make(chan protocols.OperationMessage),
		ka:       make(chan struct{}),
		mgr:      manager.NewManager(),
	}

	// validate the subprotocol
	if c.ws.Subprotocol() != Subprotocol {
		err := fmt.Errorf("subprotocol %q not acceptable", c.ws.Subprotocol())
		c.log.WithError(err).Errorf("failed to create connection")
		c.ws.WriteMessage(int(ProtocolError), []byte(err.Error()))
		time.Sleep(10 * time.Millisecond)
		c.ws.Close()
		return nil, err
	}

	go c.writeLoop()
	go c.readLoop()

	return c, nil
}

func (c *wsConnection) ID() string {
	return c.id
}

func (c *wsConnection) Context() context.Context {
	return c.ctx
}

func (c *wsConnection) WS() *websocket.Conn {
	return c.ws
}

func (c *wsConnection) C() chan protocols.OperationMessage {
	return c.outgoing
}

// ConnectionInitReceived
func (c *wsConnection) ConnectionInitReceived() bool {
	c.initMx.RLock()
	defer c.initMx.RUnlock()
	return c.connectionInitReceived
}

// ConnectionParams
func (c *wsConnection) ConnectionParams() interface{} {
	return c.connectionParams
}

// Send sends a message
func (c *wsConnection) sendMessage(msg protocols.OperationMessage) {
	if !c.isClosed() {
		c.outgoing <- msg
	}
}

// close closes the connection
func (c *wsConnection) close(code CloseCode, msg string) {
	c.closeMx.Lock()
	c.closed = true
	c.closeMx.Unlock()

	c.mgr.UnsubscribeAll()

	if c.config.OnDisconnect != nil {
		c.config.OnDisconnect(c)
	}

	// close the outgoing channels
	close(c.ka)
	close(c.outgoing)
	c.ws.Close()
	c.log.Infof("closed connection")
}

func (c *wsConnection) writeLoop() {
	// Close the WebSocket connection when leaving the write loop;
	// this ensures the read loop is also terminated and the connection
	// closed cleanly
	defer c.ws.Close()

	for {
		if c.isClosed() {
			return
		}

		msg, ok := <-c.outgoing
		// Close the write loop when the outgoing messages channel is closed;
		// this will close the connection
		if !ok {
			break
		}

		// conn.logger.Debugf("send message: %s", msg.String())
		c.ws.SetWriteDeadline(time.Now().Add(WriteTimeout))

		// Send the message to the client; if this times out, the WebSocket
		// connection will be corrupt, hence we need to close the write loop
		// and the connection immediately
		if err := c.ws.WriteJSON(msg); err != nil {
			c.log.WithError(err).Warnf("failed to write message")
			return
		}
	}
}

func (c *wsConnection) readLoop() {
	// Close the WebSocket connection when leaving the read loop
	defer c.ws.Close()

	for {
		if c.isClosed() {
			break
		}

		// Read the next message received from the client
		msg := &protocols.OperationMessage{}
		err := c.ws.ReadJSON(msg)

		// If this causes an error, close the connection and read loop immediately;
		// see https://github.com/gorilla/websocket/blob/master/conn.go#L924 for
		// more information on why this is necessary
		if err != nil {
			// look for a normal closure and exit
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				c.log.Debugf("gracefully closing connection with normal closure")
				c.close(NormalClosure, "")
				break
			}

			c.log.WithError(err).Errorf("force closing connection")
			c.sendError("", protocols.MsgConnectionError, map[string]interface{}{
				"message": err.Error(),
			})
			time.Sleep(10 * time.Millisecond)
			c.close(UnexpectedCondition, err.Error())
			break
		}

		switch msg.Type {

		case protocols.MsgConnectionInit:
			c.handleConnectionInit(msg)

		case protocols.MsgConnectionTerminate:
			c.handleConnectionTerminate(msg)

		case protocols.MsgStart:
			c.handleStart(msg)

		case protocols.MsgStop:
			c.handleStop(msg)

		default:
			err := fmt.Errorf("unhandled message type %q", msg.Type)
			c.log.WithError(err).Errorf("failed to handle message")
			c.sendError(msg.ID, protocols.MsgError, map[string]interface{}{
				"message": err.Error(),
			})
		}
	}
}

// isClosed returns true if the connection is closed
func (c *wsConnection) isClosed() bool {
	c.closeMx.RLock()
	defer c.closeMx.RUnlock()
	return c.closed
}

// handleGQLErrors handles graphql errors
func (c *wsConnection) sendError(id string, t protocols.MessageType, errs interface{}) error {
	c.sendMessage(protocols.OperationMessage{
		ID:      id,
		Type:    t,
		Payload: errs,
	})
	return nil
}
