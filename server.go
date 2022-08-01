package server

import (
	"net/http"
	"strings"

	"github.com/bhoriuchi/graphql-go-server/ide"
	"github.com/bhoriuchi/graphql-go-server/logger"
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
	options  *Options
	upgrader websocket.Upgrader
}

// New creates a new server
func New(schema graphql.Schema, opts ...Option) *Server {
	options := &Options{
		LogFunc:    logger.NoopLogFunc,
		Playground: ide.NewDefaultPlaygroundOptions(),
	}

	for _, opt := range opts {
		opt(options)
	}

	s := &Server{
		schema:  schema,
		log:     logger.NewLogWrapper(options.LogFunc, nil),
		options: options,
	}

	// define the supported subprotocols, the protocols are ordered by
	// priority and the negotiation process will pick the first match
	subprotocols := []string{}

	// prefer newer protocol
	if options.GraphQLTransportWS != nil {
		subprotocols = append(subprotocols, graphqltransportws.Subprotocol)
	}

	// fallback to older protocol
	if options.GraphQLWS != nil {
		subprotocols = append(subprotocols, graphqlws.Subprotocol)
	}

	if len(subprotocols) > 0 {
		s.upgrader = websocket.Upgrader{
			// TODO: make cors configurable
			CheckOrigin:  func(r *http.Request) bool { return true },
			Subprotocols: subprotocols,
		}
	}

	return s
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
		s.WSHandler(w, r)
		return
	}

	if s.options.ContextFunc != nil {
		ctx = s.options.ContextFunc(r)
	}
	s.ContextHandler(ctx, w, r)
}
