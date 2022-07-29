package options

import (
	"context"
	"net/http"

	"github.com/bhoriuchi/graphql-go-server/ide"
	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

const (
	RequestTypeHTTP RequestType = "http"
	RequestTypeWS   RequestType = "ws"
)

type RequestType string

type RootValueFunc func(ctx context.Context, r *http.Request) map[string]interface{}

type FormatErrorFunc func(err error) gqlerrors.FormattedError

type ContextFunc func(t RequestType, r *http.Request) context.Context

type ResultCallbackFunc func(ctx context.Context, params *graphql.Params, result *graphql.Result, responseBody []byte)

type Option func(opts *Options)

type Options struct {
	Pretty             bool
	RootValueFunc      RootValueFunc
	FormatErrorFunc    FormatErrorFunc
	ContextFunc        ContextFunc
	WSContextFunc      ContextFunc
	ResultCallbackFunc ResultCallbackFunc
	LogFunc            logger.LogFunc
	WS                 *WSOptions
	Playground         *ide.PlaygroundOptions
	GraphiQL           *ide.GraphiQLOptions
}

type WSOptions struct {
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

func WithWebsocketContextFunc(f ContextFunc) Option {
	return func(opts *Options) {
		opts.WSContextFunc = f
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
