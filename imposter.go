// Copyright (c) 2018 Senseye Ltd. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package mbgo

import (
	"net"
	"net/http"
	"net/url"
)

// HTTPRequest describes an incoming HTTP request received by an
// Imposter of the "http" protocol.
//
// See more information about HTTP requests in mountebank at:
// http://www.mbtest.org/docs/protocols/http.
type HTTPRequest struct {
	// RequestFrom is the originating address of the incoming request.
	RequestFrom net.IP

	// Method is the HTTP request method.
	Method string

	// Path is the path of the request, without the query parameters.
	Path string

	// Query contains the URL query parameters of the request.
	Query url.Values

	// Headers contains the HTTP headers of the request.
	Headers http.Header

	// Body is the body of the request.
	Body interface{}

	// Timestamp is the timestamp of the request.
	Timestamp string
}

// TCPRequest describes incoming TCP data received by an Imposter of
// the "tcp" protocol.
//
// See more information about TCP requests in mountebank at:
// http://www.mbtest.org/docs/protocols/tcp.
type TCPRequest struct {
	// RequestFrom is the originating address of the incoming request.
	RequestFrom net.IP

	// Data is the data in the request as plaintext.
	Data string
}

// JSONPath is a predicate parameter used to narrow the scope of a tested value
// to one found at the specified path in the response JSON.
//
// See more information about the JSONPath parameter at:
// http://www.mbtest.org/docs/api/jsonpath.
type JSONPath struct {
	// Selector is the JSON path of the value tested against the predicate.
	Selector string `json:"selector"`
}

// Predicate represents conditional behaviour attached to a Stub in order
// for it to match or not match an incoming request.
//
// The supported operations for a Predicate are listed at:
// http://www.mbtest.org/docs/api/predicates.
type Predicate struct {
	// Operator is the conditional or logical operator of the Predicate.
	Operator string

	// Request is the request value challenged against the Operator;
	// either of type HTTPRequest or TCPRequest.
	Request interface{}

	// JSONPath is the predicate parameter for narrowing the scope of JSON
	// comparison; leave nil to disable functionality.
	JSONPath *JSONPath

	// CaseSensitive determines if the match is case sensitive or not.
	CaseSensitive bool
}

// HTTPResponse is a Response.Value used to respond to a matched HTTPRequest.
//
// See more information about HTTP responses in mountebank at:
// http://www.mbtest.org/docs/protocols/http.
type HTTPResponse struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// Headers are the HTTP headers in the response.
	Headers http.Header

	// Body is the body of the response. It will be JSON encoded before sending to mountebank
	Body interface{}

	// Mode is the mode of the response; either "text" or "binary".
	// Defaults to "text" if excluded.
	Mode string
}

// TCPResponse is a Response.Value to a matched incoming TCPRequest.
//
// See more information about TCP responses in mountebank at:
// http://www.mbtest.org/docs/protocols/tcp.
type TCPResponse struct {
	// Data is the data in the data contained in the response.
	// An empty string does not respond with data, but does send
	// the FIN bit.
	Data string
}

// Behaviors defines the possible response behaviors for a stub.
//
// See more information on stub behaviours in mountebank at:
// http://www.mbtest.org/docs/api/behaviors.
type Behaviors struct {
	// Wait adds latency to a response by waiting a specified number of milliseconds before sending the response.
	Wait int `json:"wait,omitempty"`
}

// Response defines a networked response sent by a Stub whenever an
// incoming Request matches one of its Predicates. Each Response is
// has a Type field that defines its behaviour. Its currently supported
// values are:
//	is - Merges the specified Response fields with the defaults.
//	proxy - Proxies the request to the specified destination and returns the response.
//	inject - Creates the Response object based on the injected Javascript.
//
// See more information on stub responses in mountebank at:
// http://www.mbtest.org/docs/api/stubs.
type Response struct {
	// Type is the type of the Response; one of "is", "proxy" or "inject".
	Type string

	// Value is the value of the Response; either of type HTTPResponse or TCPResponse.
	Value interface{}

	// Behaviors is an optional field allowing the user to define response behavior.
	Behaviors *Behaviors
}

// Stub adds behaviour to Imposters where one or more registered Responses
// will be returned if an incoming request matches all of the registered
// Predicates. Any Stub value without Predicates always matches and returns
// its next Response. Note that the Responses slice acts as a circular-queue
// type structure, where every time the Stub matches an incoming request, the
// first Response is moved to the end of the slice. This allows for test cases
// to define and handle a sequence of Responses.
//
// See more information about stubs in mountebank at:
// http://www.mbtest.org/docs/api/stubs.
type Stub struct {
	// Predicates are the list of Predicates associated with the Stub,
	// which are logically AND'd together if more than one exists.
	Predicates []Predicate

	// Responses are the circular queue of Responses used to respond to
	// incoming matched requests.
	Responses []Response
}

// Imposter is the primary mountebank resource, representing a server/service
// that listens for networked traffic of a specified protocol and port, with the
// ability to match incoming requests and respond to them based on the behaviour
// of any attached Stub values.
//
// See one of the following links below for details on Imposter creation
// parameters, which varies by protocol:
//
// http://www.mbtest.org/docs/protocols/http
//
// http://www.mbtest.org/docs/protocols/tcp
type Imposter struct {
	// Port is the listening port of the Imposter; required.
	Port int

	// Proto is the listening protocol of the Imposter; required.
	Proto string

	// Name is the name of the Imposter.
	Name string

	// RecordRequests adds mock verification support to the Imposter
	// by having it remember any requests made to it, which can later
	// be retrieved and examined by the testing environment.
	RecordRequests bool

	// Requests are the list of recorded requests, or nil if RecordRequests == false.
	// Note that the underlying type will be HTTPRequest or TCPRequest depending on
	// the protocol of the Imposter.
	Requests []interface{}

	// RequestCount is the number of matched requests received by the Imposter.
	// Note that this value is only used/set when receiving Imposter data
	// from the mountebank server.
	RequestCount int

	// AllowCORS will allow all CORS pre-flight requests on the Imposter.
	AllowCORS bool

	// DefaultResponse is the default response to send if no predicate matches.
	// Only used by HTTP and TCP Imposters; should be one of HTTPResponse or TCPResponse.
	DefaultResponse interface{}

	// Stubs contains zero or more valid Stubs associated with the Imposter.
	Stubs []Stub
}
