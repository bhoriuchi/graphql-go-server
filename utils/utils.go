package utils

import (
	"encoding/json"
	"fmt"

	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// GetOperationAST
func GetOperationAST(nodes *ast.Document, operationName string) (*ast.OperationDefinition, error) {
	var operation *ast.OperationDefinition

	for _, def := range nodes.Definitions {
		switch def := def.(type) {
		case *ast.OperationDefinition:
			if operationName == "" && operation != nil {
				return nil, fmt.Errorf("must provide operation name if query contains multiple operations")
			}
			if operationName == "" || (def.GetName() != nil && def.GetName().Value == operationName) {
				operation = def
			}
		}
	}

	return operation, nil
}

func ParseQuery(query string) (*ast.Document, error) {
	return parser.Parse(parser.ParseParams{
		Source: source.NewSource(&source.Source{
			Body: []byte(query),
			Name: "GraphQL request",
		}),
	})
}

// ReMarshal converts one type to another
func ReMarshal(in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// GQLErrors
func GQLErrors(in interface{}) gqlerrors.FormattedErrors {
	switch v := in.(type) {
	case gqlerrors.FormattedErrors:
		return v
	case []gqlerrors.FormattedError:
		return v
	case []gqlerrors.Error:
		errs := gqlerrors.FormattedErrors{}
		for _, err := range v {
			formattedErr := gqlerrors.FormatError(err.OriginalError)
			formattedErr.Message = err.Message
			formattedErr.Locations = err.Locations
			formattedErr.Path = err.Path
			errs = append(errs, formattedErr)
		}
		return errs
	case []error:
		errs := gqlerrors.FormattedErrors{}
		for _, err := range v {
			errs = append(errs, gqlerrors.FormatError(err))
		}
		return errs
	case error:
		return gqlerrors.FormattedErrors{gqlerrors.FormatError(v)}
	}

	err := fmt.Errorf("unspecified error")
	return gqlerrors.FormattedErrors{gqlerrors.FormatError(err)}
}
