package rest_test

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/senseyeio/mbgo/internal/rest"
)

func TestClient_BuildRequest(t *testing.T) {
	cases := []struct {
		// general
		Description string

		// inputs
		Client *http.Client
		Root   *url.URL
		Method string
		Path   string
		Body   io.Reader
		Query  url.Values

		// output expectations
		Request *http.Request
		Err     error
	}{
		{
			Description: "should return an error if the provided request method is invalid",
			Client:      &http.Client{},
			Root:        &url.URL{},
			Method:      "bad method",
			Err:         errors.New(`net/http: invalid method "bad method"`),
		},
		{
			Description: "should construct the URL based on provided root URL, path and query parameters",
			Client:      &http.Client{},
			Root: &url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort("localhost", "2525"),
			},
			Method: http.MethodGet,
			Path:   "foo",
			Query: url.Values{
				"replayable": []string{"true"},
			},
			Request: &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Scheme:   "http",
					Host:     net.JoinHostPort("localhost", "2525"),
					Path:     "/foo",
					RawQuery: "replayable=true",
				},
				Host:       net.JoinHostPort("localhost", "2525"),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     http.Header{"Accept": []string{"application/json"}},
			},
		},
		{
			Description: "should only set the 'Accept' header if method is GET",
			Client:      &http.Client{},
			Root:        &url.URL{},
			Method:      http.MethodGet,
			Request: &http.Request{
				Method:     http.MethodGet,
				URL:        &url.URL{},
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     http.Header{"Accept": []string{"application/json"}},
			},
		},
		{
			Description: "should only set the 'Accept' header if method is DELETE",
			Client:      &http.Client{},
			Root:        &url.URL{},
			Method:      http.MethodDelete,
			Request: &http.Request{
				Method:     http.MethodDelete,
				URL:        &url.URL{},
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     http.Header{"Accept": []string{"application/json"}},
			},
		},
		{
			Description: "should set both the 'Accept' and 'Content-Type' headers if method is POST",
			Client:      &http.Client{},
			Root:        &url.URL{},
			Method:      http.MethodPost,
			Request: &http.Request{
				Method:     http.MethodPost,
				URL:        &url.URL{},
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header: http.Header{
					"Accept":       []string{"application/json"},
					"Content-Type": []string{"application/json"},
				},
			},
		},
		{
			Description: "should set both the 'Accept' and 'Content-Type' headers if method is PUT",
			Client:      &http.Client{},
			Root:        &url.URL{},
			Method:      http.MethodPut,
			Request: &http.Request{
				Method:     http.MethodPut,
				URL:        &url.URL{},
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header: http.Header{
					"Accept":       []string{"application/json"},
					"Content-Type": []string{"application/json"},
				},
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Description, func(t *testing.T) {
			t.Parallel()

			cli := rest.NewClient(c.Client, c.Root)
			req, err := cli.BuildRequest(c.Method, c.Path, c.Body, c.Query)
			if !reflect.DeepEqual(err, c.Err) {
				t.Errorf("expected %v to equal %v\n", err, c.Err)
			}
			if !reflect.DeepEqual(req, c.Request) {
				t.Errorf("expected\n%v\nto equal\n%v\n", req, c.Request)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	cases := []struct {
		Description string
		Error       rest.Error
		Expected    string
	}{
		{
			Description: "foo",
			Error: rest.Error{
				Code:    "code",
				Message: "message",
			},
			Expected: "code: message",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Description, func(t *testing.T) {
			t.Parallel()

			actual := c.Error.Error()
			if actual != c.Expected {
				t.Errorf("expected %v to equal %v", actual, c.Expected)
			}
		})
	}
}
