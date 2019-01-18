package mbgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
)

const behaviorsKey = "_behaviors"

func parseHybridAddress(s string) (ip net.IP, err error) {
	parts := strings.Split(s, ":")
	ipStr := strings.Join(parts[0:len(parts)-1], ":")

	ip = net.ParseIP(ipStr)
	if ip == nil {
		err = fmt.Errorf("invalid IP address: %s", ipStr)
	}
	return
}

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
	// Note that more than one value per key is not supported.
	Query map[string]string
	// Headers contains the HTTP headers of the request.
	// Note that more than one value per key is not supported.
	Headers map[string]string
	// Body is the body of the request.
	Body interface{}
	// Timestamp is the timestamp of the request.
	Timestamp string
}

// httpRequestDTO is a data transfer object used as an intermediary value
// for marshalling and un-marshalling the JSON structure of an HTTPRequest.
type httpRequestDTO struct {
	RequestFrom string            `json:"requestFrom,omitempty"`
	Method      string            `json:"method,omitempty"`
	Path        string            `json:"path,omitempty"`
	Query       map[string]string `json:"query,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        interface{}       `json:"body,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
}

// toDTO maps an HTTPRequest value to a httpRequestDTO value.
func (r HTTPRequest) toDTO() httpRequestDTO {
	dto := httpRequestDTO{}
	if r.RequestFrom != nil {
		dto.RequestFrom = r.RequestFrom.String()
	}
	dto.Method = r.Method
	dto.Path = r.Path
	dto.Query = r.Query
	dto.Headers = r.Headers
	dto.Body = r.Body
	dto.Timestamp = r.Timestamp
	return dto
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

// tcpRequestDTO is a data transfer object used as an intermediary value
// for marshalling and un-marshalling the JSON structure of a TCPRequest.
type tcpRequestDTO struct {
	RequestFrom string `json:"requestFrom,omitempty"`
	Data        string `json:"data,omitempty"`
}

// toDTO maps a TCPRequest value to a tcpRequestDTO value.
func (r TCPRequest) toDTO() tcpRequestDTO {
	dto := tcpRequestDTO{}
	if r.RequestFrom != nil {
		dto.RequestFrom = r.RequestFrom.String()
	}
	dto.Data = r.Data
	return dto
}

// unmarshalRequest unmarshals a network request given its protocol proto and the JSON data b.
func unmarshalRequest(proto string, b json.RawMessage) (v interface{}, err error) {
	switch proto {
	case "http":
		var dto httpRequestDTO
		if err = json.Unmarshal(b, &dto); err != nil {
			return
		}
		var ip net.IP
		if dto.RequestFrom != "" {
			ip, err = parseHybridAddress(dto.RequestFrom)
			if err != nil {
				return
			}
		}
		v = HTTPRequest{
			RequestFrom: ip,
			Method:      dto.Method,
			Path:        dto.Path,
			Query:       dto.Query,
			Headers:     dto.Headers,
			Body:        dto.Body,
			Timestamp:   dto.Timestamp,
		}
		return

	case "tcp":
		var dto tcpRequestDTO
		if err = json.Unmarshal(b, &dto); err != nil {
			return
		}
		var ip net.IP
		if dto.RequestFrom != "" {
			ip, err = parseHybridAddress(dto.RequestFrom)
			if err != nil {
				return
			}
		}
		v = TCPRequest{
			RequestFrom: ip,
			Data:        dto.Data,
		}
		return

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}
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

// toDTO maps a Predicate value to a predicateDTO value.
func (p Predicate) toDTO() (predicateDTO, error) {
	dto := predicateDTO{}

	var v interface{}
	switch typ := p.Request.(type) {
	case HTTPRequest:
		v = typ.toDTO()
	case *HTTPRequest:
		v = typ.toDTO()
	case TCPRequest:
		v = typ.toDTO()
	case *TCPRequest:
		v = typ.toDTO()
	}

	b, err := json.Marshal(v)
	if err != nil {
		return dto, err
	}
	dto[p.Operator] = b

	if p.JSONPath != nil {
		b, err = json.Marshal(p.JSONPath)
		if err != nil {
			return dto, err
		}
		dto["jsonpath"] = b
	}

	if p.CaseSensitive {
		dto["caseSensitive"] = []byte("true")
	}

	return dto, nil
}

// predicateDTO is an data-transfer object used as an intermediary value
// for delaying the marshalling and un-marshalling of its inner request
// value until the protocol is known at runtime. See the unmarshalProto
// method for more details.
type predicateDTO map[string]json.RawMessage

// unmarshalProto unmarshals the predicateDTO dto into a Predicate value
// with its Request field set to the appropriate type based on the specified
// network protocol proto - currently either HTTPRequest or TCPRequest.
func (dto predicateDTO) unmarshalProto(proto string) (p Predicate, err error) {
	if len(dto) < 1 {
		err = errors.New("unexpected Predicate JSON structure")
		return
	}

	for key, b := range dto {
		p.Operator = key
		p.Request, err = unmarshalRequest(proto, b)
	}
	return
}

// HTTPResponse is a Response.Value used to respond to a matched HTTPRequest.
//
// See more information about HTTP responses in mountebank at:
// http://www.mbtest.org/docs/protocols/http.
type HTTPResponse struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Headers are the HTTP headers in the response.
	Headers map[string]string
	// Body is the body of the response. It will be JSON encoded before sending to mountebank
	Body interface{}
	// Mode is the mode of the response; either "text" or "binary".
	// Defaults to "text" if excluded.
	Mode string
}

// httpResponseDTO is the data-transfer object used to describe the
// JSON structure of an HTTPResponse value.
type httpResponseDTO struct {
	StatusCode int               `json:"statusCode,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       interface{}       `json:"body,omitempty"`
	Mode       string            `json:"_mode,omitempty"`
}

// toDTO maps a HTTPResponse value to an httpResponseDTO value.
func (r HTTPResponse) toDTO() httpResponseDTO {
	return httpResponseDTO{
		StatusCode: r.StatusCode,
		Headers:    r.Headers,
		Body:       r.Body,
		Mode:       r.Mode,
	}
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

// tcpResponseDTO is the data-transfer object used to describe the
// JSON structure of an TCPResponse value.
type tcpResponseDTO struct {
	Data string `json:"data"`
}

// toDTO maps a TCPResponse value to a tcpResponseDTO value.
func (r TCPResponse) toDTO() tcpResponseDTO {
	return tcpResponseDTO{
		Data: r.Data,
	}
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

// Behaviors defines the possible response behaviors for a stub.
// Currently supported values are:
// wait - Adds latency to a response by waiting a specified number of milliseconds before sending the response.
// See more information on stub response behaviors in mountebank at:
// http://www.mbtest.org/docs/api/behaviors.
type Behaviors struct {
	Wait int `json:"wait,omitempty"`
}

func getResponseSubTypeDTO(v interface{}) (interface{}, error) {
	switch typ := v.(type) {
	case HTTPResponse:
		v = typ.toDTO()
	case *HTTPResponse:
		v = typ.toDTO()
	case TCPResponse:
		v = typ.toDTO()
	case *TCPResponse:
		v = typ.toDTO()
	default:
		return nil, errors.New("invalid response type")
	}
	return v, nil
}

// toDTO maps a Response value to a responseDTO value; used for json.Marshal.
func (r Response) toDTO() (responseDTO, error) {
	dto := responseDTO{}

	v, err := getResponseSubTypeDTO(r.Value)
	if err != nil {
		return dto, err
	}

	b, err := json.Marshal(v)
	if err != nil {
		return dto, err
	}

	dto[r.Type] = b

	if r.Behaviors != nil {
		behaviors, err := json.Marshal(r.Behaviors)
		if err != nil {
			return dto, err
		}
		dto[behaviorsKey] = behaviors
	}

	return dto, nil
}

// responseDTO is an data-transfer object used as an intermediary value
// for delaying the marshalling and un-marshalling of its inner response
// value until the protocol is known at runtime. See the unmarshalProto
// method for more details.
type responseDTO map[string]json.RawMessage

// unmarshalProto un-marshals the responseDTO dto into a Response value
// with its Value field set to the appropriate type based on the specified
// network protocol proto - currently either type HTTPResponse or TCPResponse.
func (dto responseDTO) unmarshalProto(proto string) (resp Response, err error) {
	if len(dto) != 1 {
		if len(dto) == 2 {
			if _, ok := dto[behaviorsKey]; !ok {
				err = errors.New("unexpected Predicate JSON structure")
				return
			}
		}
	}

	for key, b := range dto {
		resp.Type = key

		switch proto {
		case "http":
			r := httpResponseDTO{}
			if err = json.Unmarshal(b, &r); err != nil {
				return
			}
			resp.Value = HTTPResponse{
				StatusCode: r.StatusCode,
				Headers:    r.Headers,
				Body:       r.Body,
				Mode:       r.Mode,
			}

		case "tcp":
			r := tcpResponseDTO{}
			if err = json.Unmarshal(b, &r); err != nil {
				return
			}
			resp.Value = TCPResponse{
				Data: r.Data,
			}
		}
	}
	return
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

// toDTO maps a Stub value to a stubDTO value; used for json.Marshal.
func (s Stub) toDTO() (stubDTO, error) {
	dto := stubDTO{
		Predicates: make([]predicateDTO, len(s.Predicates)),
		Responses:  make([]responseDTO, len(s.Responses)),
	}
	for i, p := range s.Predicates {
		v, err := p.toDTO()
		if err != nil {
			return dto, err
		}
		dto.Predicates[i] = v
	}
	for i, r := range s.Responses {
		v, err := r.toDTO()
		if err != nil {
			return dto, err
		}
		dto.Responses[i] = v
	}
	return dto, nil
}

// stubDTO is an data-transfer object used as an intermediary value
// for delaying the marshalling and un-marshalling the JSON structure
// of a Stub value.
type stubDTO struct {
	Predicates []predicateDTO `json:"predicates,omitempty"`
	Responses  []responseDTO  `json:"responses"`
}

// unmarshalProto un-marshals the stubDTO dto into a Stub value with its
// inner Predicate.Request fields set to the appropriate type based on the
// specified network protocol proto.
func (dto stubDTO) unmarshalProto(proto string) (s Stub, err error) {
	// build up Predicate.Request values based on the protocol.
	s.Predicates = make([]Predicate, 0, len(dto.Predicates))
	for _, v := range dto.Predicates {
		var p Predicate
		p, err = v.unmarshalProto(proto)
		if err != nil {
			return
		}
		s.Predicates = append(s.Predicates, p)
	}

	// likewise, build up the Response.Value values based on the protocol
	s.Responses = make([]Response, 0, len(dto.Responses))
	for _, v := range dto.Responses {
		var r Response
		r, err = v.unmarshalProto(proto)
		if err != nil {
			return
		}
		s.Responses = append(s.Responses, r)
	}
	return
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

// MarshalJSON implements the json.Marshaler interface for Imposter,
// used to map an Imposter value to its JSON structure for creation.
//
// See details about the full creation structure of an Imposter at:
// http://www.mbtest.org/docs/api/contracts?type=imposter
func (imp Imposter) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	// required fields
	m["port"] = imp.Port
	m["protocol"] = imp.Proto

	// optional fields
	if imp.Name != "" {
		m["name"] = imp.Name
	}
	if imp.RecordRequests {
		m["recordRequests"] = imp.RecordRequests
	}
	if len(imp.Stubs) > 0 {
		stubs := make([]stubDTO, len(imp.Stubs))
		for i, stub := range imp.Stubs {
			v, err := stub.toDTO()
			if err != nil {
				return nil, err
			}
			stubs[i] = v
		}
		m["stubs"] = stubs
	}
	if imp.AllowCORS {
		m["allowCORS"] = imp.AllowCORS
	}
	if imp.DefaultResponse != nil {
		v, err := getResponseSubTypeDTO(imp.DefaultResponse)
		if err != nil {
			return nil, err
		}
		m["defaultResponse"] = v
	}
	return json.Marshal(&m)
}

// imposterResponseDTO is an data-transfer object used as an intermediary
// value for an Imposter value from a mountebank API response. Note that
// this JSON structure differs from the structure provided during creation.
//
// See details about the full response structure of an Imposter at:
// http://www.mbtest.org/docs/api/contracts?type=imposter
type imposterResponseDTO struct {
	Port         int               `json:"port"`
	Proto        string            `json:"protocol"`
	Name         string            `json:"name,omitempty"`
	RequestCount int               `json:"numberOfRequests"`
	Stubs        []stubDTO         `json:"stubs,omitempty"`
	Requests     []json.RawMessage `json:"requests,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface for Imposter.
//
// The un-marshalling of any nested Predicate.Request or Response values
// within the Stubs will be determined at runtime based on the protocol
// used by the Imposter. For instance, all Predicate.Request values would
// be of type HTTPRequest and all Response.Value values of type HTTPResponse
// if Proto == "http".
//
// See details about the full structure of an Imposter at:
// http://www.mbtest.org/docs/api/contracts?type=imposter
func (imp *Imposter) UnmarshalJSON(b []byte) error {
	dto := imposterResponseDTO{}
	if err := json.Unmarshal(b, &dto); err != nil {
		return err
	}
	imp.Port = dto.Port
	imp.Proto = dto.Proto
	imp.Name = dto.Name
	imp.RequestCount = dto.RequestCount
	if len(dto.Stubs) > 0 {
		imp.Stubs = make([]Stub, 0, len(dto.Stubs))
		for _, v := range dto.Stubs {
			stub, err := v.unmarshalProto(imp.Proto)
			if err != nil {
				return err
			}
			imp.Stubs = append(imp.Stubs, stub)
		}
	}
	if len(dto.Requests) > 0 {
		imp.Requests = make([]interface{}, 0, len(dto.Requests))

		for _, b := range dto.Requests {
			req, err := unmarshalRequest(imp.Proto, b)
			if err != nil {
				return err
			}
			imp.Requests = append(imp.Requests, req)
		}
	}
	return nil
}
