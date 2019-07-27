// Copyright (c) 2018 Senseye Ltd. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

// +build integration

package mbgo_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/senseyeio/mbgo"
	"github.com/senseyeio/mbgo/internal/assert"
)

// newMountebankClient creates a new mountebank client instance pointing to the host
// denoted by the MB_HOST environment variable, or localhost:2525 if blank.
func newMountebankClient() *mbgo.Client {
	return mbgo.NewClient(&http.Client{
		Timeout: time.Second,
	}, &url.URL{
		Scheme: "http",
		Host:   "localhost:2525",
	})
}

// newContext returns a new context instance with the given timeout.
func newContext(timeout time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return ctx
}

func TestClient_Logs(t *testing.T) {
	mb := newMountebankClient()

	vs, err := mb.Logs(newContext(time.Second), -1, -1)
	assert.Equals(t, err, nil)
	assert.Equals(t, len(vs) >= 2, true)
	assert.Equals(t, vs[0].Message, "[mb:2525] mountebank v2.0.0 now taking orders - point your browser to http://localhost:2525/ for help")
	assert.Equals(t, vs[1].Message, "[mb:2525] GET /logs")
}

func TestClient_Create(t *testing.T) {
	mb := newMountebankClient()

	cases := []struct {
		// general
		Description string
		Before      func(*testing.T, *mbgo.Client)
		After       func(*testing.T, *mbgo.Client)

		// input
		Input mbgo.Imposter

		// output expectations
		Expected *mbgo.Imposter
		Err      error
	}{
		{
			Description: "should error if an invalid port is provided",
			Input: mbgo.Imposter{
				Proto: "http",
				Port:  328473289572983424,
			},
			Err: errors.New("bad data: invalid value for 'port'"),
		},
		{
			Description: "should error if an invalid protocol is provided",
			Input: mbgo.Imposter{
				Proto: "udp",
				Port:  8080,
			},
			Err: errors.New("bad data: the udp protocol is not yet supported"),
		},
		{
			Description: "should create the expected HTTP Imposter on success",
			Input: mbgo.Imposter{
				Proto:          "http",
				Port:           8080,
				Name:           "create_test",
				RecordRequests: true,
				AllowCORS:      true,
				Stubs: []mbgo.Stub{
					{
						Predicates: []mbgo.Predicate{
							{
								Operator: "equals",
								Request: mbgo.HTTPRequest{
									Method: http.MethodGet,
									Path:   "/foo",
									Query: map[string][]string{
										"page": {"3"},
									},
									Headers: map[string][]string{
										"Accept": {"application/json"},
									},
								},
							},
						},
						Responses: []mbgo.Response{
							{
								Type: "is",
								Value: mbgo.HTTPResponse{
									StatusCode: http.StatusOK,
									Headers: map[string][]string{
										"Content-Type": {"application/json"},
									},
									Body: `{"test":true}`,
								},
							},
						},
					},
				},
			},
			Before: func(t *testing.T, mb *mbgo.Client) {
				_, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)
			},
			After: func(t *testing.T, mb *mbgo.Client) {
				imp, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)
				assert.Equals(t, imp.Name, "create_test")
			},
			Expected: &mbgo.Imposter{
				Proto:          "http",
				Port:           8080,
				Name:           "create_test",
				RecordRequests: true,
				AllowCORS:      true,
				RequestCount:   0,
				Stubs: []mbgo.Stub{
					{
						Predicates: []mbgo.Predicate{
							{
								Operator: "equals",
								Request: mbgo.HTTPRequest{
									Method: http.MethodGet,
									Path:   "/foo",
									Query: map[string][]string{
										"page": {"3"},
									},
									Headers: map[string][]string{
										"Accept": {"application/json"},
									},
								},
							},
						},
						Responses: []mbgo.Response{
							{
								Type: "is",
								Value: mbgo.HTTPResponse{
									StatusCode: http.StatusOK,
									Headers: map[string][]string{
										"Content-Type": {"application/json"},
									},
									Body: `{"test":true}`,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(t, mb)
			}

			actual, err := mb.Create(newContext(time.Second), c.Input)
			assert.Equals(t, err, c.Err)
			assert.Equals(t, actual, c.Expected)

			if c.After != nil {
				c.After(t, mb)
			}
		})
	}
}

func TestClient_Imposter(t *testing.T) {
	mb := newMountebankClient()

	cases := []struct {
		// general
		Description string
		Before      func(*testing.T, *mbgo.Client)
		After       func(*testing.T, *mbgo.Client)

		// input
		Port   int
		Replay bool

		// output expectations
		Expected *mbgo.Imposter
		Err      error
	}{
		{
			Description: "should error if an Imposter does not exist on the specified port",
			Port:        8080,
			Before: func(t *testing.T, mb *mbgo.Client) {
				_, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)
			},
			Err: errors.New("no such resource: Try POSTing to /imposters first?"),
		},
		{
			Description: "should return the expected TCP Imposter if it exists on the specified port",
			Before: func(t *testing.T, mb *mbgo.Client) {
				_, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)

				imp, err := mb.Create(newContext(time.Second), mbgo.Imposter{
					Port:           8080,
					Proto:          "tcp",
					Name:           "imposter_test",
					RecordRequests: true,
					Stubs: []mbgo.Stub{
						{
							Predicates: []mbgo.Predicate{
								{
									Operator: "endsWith",
									Request: mbgo.TCPRequest{
										Data: "SGVsbG8sIHdvcmxkIQ==",
									},
								},
							},
							Responses: []mbgo.Response{
								{
									Type: "is",
									Value: mbgo.TCPResponse{
										Data: "Z2l0aHViLmNvbS9zZW5zZXllaW8vbWJnbw==",
									},
								},
							},
						},
					},
				})
				assert.Equals(t, err, nil)
				assert.Equals(t, imp.Name, "imposter_test")
			},
			After: func(t *testing.T, mb *mbgo.Client) {
				imp, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)
				assert.Equals(t, imp.Name, "imposter_test")
			},
			Port:   8080,
			Replay: false,
			Expected: &mbgo.Imposter{
				Port:           8080,
				Proto:          "tcp",
				Name:           "imposter_test",
				RecordRequests: false, // this field is only used for creation
				RequestCount:   0,
				Stubs: []mbgo.Stub{
					{
						Predicates: []mbgo.Predicate{
							{
								Operator: "endsWith",
								Request: mbgo.TCPRequest{
									Data: "SGVsbG8sIHdvcmxkIQ==",
								},
							},
						},
						Responses: []mbgo.Response{
							{
								Type: "is",
								Value: mbgo.TCPResponse{
									Data: "Z2l0aHViLmNvbS9zZW5zZXllaW8vbWJnbw==",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(t, mb)
			}

			actual, err := mb.Imposter(newContext(time.Second), c.Port, c.Replay)
			assert.Equals(t, err, c.Err)
			assert.Equals(t, actual, c.Expected)

			if c.After != nil {
				c.After(t, mb)
			}
		})
	}
}

func TestClient_Delete(t *testing.T) {
	mb := newMountebankClient()

	cases := []struct {
		// general
		Description string
		Before      func(*mbgo.Client)
		After       func(*mbgo.Client)

		// input
		Port   int
		Replay bool

		// output expectations
		Expected *mbgo.Imposter
		Err      error
	}{
		{
			Description: "should return an empty Imposter struct if one is not configured on the specified port",
			Port:        8080,
			Before: func(mb *mbgo.Client) {
				_, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)
			},
			Expected: &mbgo.Imposter{},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(mb)
			}

			actual, err := mb.Delete(newContext(time.Second), c.Port, c.Replay)
			assert.Equals(t, err, c.Err)
			assert.Equals(t, actual, c.Expected)

			if c.After != nil {
				c.After(mb)
			}
		})
	}
}

func TestClient_DeleteRequests(t *testing.T) {
	mb := newMountebankClient()

	cases := []struct {
		// general
		Description string
		Before      func(*testing.T, *mbgo.Client)
		After       func(*testing.T, *mbgo.Client)

		// input
		Port int

		// output expectations
		Expected *mbgo.Imposter
		Err      error
	}{
		{
			Description: "should return an empty Imposter struct if one is not configured on the specified port",
			Before: func(t *testing.T, mb *mbgo.Client) {
				_, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)
			},
			Port:     8080,
			Expected: &mbgo.Imposter{},
		},
		{
			Description: "should return the expected Imposter if it exists on successful deletion",
			Before: func(t *testing.T, mb *mbgo.Client) {
				_, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)

				_, err = mb.Create(newContext(time.Second), mbgo.Imposter{
					Port:           8080,
					Proto:          "http",
					Name:           "delete_requests_test",
					RecordRequests: true,
				})
				assert.Equals(t, err, nil)
			},
			After: func(t *testing.T, mb *mbgo.Client) {
				imp, err := mb.Delete(newContext(time.Second), 8080, false)
				assert.Equals(t, err, nil)
				assert.Equals(t, imp.Name, "delete_requests_test")
			},
			Port: 8080,
			Expected: &mbgo.Imposter{
				Port:         8080,
				Proto:        "http",
				Name:         "delete_requests_test",
				RequestCount: 0,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(t, mb)
			}

			actual, err := mb.DeleteRequests(newContext(time.Second), c.Port)
			assert.Equals(t, err, c.Err)

			for i := 0; i < len(actual.Requests); i++ {
				req := actual.Requests[i].(mbgo.HTTPRequest)
				ts := req.Timestamp
				if len(ts) == 0 {
					t.Errorf("expected non-empty timestamp in %v", req)
				}
				// clear out the timestamp before doing a deep equality check
				// see https://github.com/senseyeio/mbgo/pull/5 for details
				req.Timestamp = ""
				actual.Requests[i] = req
			}

			assert.Equals(t, actual, c.Expected)

			if c.After != nil {
				c.After(t, mb)
			}
		})
	}
}

func TestClient_Config(t *testing.T) {
	mb := newMountebankClient()

	cfg, err := mb.Config(newContext(time.Second))
	assert.Equals(t, err, nil)
	assert.Equals(t, cfg.Version, "2.0.0")
}

func TestClient_Imposters(t *testing.T) {
	cases := []struct {
		// general
		Description string
		Before      func(*testing.T, *mbgo.Client)
		After       func(*testing.T, *mbgo.Client)

		// input
		Replay bool

		// output expectations
		Expected []mbgo.Imposter
		Err      error
	}{
		{
			Description: "should return a minimal representation of all registered Imposters",
			Before: func(t *testing.T, mb *mbgo.Client) {
				_, err := mb.DeleteAll(newContext(time.Second), false)
				assert.Equals(t, err, nil)

				// create a tcp imposter
				imp, err := mb.Create(newContext(time.Second), mbgo.Imposter{
					Port:           8080,
					Proto:          "tcp",
					Name:           "imposters_tcp_test",
					RecordRequests: true,
					Stubs: []mbgo.Stub{
						{
							Predicates: []mbgo.Predicate{
								{
									Operator: "endsWith",
									Request: mbgo.TCPRequest{
										Data: "SGVsbG8sIHdvcmxkIQ==",
									},
								},
							},
							Responses: []mbgo.Response{
								{
									Type: "is",
									Value: mbgo.TCPResponse{
										Data: "Z2l0aHViLmNvbS9zZW5zZXllaW8vbWJnbw==",
									},
								},
							},
						},
					},
				})
				assert.Equals(t, err, nil)
				assert.Equals(t, imp.Name, "imposters_tcp_test")

				// and an http imposter
				imp, err = mb.Create(newContext(time.Second), mbgo.Imposter{
					Proto:          "http",
					Port:           8081,
					Name:           "imposters_http_test",
					RecordRequests: true,
					AllowCORS:      true,
					Stubs: []mbgo.Stub{
						{
							Predicates: []mbgo.Predicate{
								{
									Operator: "equals",
									Request: mbgo.HTTPRequest{
										Method: http.MethodGet,
										Path:   "/foo",
										Query: map[string][]string{
											"page": {"3"},
										},
										Headers: map[string][]string{
											"Accept": {"application/json"},
										},
									},
								},
							},
							Responses: []mbgo.Response{
								{
									Type: "is",
									Value: mbgo.HTTPResponse{
										StatusCode: http.StatusOK,
										Headers: map[string][]string{
											"Content-Type": {"application/json"},
										},
										Body: `{"test":true}`,
									},
								},
							},
						},
					},
				})
				assert.Equals(t, err, nil)
				assert.Equals(t, imp.Name, "imposters_http_test")
			},
			Expected: []mbgo.Imposter{
				{
					Port:         8080,
					Proto:        "tcp",
					RequestCount: 0,
				},
				{
					Port:         8081,
					Proto:        "http",
					RequestCount: 0,
				},
			},
		},
	}

	mb := newMountebankClient()

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(t, mb)
			}

			actual, err := mb.Imposters(newContext(time.Second), c.Replay)
			assert.Equals(t, err, c.Err)
			assert.Equals(t, actual, c.Expected)

			if c.After != nil {
				c.After(t, mb)
			}
		})
	}
}
