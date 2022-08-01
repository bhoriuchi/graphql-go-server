package server

import (
	"context"
	"net/http"
	"time"

	"github.com/bhoriuchi/graphql-go-server/ide"
	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol/graphqltransportws"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
)

type RequestType string
type RootValueFunc func(ctx context.Context, r *http.Request) map[string]interface{}
type FormatErrorFunc func(err error) gqlerrors.FormattedError
type ContextFunc func(r *http.Request) context.Context
type ResultCallbackFunc func(ctx context.Context, params *graphql.Params, result *graphql.Result, responseBody []byte)
type Option func(opts *Options)

type Options struct {
	// common configs
	Pretty             bool
	LogFunc            logger.LogFunc
	RootValueFunc      RootValueFunc
	ContextFunc        ContextFunc
	FormatErrorFunc    FormatErrorFunc
	ResultCallbackFunc ResultCallbackFunc

	// WebSocket configs
	GraphQLWS          *GraphQLWS
	GraphQLTransportWS *GraphQLTransportWS

	// IDE configs
	Playground *ide.PlaygroundOptions
	GraphiQL   *ide.GraphiQLOptions
}

type GraphQLWS struct {
	ConnectionInitWaitTimeout time.Duration
	KeepAlive                 time.Duration
	RootValueFunc             func(ctx context.Context, r *http.Request, op *ast.OperationDefinition) map[string]interface{}
	ContextValueFunc          func(c protocol.Context, msg protocol.OperationMessage, execArgs graphql.Params) (context.Context, gqlerrors.FormattedErrors)
	OnConnect                 func(c protocol.Context, payload interface{}) (interface{}, error)
	OnDisconnect              func(c protocol.Context)
	OnOperation               func(c protocol.Context, msg graphqlws.StartMessage, params *graphql.Params) (*graphql.Params, error)
	OnOperationComplete       func(c protocol.Context, id string)
}

type GraphQLTransportWS struct {
	ConnectionInitWaitTimeout time.Duration
	RootValueFunc             func(ctx context.Context, r *http.Request, op *ast.OperationDefinition) map[string]interface{}
	ContextValueFunc          func(c protocol.Context, msg protocol.OperationMessage, execArgs graphql.Params) (context.Context, gqlerrors.FormattedErrors)
	OnConnect                 func(c protocol.Context) (interface{}, error)
	OnPing                    func(c protocol.Context, payload map[string]interface{})
	OnPong                    func(c protocol.Context, payload map[string]interface{})
	OnDisconnect              func(c protocol.Context, code graphqltransportws.CloseCode, reason string)
	OnClose                   func(c protocol.Context, code graphqltransportws.CloseCode, reason string)
	OnSubscribe               func(c protocol.Context, msg graphqltransportws.SubscribeMessage) (*graphql.Params, gqlerrors.FormattedErrors)
	OnNext                    func(c protocol.Context, msg graphqltransportws.NextMessage, args graphql.Params, Result *graphql.Result) (*protocol.ExecutionResult, error)
	OnError                   func(c protocol.Context, msg graphqltransportws.ErrorMessage, errs gqlerrors.FormattedErrors) (gqlerrors.FormattedErrors, error)
	OnComplete                func(c protocol.Context, msg graphqltransportws.CompleteMessage) error
	OnOperation               func(c protocol.Context, msg graphqltransportws.SubscribeMessage, args graphql.Params, result interface{}) (interface{}, error)
}

// NewOptions creates a new default options with optional options funcs
func NewOptions(opts ...Option) *Options {
	o := &Options{
		LogFunc: logger.NoopLogFunc,
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

func WithGraphQLWS(o *GraphQLWS) Option {
	return func(opts *Options) {
		opts.GraphQLWS = o
	}
}

func WithGraphQLTransportWS(o *GraphQLTransportWS) Option {
	return func(opts *Options) {
		opts.GraphQLTransportWS = o
	}
}

func WithPretty() Option {
	return func(opts *Options) {
		opts.Pretty = true
	}
}

func WithLogFunc(l logger.LogFunc) Option {
	return func(opts *Options) {
		opts.LogFunc = l
	}
}

func WithRootValueFunc(f RootValueFunc) Option {
	return func(opts *Options) {
		opts.RootValueFunc = f
	}
}

func WithFormatErrorFunc(f FormatErrorFunc) Option {
	return func(opts *Options) {
		opts.FormatErrorFunc = f
	}
}

func WithContextFunc(f ContextFunc) Option {
	return func(opts *Options) {
		opts.ContextFunc = f
	}
}

func WithResultCallbackFunc(f ResultCallbackFunc) Option {
	return func(opts *Options) {
		opts.ResultCallbackFunc = f
	}
}

func WithPlaygroundOptions(o *ide.PlaygroundOptions) Option {
	return func(opts *Options) {
		opts.Playground = o
	}
}

func WithGraphiQLOptions(o *ide.GraphiQLOptions) Option {
	return func(opts *Options) {
		opts.GraphiQL = o
	}
}
