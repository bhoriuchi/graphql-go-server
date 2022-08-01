package gqlclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

const defaultRequestTimeout = 10

// BeforeFunc modifies the request before it is sent
type BeforeFunc func(req *http.Request) error

// Request interface to a request object so you can bring your own
type Request struct {
	Query         string
	OperationName string
	Variables     map[string]interface{}
}

// GetQuery gets the query
func (r *Request) GetQuery() string {
	return r.Query
}

// GetOperationName gets the operation name
func (r *Request) GetOperationName() string {
	return r.OperationName
}

// GetVariables gets the variables
func (r *Request) GetVariables() map[string]interface{} {
	if r.Variables == nil {
		return map[string]interface{}{}
	}
	return r.Variables
}

// converts the request to an io.Reader
func (r *Request) toReader() (body io.Reader, err error) {
	var j []byte
	j, err = json.Marshal(r)
	if err != nil {
		return
	}

	body = bytes.NewBuffer(j)
	return
}
