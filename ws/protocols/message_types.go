package protocols

// MessageType is a message type
type MessageType string

const (
	// Common
	MsgConnectionInit MessageType = "connection_init"
	MsgConnectionAck  MessageType = "connection_ack"
	MsgError          MessageType = "error"
	MsgComplete       MessageType = "complete"

	// graphql-ws protocol specific
	MsgPing      MessageType = "ping"
	MsgPong      MessageType = "pong"
	MsgSubscribe MessageType = "subscribe"
	MsgNext      MessageType = "next"

	// graphql-transport-ws specific - deprecated protocol
	MsgKeepAlive           MessageType = "ka"
	MsgConnectionError     MessageType = "connection_error"
	MsgConnectionTerminate MessageType = "connection_terminate"
	MsgStart               MessageType = "start"
	MsgData                MessageType = "data"
	MsgStop                MessageType = "stop"
)
