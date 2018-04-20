package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	root *url.URL
	cli  *http.Client
}

func NewClient(cli *http.Client, root *url.URL) *Client {
	return &Client{
		cli:  cli,
		root: root,
	}
}

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

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Code, err.Message)
}

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
