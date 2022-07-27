package main

import (
	"fmt"
	"time"

	"github.com/bhoriuchi/graphql-go-server/logger"
	"github.com/graphql-go/graphql"
)

func buildSchema(l logger.Logger) (*graphql.Schema, error) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"hello": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return "world", nil
					},
				},
			},
		}),
		Subscription: graphql.NewObject(graphql.ObjectConfig{
			Name: "Subscription",
			Fields: graphql.Fields{
				"watch": &graphql.Field{
					Type: graphql.String,
					Args: graphql.FieldConfigArgument{
						"iterations": &graphql.ArgumentConfig{
							Type:         graphql.Int,
							DefaultValue: 10,
						},
						"waitSeconds": &graphql.ArgumentConfig{
							Type:         graphql.Int,
							DefaultValue: 2,
						},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Source, nil
					},
					Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
						iterations := p.Args["iterations"].(int)
						waitSeconds := p.Args["waitSeconds"].(int)
						waitDuration := time.Duration(waitSeconds) * time.Second

						c := make(chan interface{})
						go func() {
							for i := 0; i < iterations; i++ {
								time.Sleep(waitDuration)
								msg := fmt.Sprintf("Iteration %d of %d", i+1, iterations)
								l.Tracef("Sending message: %q", msg)

								select {
								case <-p.Context.Done():
									close(c)
									return
								case c <- msg:
								}
							}
							l.Tracef("Closing channel")
							close(c)
						}()

						return c, nil
					},
				},
			},
		}),
	})

	if err != nil {
		return nil, err
	}

	return &schema, nil
}
