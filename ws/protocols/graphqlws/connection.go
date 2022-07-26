package graphqlws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/ws/connection"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ConnectionEventHandlers define the event handlers for a connection.
// Event handlers allow other system components to react to events such
// as the connection closing or an operation being started or stopped.
type ConnectionEventHandlers struct {
	// Close is called whenever the connection is closed, regardless of
	// whether this happens because of an error or a deliberate termination
	// by the client.
	Close func(connection.Connection)

	// StartOperation is called whenever the client demands that a GraphQL
	// operation be started (typically a subscription). Event handlers
	// are expected to take the necessary steps to register the operation
	// and send data back to the client with the results eventually.
	StartOperation func(connection.Connection, string, *StartMessagePayload) []error

	// StopOperation is called whenever the client stops a previously
	// started GraphQL operation (typically a subscription). Event handlers
	// are expected to unregister the operation and stop sending result
	// data to the client.
	StopOperation func(connection.Connection, string)
}

// ConnectionConfig defines the configuration parameters of a
// GraphQL WebSocket connection.
type ConnectionConfig struct {
	Logger        logger.Logger
	Authenticate  connection.AuthenticateFunc
	EventHandlers ConnectionEventHandlers
}

/**
 * The default implementation of the Connection interface.
 */

type wsConnection struct {
	id         string
	ws         *websocket.Conn
	config     ConnectionConfig
	logger     logger.Logger
	outgoing   chan OperationMessage
	closeMutex *sync.Mutex
	closed     bool
	context    context.Context
}

// NewConnection establishes a GraphQL WebSocket connection. It implements
// the GraphQL WebSocket protocol by managing its internal state and handling
// the client-server communication.
func NewConnection(ws *websocket.Conn, config ConnectionConfig) connection.Connection {
	conn := new(wsConnection)
	conn.id = uuid.New().String()
	conn.ws = ws
	conn.context = context.Background()
	conn.config = config
	conn.logger = config.Logger
	conn.closed = false
	conn.closeMutex = &sync.Mutex{}
	conn.outgoing = make(chan OperationMessage)

	go conn.writeLoop()
	go conn.readLoop()
	conn.logger.Infof("Created connection")

	return conn
}

func (conn *wsConnection) ID() string {
	return conn.id
}

func (conn *wsConnection) Context() context.Context {
	return conn.context
}

func (conn *wsConnection) WS() *websocket.Conn {
	return conn.ws
}

func (conn *wsConnection) SendData(opID string, data interface{}) {
	msg := operationMessageForType(MsgData)
	msg.ID = opID
	msg.Payload = data
	conn.closeMutex.Lock()
	if !conn.closed {
		conn.outgoing <- msg
	}
	conn.closeMutex.Unlock()
}

func (conn *wsConnection) SendError(err error) {
	msg := operationMessageForType(MsgError)
	msg.Payload = err.Error()
	conn.closeMutex.Lock()
	if !conn.closed {
		conn.outgoing <- msg
	}
	conn.closeMutex.Unlock()
}

func (conn *wsConnection) sendOperationErrors(opID string, errs []error) {
	if conn.closed {
		return
	}

	msg := operationMessageForType(MsgError)
	msg.ID = opID
	msg.Payload = errs
	conn.closeMutex.Lock()
	if !conn.closed {
		conn.outgoing <- msg
	}

	conn.closeMutex.Unlock()
}

func (conn *wsConnection) close() {
	// Close the write loop by closing the outgoing messages channels
	conn.closeMutex.Lock()
	conn.closed = true
	close(conn.outgoing)
	conn.closeMutex.Unlock()

	// Notify event handlers
	if conn.config.EventHandlers.Close != nil {
		conn.config.EventHandlers.Close(conn)
	}

	conn.logger.Infof("closed connection")
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

func (conn *wsConnection) readLoop() {
	// Close the WebSocket connection when leaving the read loop
	defer conn.ws.Close()
	conn.ws.SetReadLimit(ReadLimit)

	for {
		// Read the next message received from the client
		rawPayload := json.RawMessage{}
		msg := OperationMessage{
			Payload: &rawPayload,
		}
		err := conn.ws.ReadJSON(&msg)

		// If this causes an error, close the connection and read loop immediately;
		// see https://github.com/gorilla/websocket/blob/master/conn.go#L924 for
		// more information on why this is necessary
		if err != nil {
			conn.logger.Warnf("force closing connection: %s", err)
			conn.close()
			return
		}

		// conn.logger.Debugf("received message (%s): %s", msg.ID, msg.Type)

		switch msg.Type {
		case MsgConnectionAuth:
			data := map[string]interface{}{}
			if err := json.Unmarshal(rawPayload, &data); err != nil {
				conn.logger.Errorf("Invalid %s data: %v", msg.Type, err)
				conn.SendError(errors.New("invalid GQL_CONNECTION_AUTH payload"))
			} else {
				if conn.config.Authenticate != nil {
					ctx, err := conn.config.Authenticate(data, conn)
					if err != nil {
						msg := operationMessageForType(MsgConnectionError)
						msg.Payload = fmt.Sprintf("Failed to authenticate user: %v", err)
						conn.outgoing <- msg
					} else {
						conn.context = ctx
					}
				}
			}

		// When the GraphQL WS connection is initiated, send an ACK back
		case MsgConnectionInit:
			data := map[string]interface{}{}
			if err := json.Unmarshal(rawPayload, &data); err != nil {
				conn.logger.Errorf("Invalid %s data: %v", msg.Type, err)
				conn.SendError(errors.New("invalid GQL_CONNECTION_INIT payload"))
			} else {
				if conn.config.Authenticate != nil {
					ctx, err := conn.config.Authenticate(data, conn)
					if err != nil {
						msg := operationMessageForType(MsgConnectionError)
						msg.Payload = fmt.Sprintf("Failed to authenticate user: %v", err)
						conn.outgoing <- msg
					} else {
						conn.context = ctx
						conn.outgoing <- operationMessageForType(MsgConnectionAck)
					}
				} else {
					conn.outgoing <- operationMessageForType(MsgConnectionAck)
				}
			}

		// Let event handlers deal with starting operations
		case MsgStart:
			if conn.config.EventHandlers.StartOperation != nil {
				data := StartMessagePayload{}
				if err := json.Unmarshal(rawPayload, &data); err != nil {
					conn.SendError(errors.New("invalid GQL_START payload"))
				} else {
					errs := conn.config.EventHandlers.StartOperation(conn, msg.ID, &data)
					if errs != nil {
						conn.sendOperationErrors(msg.ID, errs)
					}
				}
			}

		// Let event handlers deal with stopping operations
		case MsgStop:
			if conn.config.EventHandlers.StopOperation != nil {
				conn.config.EventHandlers.StopOperation(conn, msg.ID)
			}

		// When the GraphQL WS connection is terminated by the client,
		// close the connection and close the read loop
		case MsgConnectionTerminate:
			// conn.logger.Debugf("connection terminated by client")
			conn.close()
			return

		// GraphQL WS protocol messages that are not handled represent
		// a bug in our implementation; make this very obvious by logging
		// an error
		default:
			conn.logger.Errorf("unhandled message: %s", msg.String())
		}
	}
}
