// Copyright (c) 2018 Senseye Ltd. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

// Package mbgo implements a mountebank API client.
package mbgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/senseyeio/mbgo/internal/rest"
)

// Client represents a native client to the mountebank REST API.
type Client struct {
	restCli *rest.Client
}

// NewClient returns a new instance of *Client given its underlying
// *http.Client restCli and base *url.URL to the mountebank API root.
//
// If nil, defaults the root *url.URL value to point to http://localhost:2525.
func NewClient(cli *http.Client, root *url.URL) *Client {
	if root == nil {
		root = &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort("localhost", "2525"),
		}
	}
	return &Client{
		restCli: rest.NewClient(cli, root),
	}
}

// errorDTO represents the structure of an error received from the mountebank API.
type errorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// decodeError is a helper method used to decode an errorDTO structure from the
// given response body, usually when an unexpected response code is returned.
func (cli *Client) decodeError(body io.ReadCloser) error {
	var wrap struct {
		Errors []errorDTO `json:"errors"`
	}
	if err := cli.restCli.DecodeResponseBody(body, &wrap); err != nil {
		return err
	}
	// Silently ignore all but the first error value if multiple are returned
	dto := wrap.Errors[0]
	return fmt.Errorf("%s: %s", dto.Code, dto.Message)
}

// Create creates a single new Imposter given its creation details imp.
//
// Note that the Imposter.RequestCount field is not used during creation.
//
// See more information on this resource at:
// http://www.mbtest.org/docs/api/overview#post-imposters.
func (cli *Client) Create(ctx context.Context, imp Imposter) (*Imposter, error) {
	p := "/imposters"
	b, err := json.Marshal(&imp)
	if err != nil {
		return nil, err
	}

	req, err := cli.restCli.NewRequest(ctx, http.MethodPost, p, bytes.NewReader(b), nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusCreated {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}

	return &imp, nil
}

// Imposter retrieves the Imposter data at the given port.
//
// Note that the Imposter.RecordRequests and Imposter.AllowCORS fields
// are ignored when un-marshalling an Imposter value and should only be
// used when creating an Imposter.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#get-imposter.
func (cli *Client) Imposter(ctx context.Context, port int, replay bool) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d", port)
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.restCli.NewRequest(ctx, http.MethodGet, p, nil, vs)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}

	return &imp, nil
}

// AddStub adds a new Stub without restarting its Imposter given the imposter's
// port and the new stub's index, or simply to the end of the array if index < 0.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#add-stub
func (cli *Client) AddStub(ctx context.Context, port, index int, stub Stub) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d/stubs", port)

	dto := map[string]interface{}{"stub": stub}
	if index >= 0 {
		dto["index"] = index
	}
	b, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}

	req, err := cli.restCli.NewRequest(ctx, http.MethodPost, p, bytes.NewReader(b), nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return &imp, nil
}

// OverwriteStub overwrites an existing Stub without restarting its Imposter,
// where the stub index denotes the stub to be changed.
//
// See more information about this resouce at:
// http://www.mbtest.org/docs/api/overview#change-stub
func (cli *Client) OverwriteStub(ctx context.Context, port, index int, stub Stub) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d/stubs/%d", port, index)

	b, err := json.Marshal(stub)
	if err != nil {
		return nil, err
	}

	req, err := cli.restCli.NewRequest(ctx, http.MethodPut, p, bytes.NewReader(b), nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return &imp, nil
}

// OverwriteAllStubs overwrites all existing Stubs without restarting their Imposter.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#change-stubs
func (cli *Client) OverwriteAllStubs(ctx context.Context, port int, stubs []Stub) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d/stubs", port)

	b, err := json.Marshal(map[string]interface{}{
		"stubs": stubs,
	})
	if err != nil {
		return nil, err
	}

	req, err := cli.restCli.NewRequest(ctx, http.MethodPut, p, bytes.NewReader(b), nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return &imp, nil
}

// RemoveStub removes a Stub without restarting its Imposter.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#delete-stub
func (cli *Client) RemoveStub(ctx context.Context, port, index int) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d/stubs/%d", port, index)

	req, err := cli.restCli.NewRequest(ctx, http.MethodDelete, p, http.NoBody, nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return &imp, nil
}

// Delete removes an Imposter configured on the given port and returns
// the deleted Imposter data, or an empty Imposter struct if one does not
// exist on the port.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#delete-imposter.
func (cli *Client) Delete(ctx context.Context, port int, replay bool) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d", port)
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.restCli.NewRequest(ctx, http.MethodDelete, p, nil, vs)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return &imp, nil
}

// DeleteRequests removes any recorded requests associated with the
// Imposter on the given port and returns the Imposter including the
// deleted requests, or an empty Imposter struct if one does not exist
// on the port.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#delete-imposter-requests.
func (cli *Client) DeleteRequests(ctx context.Context, port int) (*Imposter, error) {
	p := fmt.Sprintf("/imposters/%d/savedProxyResponses", port)

	req, err := cli.restCli.NewRequest(ctx, http.MethodDelete, p, nil, nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var imp Imposter
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &imp); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return &imp, nil
}

type imposterListWrapper struct {
	Imposters []Imposter `json:"imposters"`
}

// Overwrite is used to overwrite all registered Imposters with a new
// set of Imposters. This call is destructive, removing all previous
// Imposters even if the new set of Imposters do not conflict with
// previously registered protocols/ports.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#put-imposters.
func (cli *Client) Overwrite(ctx context.Context, imps []Imposter) ([]Imposter, error) {
	p := "/imposters"

	b, err := json.Marshal(&struct {
		Imposters []Imposter `json:"imposters"`
	}{
		Imposters: imps,
	})
	if err != nil {
		return nil, err
	}

	req, err := cli.restCli.NewRequest(ctx, http.MethodPut, p, bytes.NewReader(b), nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var wrap imposterListWrapper
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &wrap); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return wrap.Imposters, nil
}

// Imposters retrieves a list of all Imposters registered in mountebank.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#get-imposters.
func (cli *Client) Imposters(ctx context.Context, replay bool) ([]Imposter, error) {
	p := "/imposters"
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.restCli.NewRequest(ctx, http.MethodGet, p, nil, vs)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var wrap imposterListWrapper
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &wrap); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return wrap.Imposters, nil
}

// DeleteAll removes all registered Imposters from mountebank and closes
// their listening socket. This is the surest way to reset mountebank
// between test runs.
//
// See more information about this resource at:
// http://www.mbtest.org/docs/api/overview#delete-imposters.
func (cli *Client) DeleteAll(ctx context.Context, replay bool) ([]Imposter, error) {
	p := "/imposters"
	vs := url.Values{}
	vs.Add("replayable", strconv.FormatBool(replay))

	req, err := cli.restCli.NewRequest(ctx, http.MethodDelete, p, nil, vs)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var wrap imposterListWrapper
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &wrap); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return wrap.Imposters, nil
}

// Config represents information about the configuration of the mountebank
// server runtime, including its version, options and runtime information.
//
// See more information about its full structure at:
// http://www.mbtest.org/docs/api/contracts?type=config.
type Config struct {
	// Version represents the mountebank version in semantic M.m.p format.
	Version string `json:"version"`

	// Options represent runtime options of the mountebank server process.
	Options struct {
		Help           bool     `json:"help"`
		NoParse        bool     `json:"noParse"`
		NoLogFile      bool     `json:"nologfile"`
		AllowInjection bool     `json:"allowInjection"`
		LocalOnly      bool     `json:"localOnly"`
		Mock           bool     `json:"mock"`
		Debug          bool     `json:"debug"`
		Port           int      `json:"port"`
		PIDFile        string   `json:"pidfile"`
		LogFile        string   `json:"logfile"`
		LogLevel       string   `json:"loglevel"`
		IPWhitelist    []string `json:"ipWhitelist"`
	} `json:"options"`

	// Process represents information about the mountebank server NodeJS runtime.
	Process struct {
		NodeVersion  string  `json:"nodeVersion"`
		Architecture string  `json:"architecture"`
		Platform     string  `json:"platform"`
		RSS          int64   `json:"rss"`
		HeapTotal    int64   `json:"heapTotal"`
		HeapUsed     int64   `json:"heapUsed"`
		Uptime       float64 `json:"uptime"`
		CWD          string  `json:"cwd"`
	} `json:"process"`
}

// Config retrieves the configuration information of the mountebank
// server pointed to by the client.
//
// See more information on this resource at:
// http://www.mbtest.org/docs/api/overview#get-config.
func (cli *Client) Config(ctx context.Context) (*Config, error) {
	p := "/config"

	req, err := cli.restCli.NewRequest(ctx, http.MethodGet, p, nil, nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &cfg); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return &cfg, nil
}

// Log represents a log entry value in mountebank.
//
// See more information about its full structure at:
// http://www.mbtest.org/docs/api/contracts?type=logs.
type Log struct {
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// Logs retrieves the Log values across all registered Imposters
// between the provided start and end indices, with either index
// filter being excluded if less than zero. Set start < 0 and
// end < 0 to include all Log values.
//
// See more information on this resource at:
// http://www.mbtest.org/docs/api/overview#get-logs.
func (cli *Client) Logs(ctx context.Context, start, end int) ([]Log, error) {
	p := "/logs"
	vs := url.Values{}
	if start >= 0 {
		vs.Add("startIndex", strconv.Itoa(start))
	}
	if end >= 0 {
		vs.Add("endIndex", strconv.Itoa(end))
	}

	req, err := cli.restCli.NewRequest(ctx, http.MethodGet, p, nil, vs)
	if err != nil {
		return nil, err
	}

	resp, err := cli.restCli.Do(req)
	if err != nil {
		return nil, err
	}

	var wrap struct {
		Logs []Log `json:"logs"`
	}
	if resp.StatusCode == http.StatusOK {
		if err := cli.restCli.DecodeResponseBody(resp.Body, &wrap); err != nil {
			return nil, err
		}
	} else {
		return nil, cli.decodeError(resp.Body)
	}
	return wrap.Logs, nil
}
