package gqlclient

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/graphql-go/graphql/gqlerrors"
)

// graphql json response
type graphQLResponse struct {
	Data   interface{}               `json:"data"`
	Errors gqlerrors.FormattedErrors `json:"errors"`
}

// Response response object
type Response struct {
	httpRequest  *http.Request
	httpResponse *http.Response
	rawResult    []byte
	data         interface{}
	errors       gqlerrors.FormattedErrors
}

// HTTPRequest returns the http request
func (c *Response) HTTPRequest() *http.Request {
	return c.httpRequest
}

// HTTPResponse returns the http response
func (c *Response) HTTPResponse() *http.Response {
	return c.httpResponse
}

// RawResult returns the raw result body
func (c *Response) RawResult() []byte {
	return c.rawResult
}

// Data returns the data
func (c *Response) Data() interface{} {
	return c.data
}

// Errors returns the errors
func (c *Response) Errors() gqlerrors.FormattedErrors {
	return c.errors
}

// FirstError returns the first error
func (c *Response) FirstError() *gqlerrors.FormattedError {
	if c.HasErrors() {
		first := c.errors[0]
		return &first
	}
	return nil
}

// HasErrors returns true if errors are present
func (c *Response) HasErrors() bool {
	return c.errors != nil && len(c.errors) > 0
}

// Decode decodes the result into the provided interface
func (c *Response) Decode(out interface{}) (err error) {
	var j []byte
	if len(c.rawResult) == 0 {
		err = fmt.Errorf("no data to decode")
		return
	}

	j, err = json.Marshal(c.data)
	if err != nil {
		return
	}

	err = json.Unmarshal(j, out)
	return
}
