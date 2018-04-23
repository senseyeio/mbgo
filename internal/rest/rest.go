package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client represents a generic HTTP REST client that handles
// a JSON structure in requests and responses.
type Client struct {
	root *url.URL
	cli  *http.Client
}

// NewClient returns a new instance of *Client from the provided
// inner *http.Client cli and API base *url.URL root.
func NewClient(cli *http.Client, root *url.URL) *Client {
	return &Client{
		cli:  cli,
		root: root,
	}
}

// BuildRequest builds the specified *http.Request value from the
// provided request method, path, body and optional body/query
// parameters, with the appropriate headers set depending on
// the particular request method.
func (cli *Client) BuildRequest(method, path string, body io.Reader, q url.Values) (*http.Request, error) {
	u := *cli.root
	u.Path = path
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

	return req, nil
}

// Error represents the structure of an error received from the mountebank API.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Code, err.Message)
}

// ProcessRequest processes the provided *http.Request value req and
// decodes the response body JSON into the value pointed to by v if
// the response code matches the provided code, or decodes into an
// error otherwise.
func (cli *Client) ProcessRequest(req *http.Request, code int, v interface{}) error {
	resp, err := cli.cli.Do(req)
	if err != nil {
		return err
	}

	var closeErr error
	defer func() {
		closeErr = resp.Body.Close()
	}()

	if resp.StatusCode != code {
		dto := struct {
			Errors []Error
		}{}
		if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
			return err
		}
		// return the first decoded error, silently ignore the rest
		return dto.Errors[0]
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return err
	}
	return closeErr
}
