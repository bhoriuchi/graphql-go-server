package graphqltransportws

import (
	"fmt"
	"sync"
	"time"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/ws/manager"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

var ConnKey interface{} = "conn"

// ConnectionConfig defines the configuration parameters of a
// GraphQL WebSocket connection.
type Config struct {
	Schema       graphql.Schema
	Logger       logger.Logger
	OnConnect    func(conn *wsConnection, ctx Context) (interface{}, error)
	OnPing       func(conn *wsConnection, payload interface{})
	OnPong       func(conn *wsConnection, payload interface{})
	OnDisconnect func()
	OnClose      func()
	OnSubscribe  func()
	OnOperation  func()
	OnNext       func()
	OnError      func()
	OnComplete   func(conn *wsConnection, ctx Context, msg *OperationMessage)
}

type wsConnection struct {
	schema   graphql.Schema
	mx       sync.Mutex
	closeMx  sync.Mutex
	id       string
	ws       *websocket.Conn
	config   Config
	logger   logger.Logger
	outgoing chan OperationMessage
	closed   bool
	ctx      Context
	mgr      *manager.Manager
}

// NewConnection establishes a GraphQL WebSocket connection. It implements
// the GraphQL WebSocket protocol by managing its internal state and handling
// the client-server communication.
func NewConnection(ws *websocket.Conn, config Config) *wsConnection {
	conn := &wsConnection{
		schema: config.Schema,
		id:     uuid.NewString(),
		ws:     ws,
		config: config,
		ctx: Context{
			"connectionInitReceived": false,
			"acknowledged":           false,
		},
		logger:   config.Logger,
		closed:   false,
		outgoing: make(chan OperationMessage),
	}

	go conn.writeLoop()
	go conn.readLoop()
	return conn
}

// ID returns the connection id
func (c *wsConnection) ID() string {
	return c.id
}

// WS returns the websocket
func (c *wsConnection) WS() *websocket.Conn {
	return c.ws
}

// Send sends a message
func (c *wsConnection) Send(msg OperationMessage) {
	c.outgoing <- msg
}

func (c *wsConnection) Close(code CloseCode, msg string) {
	// Close the write loop by closing the outgoing messages channels
	c.closeMx.Lock()
	c.closed = true

	c.ws.WriteMessage(int(code), []byte(msg))
	close(c.outgoing)
	c.closeMx.Unlock()

	// Notify event handlers
	if c.config.OnClose != nil {
		c.config.OnClose()
	}

	c.logger.Infof("closed connection")
}

func (conn *wsConnection) writeLoop() {
	// Close the WebSocket connection when leaving the write loop;
	// this ensures the read loop is also terminated and the connection
	// closed cleanly
	defer conn.ws.Close()

	for {
		msg, ok := <-conn.outgoing
		// Close the write loop when the outgoing messages channel is closed;
		// this will close the connection
		if !ok {
			return
		}

		// conn.logger.Debugf("send message: %s", msg.String())
		conn.ws.SetWriteDeadline(time.Now().Add(WriteTimeout))

		// Send the message to the client; if this times out, the WebSocket
		// connection will be corrupt, hence we need to close the write loop
		// and the connection immediately
		if err := conn.ws.WriteJSON(msg); err != nil {
			conn.logger.Warnf("sending message failed: %s", err)
			return
		}
	}
}

func (c *wsConnection) readLoop() {
	// Close the WebSocket connection when leaving the read loop
	defer c.ws.Close()
	c.ws.SetReadLimit(ReadLimit)

	for {
		msg := new(RawMessage)
		err := c.ws.ReadJSON(&msg)
		if err != nil {
			c.logger.Errorf("force closing connection: %s", err)
			c.Close(BadRequest, err.Error())
			return
		}

		msgType, err := msg.Type()
		if err != nil {
			c.logger.Errorf(err.Error())
			c.Close(BadRequest, err.Error())
			return
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
			txt := fmt.Sprintf("Unexpected message of type %s received", msgType)
			c.logger.Errorf(txt)
			c.Close(BadRequest, txt)
		}
	}
}
