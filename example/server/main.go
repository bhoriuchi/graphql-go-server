package main

import (
	"net/http"

	server "github.com/bhoriuchi/graphql-go-server"
	"github.com/bhoriuchi/graphql-go-server/logger"
)

var addr = ":3000"

func main() {
	l := logger.NewSimpleLogger()
	l.SetLevel(logger.TraceLevel)
	l.Infof("Building schema...")
	s, err := buildSchema(l)
	if err != nil {
		l.Errorf("Failed to build schema: %s", err)
		return
	}

	srv := server.New(
		*s,
		server.WithLogger(l),
		server.WithPretty(),
	)

	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)
	l.Infof("Listening on %s", addr)
	http.ListenAndServe(addr, mux)
}
