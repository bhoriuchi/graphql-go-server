package server

import (
	"github.com/bhoriuchi/graphql-go-server/ide"
	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/ws/connection"
)

type Option func(opts *serverOptions)

type serverOptions struct {
	Pretty             bool
	RootValueFunc      RootValueFunc
	FormatErrorFunc    FormatErrorFunc
	ContextFunc        ContextFunc
	WSContextFunc      ContextFunc
	ResultCallbackFunc ResultCallbackFunc
	Logger             logger.Logger
	WS                 *WSOptions
	Playground         *ide.PlaygroundOptions
	GraphiQL           *ide.GraphiQLOptions
}

type WSOptions struct {
	AuthenticateFunc connection.AuthenticateFunc
}

func WithPretty() Option {
	return func(opts *serverOptions) {
		opts.Pretty = true
	}
}

func WithLogger(l logger.Logger) Option {
	return func(opts *serverOptions) {
		opts.Logger = l
	}
}

func WithRootValueFunc(f RootValueFunc) Option {
	return func(opts *serverOptions) {
		opts.RootValueFunc = f
	}
}

func WithFormatErrorFunc(f FormatErrorFunc) Option {
	return func(opts *serverOptions) {
		opts.FormatErrorFunc = f
	}
}

func WithContextFunc(f ContextFunc) Option {
	return func(opts *serverOptions) {
		opts.ContextFunc = f
	}
}

func WithWebsocketContextFunc(f ContextFunc) Option {
	return func(opts *serverOptions) {
		opts.WSContextFunc = f
	}
}

func WithResultCallbackFunc(f ResultCallbackFunc) Option {
	return func(opts *serverOptions) {
		opts.ResultCallbackFunc = f
	}
}

func WithPlaygroundOptions(o *ide.PlaygroundOptions) Option {
	return func(opts *serverOptions) {
		opts.Playground = o
	}
}

func WithGraphiQLOptions(o *ide.GraphiQLOptions) Option {
	return func(opts *serverOptions) {
		opts.GraphiQL = o
	}
}
