package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bhoriuchi/graphql-go-server/ide"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol/graphqltransportws"
	"github.com/bhoriuchi/graphql-go-server/ws/protocol/graphqlws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// RequestOptions options
type RequestOptions struct {
	Query         string                 `json:"query" url:"query" schema:"query"`
	Variables     map[string]interface{} `json:"variables" url:"variables" schema:"variables"`
	OperationName string                 `json:"operationName" url:"operationName" schema:"operationName"`
}

// a workaround for getting`variables` as a JSON string
type requestOptionsCompatibility struct {
	Query         string `json:"query" url:"query" schema:"query"`
	Variables     string `json:"variables" url:"variables" schema:"variables"`
	OperationName string `json:"operationName" url:"operationName" schema:"operationName"`
}

func getFromForm(values url.Values) *RequestOptions {
	query := values.Get("query")
	if query != "" {
		// get variables map
		variables := make(map[string]interface{}, len(values))
		variablesStr := values.Get("variables")
		json.Unmarshal([]byte(variablesStr), &variables)

		return &RequestOptions{
			Query:         query,
			Variables:     variables,
			OperationName: values.Get("operationName"),
		}
	}

	return nil
}

// NewRequestOptions Parses a http.Request into GraphQL request options struct
func NewRequestOptions(r *http.Request) *RequestOptions {
	if reqOpt := getFromForm(r.URL.Query()); reqOpt != nil {
		return reqOpt
	}

	if r.Method != http.MethodPost {
		return &RequestOptions{}
	}

	if r.Body == nil {
		return &RequestOptions{}
	}

	// TODO: improve Content-Type handling
	contentTypeStr := r.Header.Get("Content-Type")
	contentTypeTokens := strings.Split(contentTypeStr, ";")
	contentType := contentTypeTokens[0]

	switch contentType {
	case ContentTypeGraphQL:
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &RequestOptions{}
		}
		return &RequestOptions{
			Query: string(body),
		}
	case ContentTypeFormURLEncoded:
		if err := r.ParseForm(); err != nil {
			return &RequestOptions{}
		}

		if reqOpt := getFromForm(r.PostForm); reqOpt != nil {
			return reqOpt
		}

		return &RequestOptions{}

	case ContentTypeJSON:
		fallthrough
	default:
		var opts RequestOptions
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &opts
		}
		err = json.Unmarshal(body, &opts)
		if err != nil {
			// Probably `variables` was sent as a string instead of an object.
			// So, we try to be polite and try to parse that as a JSON string
			var optsCompatible requestOptionsCompatibility
			json.Unmarshal(body, &optsCompatible)
			json.Unmarshal([]byte(optsCompatible.Variables), &opts.Variables)
		}
		return &opts
	}
}

// GetRequestOptions Parses a http.Request into GraphQL request options struct without clearning the body
func GetRequestOptions(r *http.Request) *RequestOptions {
	if reqOpt := getFromForm(r.URL.Query()); reqOpt != nil {
		return reqOpt
	}

	if r.Method != http.MethodPost {
		return &RequestOptions{}
	}

	if r.Body == nil {
		return &RequestOptions{}
	}

	// TODO: improve Content-Type handling
	contentTypeStr := r.Header.Get("Content-Type")
	contentTypeTokens := strings.Split(contentTypeStr, ";")
	contentType := contentTypeTokens[0]

	switch contentType {
	case ContentTypeGraphQL:
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &RequestOptions{}
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		return &RequestOptions{
			Query: string(body),
		}
	case ContentTypeFormURLEncoded:
		if err := r.ParseForm(); err != nil {
			return &RequestOptions{}
		}

		if reqOpt := getFromForm(r.PostForm); reqOpt != nil {
			return reqOpt
		}

		return &RequestOptions{}

	case ContentTypeJSON:
		fallthrough
	default:
		var opts RequestOptions
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &opts
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		err = json.Unmarshal(body, &opts)
		if err != nil {
			// Probably `variables` was sent as a string instead of an object.
			// So, we try to be polite and try to parse that as a JSON string
			var optsCompatible requestOptionsCompatibility
			json.Unmarshal(body, &optsCompatible)
			json.Unmarshal([]byte(optsCompatible.Variables), &opts.Variables)
		}
		return &opts
	}
}

// ContextHandler provides an entrypoint into executing graphQL queries with a
// user-provided context.
func (s *Server) ContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// get query
	opts := NewRequestOptions(r)

	// execute graphql query
	params := graphql.Params{
		Schema:         s.schema,
		RequestString:  opts.Query,
		VariableValues: opts.Variables,
		OperationName:  opts.OperationName,
		Context:        ctx,
	}

	if s.options.RootValueFunc != nil {
		params.RootObject = s.options.RootValueFunc(ctx, r)
	}

	if params.RootObject == nil {
		params.RootObject = map[string]interface{}{}
	}

	result := graphql.Do(params)

	if formatErrorFunc := s.options.FormatErrorFunc; formatErrorFunc != nil && len(result.Errors) > 0 {
		formatted := make([]gqlerrors.FormattedError, len(result.Errors))
		for i, formattedError := range result.Errors {
			formatted[i] = formatErrorFunc(formattedError.OriginalError())
		}
		result.Errors = formatted
	}

	if s.options.GraphiQL != nil {
		acceptHeader := r.Header.Get("Accept")
		_, raw := r.URL.Query()["raw"]
		if !raw && !strings.Contains(acceptHeader, "application/json") && strings.Contains(acceptHeader, "text/html") {
			ide.RenderGraphiQL(s.options.GraphiQL, w, r, params)
			return
		}
	} else if s.options.Playground != nil {
		acceptHeader := r.Header.Get("Accept")
		_, raw := r.URL.Query()["raw"]
		if !raw && !strings.Contains(acceptHeader, "application/json") && strings.Contains(acceptHeader, "text/html") {
			ide.RenderPlayground(s.options.Playground, w, r)
			return
		}
	}

	// use proper JSON Header
	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	var buff []byte
	if s.options.Pretty {
		w.WriteHeader(http.StatusOK)
		buff, _ = json.MarshalIndent(result, "", "\t")

		w.Write(buff)
	} else {
		w.WriteHeader(http.StatusOK)
		buff, _ = json.Marshal(result)

		w.Write(buff)
	}

	if s.options.ResultCallbackFunc != nil {
		s.options.ResultCallbackFunc(ctx, &params, result, buff)
	}
}

// WSHandler handles websocket connection upgrade
func (s *Server) WSHandler(w http.ResponseWriter, r *http.Request) {
	// Establish a WebSocket connection
	s.log.Debugf("upgrading connection to websocket")
	var ws, err = s.upgrader.Upgrade(w, r, nil)

	// Bail out if the WebSocket connection could not be established
	if err != nil {
		s.log.Warnf("Failed to establish WebSocket connection", err)
		return
	}

	s.log.Debugf("Client requested %q subprotocol", ws.Subprotocol())

	// Close the connection early if it doesn't implement a supported protocol
	switch ws.Subprotocol() {
	// graphql-ws protocol
	case graphqlws.Subprotocol:
		if s.options.GraphQLWS == nil {
			s.log.Warnf("Connection does not implement the GraphQL WS protocol. Subprotocol: %q", ws.Subprotocol())
			s.closeWS(
				ws,
				websocket.CloseProtocolError,
				"server does not support %q protocol",
				ws.Subprotocol(),
			)
			return
		}

		graphqlws.NewConnection(r.Context(), graphqlws.Config{
			WS:                  ws,
			Schema:              &s.schema,
			Logger:              s.log,
			Request:             r,
			KeepAlive:           s.options.GraphQLWS.KeepAlive,
			RootValueFunc:       s.options.GraphQLWS.RootValueFunc,
			ContextValueFunc:    s.options.GraphQLWS.ContextValueFunc,
			OnConnect:           s.options.GraphQLWS.OnConnect,
			OnDisconnect:        s.options.GraphQLWS.OnDisconnect,
			OnOperation:         s.options.GraphQLWS.OnOperation,
			OnOperationComplete: s.options.GraphQLWS.OnOperationComplete,
		})

	// graphql-transport-ws protocol
	case graphqltransportws.Subprotocol:
		if s.options.GraphQLTransportWS == nil {
			s.log.Warnf("Connection does not implement the GraphQL WS protocol. Subprotocol: %q", ws.Subprotocol())
			s.closeWS(
				ws,
				websocket.CloseProtocolError,
				"server does not support %q protocol",
				ws.Subprotocol(),
			)
			return
		}

		graphqltransportws.NewConnection(r.Context(), graphqltransportws.Config{
			WS:                        ws,
			Schema:                    &s.schema,
			Logger:                    s.log,
			Request:                   r,
			ConnectionInitWaitTimeout: s.options.GraphQLTransportWS.ConnectionInitWaitTimeout,
			RootValueFunc:             s.options.GraphQLTransportWS.RootValueFunc,
			ContextValueFunc:          s.options.GraphQLTransportWS.ContextValueFunc,
			OnConnect:                 s.options.GraphQLTransportWS.OnConnect,
			OnPing:                    s.options.GraphQLTransportWS.OnPing,
			OnPong:                    s.options.GraphQLTransportWS.OnPong,
			OnDisconnect:              s.options.GraphQLTransportWS.OnDisconnect,
			OnClose:                   s.options.GraphQLTransportWS.OnClose,
			OnSubscribe:               s.options.GraphQLTransportWS.OnSubscribe,
			OnNext:                    s.options.GraphQLTransportWS.OnNext,
			OnError:                   s.options.GraphQLTransportWS.OnError,
			OnComplete:                s.options.GraphQLTransportWS.OnComplete,
			OnOperation:               s.options.GraphQLTransportWS.OnOperation,
		})

	default:
		s.log.Warnf("Connection does not implement the GraphQL WS protocol. Subprotocol: %q", ws.Subprotocol())
		s.closeWS(ws, websocket.CloseProtocolError, "Connection does not implement a supported GraphQL subprotocol")
	}
}

// func closeWS closes the websocket
func (s *Server) closeWS(ws *websocket.Conn, code int, reason string, v ...interface{}) {
	deadline := time.Now().Add(100 * time.Millisecond)
	msg := websocket.FormatCloseMessage(
		code,
		fmt.Sprintf(reason, v...),
	)

	if err := ws.WriteControl(websocket.CloseMessage, msg, deadline); err != nil {
		if err != websocket.ErrCloseSent {
			if err := ws.Close(); err != nil {
				s.log.WithError(err).Errorf("failed to close websocket")
			}
		}
	}
}
