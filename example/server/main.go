package main

import (
	"net/http"

	server "github.com/bhoriuchi/graphql-go-server"
	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/bhoriuchi/graphql-go-server/options"
)

var addr = ":3000"

func main() {
	logFunc := logger.NewSimpleLogFunc(logger.TraceLevel)
	l := logger.NewLogWrapper(
		logFunc,
		nil,
	)

	l.Infof("Building schema...")
	s, err := buildSchema(l)
	if err != nil {
		l.Errorf("Failed to build schema: %s", err)
		return
	}

	srv := server.New(
		*s,
		options.WithLogFunc(logFunc),
		options.WithPretty(),
	)

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	l.Infof("Listening on %s", addr)
	http.ListenAndServe(addr, mux)
}
