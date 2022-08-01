package gqlclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// Options client options
type Options struct {
	URL            string
	Before         []BeforeFunc
	Insecure       bool
	RequestTimeout time.Duration
	HTTPClient     *http.Client
}

// Client a graphql client
type Client struct {
	url        string
	before     []BeforeFunc
	httpClient *http.Client
}

// NewClient creates a new client
func NewClient(opts *Options) (client *Client, err error) {
	var httpClient *http.Client

	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = time.Duration(defaultRequestTimeout) * time.Second
	}

	if opts.HTTPClient != nil {
		httpClient = opts.HTTPClient
	} else {
		httpClient = &http.Client{
			Timeout: opts.RequestTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: opts.Insecure,
				},
			},
		}
	}

	client = &Client{
		url:        opts.URL,
		before:     opts.Before,
		httpClient: httpClient,
	}
	return
}

// Request performs a request
func (c *Client) Request(request Request) (rsp *Response, err error) {
	var body io.Reader
	rsp = &Response{}

	body, err = request.toReader()
	if err != nil {
		return
	}

	rsp.httpRequest, err = http.NewRequest(http.MethodPost, c.url, body)
	if err != nil {
		return nil, err
	}

	// apply before middleware
	for _, before := range c.before {
		if err = before(rsp.httpRequest); err != nil {
			return
		}
	}

	rsp.httpResponse, err = c.httpClient.Do(rsp.httpRequest)
	if err != nil {
		return
	}

	rsp.rawResult, err = ioutil.ReadAll(rsp.httpResponse.Body)
	if err != nil {
		return
	}
	defer rsp.httpResponse.Body.Close()

	var grsp graphQLResponse
	if err = json.Unmarshal(rsp.rawResult, &grsp); err != nil {
		return
	}

	rsp.data = grsp.Data
	if grsp.Errors != nil && len(grsp.Errors) > 0 {
		rsp.errors = grsp.Errors
	}

	if rsp.httpResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf(rsp.httpResponse.Status)
		return
	}

	return
}
