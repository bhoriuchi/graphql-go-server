package main

import (
	"net/http"

	server "github.com/bhoriuchi/graphql-go-server"
	"github.com/bhoriuchi/graphql-go-server/logger"
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
		server.WithLogFunc(logFunc),
		server.WithPretty(),
	)

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	l.Infof("Listening on %s", addr)
	http.ListenAndServe(addr, mux)
}
