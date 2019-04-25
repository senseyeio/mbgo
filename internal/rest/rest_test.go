// Copyright (c) 2018 Senseye Ltd. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package rest_test

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/senseyeio/mbgo/internal/assert"
	"github.com/senseyeio/mbgo/internal/rest"
)

func TestClient_NewRequest(t *testing.T) {
	cases := []struct {
		// general
		Description string

		// inputs
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
			Root:        &url.URL{},
			Method:      "bad method",
			Err:         errors.New(`net/http: invalid method "bad method"`),
		},
		{
			Description: "should construct the URL based on provided root URL, path and query parameters",
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

			cli := rest.NewClient(nil, c.Root)
			req, err := cli.NewRequest(c.Method, c.Path, c.Body, c.Query)
			assert.Equals(t, err, c.Err)
			assert.Equals(t, req, c.Request)
		})
	}
}

type testDTO struct {
	Test bool   `json:"test"`
	Foo  string `json:"foo"`
}

func TestClient_DecodeResponseBody(t *testing.T) {
	cases := []struct {
		// general
		Description string

		// inputs
		Body  io.ReadCloser
		Value interface{}

		// output expectations
		Expected interface{}
		Err      error
	}{
		{
			Description: "should return an error if the JSON cannot be decoded into the value pointer",
			Body:        ioutil.NopCloser(strings.NewReader(`"foo"`)),
			Value:       &testDTO{},
			Expected:    &testDTO{},
			Err: &json.UnmarshalTypeError{
				Offset: 5, // 5 bytes read before first full JSON value
				Value:  "string",
				Type:   reflect.TypeOf(testDTO{}),
			},
		},
		{
			Description: "should unmarshal the expected JSON into value pointer when valid",
			Body:        ioutil.NopCloser(strings.NewReader(`{"test":true,"foo":"bar"}`)),
			Value:       &testDTO{},
			Expected: &testDTO{
				Test: true,
				Foo:  "bar",
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Description, func(t *testing.T) {
			t.Parallel()

			cli := rest.NewClient(nil, nil)
			err := cli.DecodeResponseBody(c.Body, c.Value)
			assert.Equals(t, err, c.Err)
			assert.Equals(t, c.Value, c.Expected)
		})
	}
}
