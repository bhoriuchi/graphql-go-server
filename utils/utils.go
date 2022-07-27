package utils

import (
	"fmt"

	"github.com/graphql-go/graphql/language/ast"
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
