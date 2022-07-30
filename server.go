package server

import (
	"net/http"
	"strings"

	"github.com/bhoriuchi/graphql-go-server/ide"
	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/options"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol/graphqltransportws"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol/graphqlws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

// Constants
const (
	ContentTypeJSON           = "application/json"
	ContentTypeGraphQL        = "application/graphql"
	ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

type Server struct {
	schema   graphql.Schema
	log      *logger.LogWrapper
	options  *options.Options
	upgrader websocket.Upgrader
}

// TODO: add hook options, root func and context func
func New(schema graphql.Schema, opts ...options.Option) *Server {
	options := &options.Options{
		LogFunc:    logger.NoopLogFunc,
		Playground: ide.NewDefaultPlaygroundOptions(),
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Server{
		schema:  schema,
		log:     logger.NewLogWrapper(options.LogFunc, nil),
		options: options,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
			Subprotocols: []string{
				graphqltransportws.Subprotocol,
				graphqlws.Subprotocol,
			},
		},
	}
}

// isWSUpgrade identifies a websocket upgrade
func (s *Server) isWSUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// ServeHTTP provides an entrypoint into executing graphQL queries.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.isWSUpgrade(r) {
		s.log.Debugf("upgrading connection to websocket")

		if s.options.WSContextFunc != nil {
			ctx = s.options.WSContextFunc(options.RequestTypeWS, r)
		}
		s.WSHandler(ctx, w, r)
		return
	}

	if s.options.ContextFunc != nil {
		ctx = s.options.ContextFunc(options.RequestTypeHTTP, r)
	}
	s.ContextHandler(ctx, w, r)
}
