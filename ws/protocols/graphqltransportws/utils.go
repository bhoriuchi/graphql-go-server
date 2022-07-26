package graphqltransportws

import (
	"encoding/json"

	"github.com/graphql-go/graphql/gqlerrors"
)

func isObject(val interface{}) bool {
	switch val.(type) {
	case map[string]interface{}, []interface{}, map[interface{}]interface{}:
		return true
	}

	return false
}

func areGraphQLErrors(val interface{}) bool {
	switch errs := val.(type) {
	case []*gqlerrors.Error:
		if len(errs) == 0 {
			return false
		}
		return true
	}

	return false
}

// ReMarshal converts one type to another
func ReMarshal(in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
