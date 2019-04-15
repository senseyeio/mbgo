package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// Client represents a generic HTTP REST client that handles
// a JSON structure in requests and responses.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewClient returns a new instance of *Client from the provided
// inner *http.Client httpClient and API base *url.URL baseURL.
func NewClient(cli *http.Client, root *url.URL) *Client {
	return &Client{
		httpClient: cli,
		baseURL:    root,
	}
}

// NewRequest builds the specified *http.Request value from the
// provided request method, path, body and optional body/query
// parameters, with the appropriate headers set depending on
// the particular request method.
func (cli *Client) NewRequest(ctx context.Context, method, path string, body io.Reader, q url.Values) (*http.Request, error) {
	u := cli.baseURL.ResolveReference(&url.URL{Path: path})
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	switch req.Method {
	case http.MethodPost, http.MethodPut:
		req.Header.Set("Content-Type", "application/json")
	}

	return req.WithContext(ctx), nil
}

// Do sends an HTTP request and returns an HTTP response.
func (cli *Client) Do(req *http.Request) (*http.Response, error) {
	return cli.httpClient.Do(req)
}

// DecodeResponseBody reads a JSON-encoded value from the provided
// HTTP response body and stores it into the value pointed to by v
// and closes the body after reading.
func (cli *Client) DecodeResponseBody(body io.ReadCloser, v interface{}) error {
	defer body.Close()

	return json.NewDecoder(body).Decode(v)
}
