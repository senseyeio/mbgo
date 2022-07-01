// Copyright (c) 2018 Senseye Ltd. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package mbgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
)

func parseClientSocket(s string) (ip net.IP, err error) {
	parts := strings.Split(s, ":")
	ipStr := strings.Join(parts[0:len(parts)-1], ":")

	ip = net.ParseIP(ipStr)
	if ip == nil {
		err = fmt.Errorf("invalid IP address: %s", ipStr)
	}
	return
}

func toMapValues(q map[string][]string) map[string]interface{} {
	if q == nil {
		return nil
	}

	out := make(map[string]interface{}, len(q))

	for k, ss := range q {
		if len(ss) == 0 {
			continue
		} else if len(ss) == 1 {
			out[k] = ss[0]
		} else {
			out[k] = ss
		}
	}

	return out
}

func fromMapValues(q map[string]interface{}) (map[string][]string, error) {
	if q == nil {
		return nil, nil
	}

	out := make(map[string][]string, len(q))

	for k, v := range q {
		switch typ := v.(type) {
		case string:
			out[k] = []string{typ}
		case []interface{}:
			ss := make([]string, len(typ))
			for i, elem := range typ {
				s, ok := elem.(string)
				if !ok {
					return nil, errors.New("invalid query key array subtype")
				}
				ss[i] = s
			}
			out[k] = ss
		default:
			return nil, fmt.Errorf("invalid query key type: %#v", typ)
		}
	}

	return out, nil
}

type httpRequestDTO struct {
	RequestFrom string                 `json:"requestFrom,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Path        string                 `json:"path,omitempty"`
	Query       map[string]interface{} `json:"query,omitempty"`
	Headers     map[string]interface{} `json:"headers,omitempty"`
	Body        interface{}            `json:"body,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
}

// MarshalJSON satisfies the json.Marshaler interface.
func (r HTTPRequest) MarshalJSON() ([]byte, error) {
	dto := httpRequestDTO{
		RequestFrom: "",
		Method:      r.Method,
		Path:        r.Path,
		Query:       toMapValues(r.Query),
		Headers:     toMapValues(r.Headers),
		Body:        r.Body,
		Timestamp:   r.Timestamp,
	}
	if r.RequestFrom != nil {
		dto.RequestFrom = r.RequestFrom.String()
	}
	return json.Marshal(dto)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (r *HTTPRequest) UnmarshalJSON(b []byte) error {
	var v httpRequestDTO
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	if v.RequestFrom != "" {
		r.RequestFrom, err = parseClientSocket(v.RequestFrom)
		if err != nil {
			return nil
		}
	}
	r.Method = v.Method
	r.Path = v.Path
	r.Query, err = fromMapValues(v.Query)
	if err != nil {
		return nil
	}
	r.Headers, err = fromMapValues(v.Headers)
	if err != nil {
		return nil
	}
	r.Body = v.Body
	r.Timestamp = v.Timestamp

	return nil
}

type httpResponseDTO struct {
	StatusCode int                    `json:"statusCode,omitempty"`
	Headers    map[string]interface{} `json:"headers,omitempty"`
	Body       interface{}            `json:"body,omitempty"`
	Mode       string                 `json:"_mode,omitempty"`
}

// MarshalJSON satisfies the json.Marshaler interface.
func (r HTTPResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(httpResponseDTO{
		StatusCode: r.StatusCode,
		Headers:    toMapValues(r.Headers),
		Body:       r.Body,
		Mode:       r.Mode,
	})
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (r *HTTPResponse) UnmarshalJSON(b []byte) error {
	var v httpResponseDTO
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	r.StatusCode = v.StatusCode
	r.Headers, err = fromMapValues(v.Headers)
	if err != nil {
		return err
	}
	r.Body = v.Body
	r.Mode = v.Mode

	return nil
}

type tcpRequestDTO struct {
	RequestFrom string `json:"requestFrom,omitempty"`
	Data        string `json:"data,omitempty"`
}

// MarshalJSON satisfies the json.Marshaler interface.
func (r TCPRequest) MarshalJSON() ([]byte, error) {
	dto := tcpRequestDTO{
		RequestFrom: "",
		Data:        r.Data,
	}
	if r.RequestFrom != nil {
		dto.RequestFrom = r.RequestFrom.String()
	}
	return json.Marshal(dto)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (r *TCPRequest) UnmarshalJSON(b []byte) error {
	var v tcpRequestDTO
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	if v.RequestFrom != "" {
		r.RequestFrom, err = parseClientSocket(v.RequestFrom)
		if err != nil {
			return err
		}
	}
	r.Data = v.Data

	return err
}

type tcpResponseDTO struct {
	Data string `json:"data"`
}

// MarshalJSON satisfies the json.Marshaler interface.
func (r TCPResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(tcpResponseDTO(r))
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (r *TCPResponse) UnmarshalJSON(b []byte) error {
	var v tcpResponseDTO
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	r.Data = v.Data

	return nil
}

const (
	// Predicate parameter keys for internal use.
	paramCaseSensitive = "caseSensitive"
	paramExcept        = "except"
	paramJSONPath      = "jsonpath"
	paramXPath         = "xpath"
)

type predicateDTO map[string]json.RawMessage

// MarshalJSON satisfies the json.Marshaler interface.
func (p Predicate) MarshalJSON() ([]byte, error) {
	dto := predicateDTO{}

	// marshal request based on type
	switch t := p.Request.(type) {
	case json.Marshaler:
		b, err := t.MarshalJSON()
		if err != nil {
			return nil, err
		}
		dto[p.Operator] = b

	case []Predicate:
		preds := make([]json.RawMessage, len(t))
		for i, sub := range t {
			b, err := sub.MarshalJSON()
			if err != nil {
				return nil, err
			}
			preds[i] = b
		}
		b, err := json.Marshal(preds)
		if err != nil {
			return nil, err
		}
		dto[p.Operator] = b

	case string:
		b, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		dto[p.Operator] = b

	default:
		return nil, fmt.Errorf("unsupported predicate request type: %v",
			reflect.TypeOf(t).String())
	}

	if p.JSONPath != nil {
		b, err := json.Marshal(p.JSONPath)
		if err != nil {
			return nil, err
		}
		dto[paramJSONPath] = b
	}

	if p.CaseSensitive {
		b, err := json.Marshal(p.CaseSensitive)
		if err != nil {
			return nil, err
		}
		dto[paramCaseSensitive] = b
	}

	return json.Marshal(dto)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (p *Predicate) UnmarshalJSON(b []byte) error {
	var dto predicateDTO
	err := json.Unmarshal(b, &dto)
	if err != nil {
		return err
	}

	// Handle and delete parameters from the DTO map before we check the
	// operator so that we can enforce only one operator exists in the map.
	if b, ok := dto[paramCaseSensitive]; ok {
		err = json.Unmarshal(b, &p.CaseSensitive)
		if err != nil {
			return err
		}
		delete(dto, paramCaseSensitive)
	}
	if b, ok := dto[paramJSONPath]; ok {
		err = json.Unmarshal(b, &p.JSONPath)
		if err != nil {
			return err
		}
		delete(dto, paramJSONPath)
	}
	// Ignore 'except' and 'xpath' parameters for now.
	delete(dto, paramExcept)
	delete(dto, paramXPath)

	if len(dto) < 1 {
		return errors.New("predicate should only have a single operator")
	}
	for key, b := range dto {
		p.Operator = key

		switch key {
		// Interpret the request as a string containing JavaScript if the
		// inject operator is used.
		case "inject":
			var js string
			err = json.Unmarshal(b, &js)
			if err != nil {
				return err
			}
			p.Request = js

		// Slice of predicates
		case "and", "or":
			var ps []Predicate
			err = json.Unmarshal(b, &ps)
			if err != nil {
				return err
			}
			p.Request = ps

		// Single predicate
		case "not":
			var v Predicate
			err = json.Unmarshal(b, &v)
			if err != nil {
				return err
			}
			p.Request = v

		// Otherwise we have a request object.
		default:
			p.Request = b // defer unmarshaling until protocol is known
		}
	}

	return nil
}

const (
	keyBehaviors = "_behaviors"
)

// MarshalJSON satisfies the json.Marshaler interface.
func (r Response) MarshalJSON() ([]byte, error) {
	dto := make(map[string]json.RawMessage)

	m, ok := r.Value.(json.Marshaler)
	if !ok {
		return nil, errors.New("response value must implement json.Marshaler")
	}

	b, err := m.MarshalJSON()
	if err != nil {
		return nil, err
	}

	dto[r.Type] = b

	if r.Behaviors != nil {
		behaviors, err := json.Marshal(r.Behaviors)
		if err != nil {
			return nil, err
		}
		dto[keyBehaviors] = behaviors
	}

	return json.Marshal(dto)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (r *Response) UnmarshalJSON(b []byte) error {
	var dto map[string]json.RawMessage
	err := json.Unmarshal(b, &dto)
	if err != nil {
		return err
	}

	// Handle and delete behaviors from the DTO map before we check the
	// type so that we can enforce only one type exists in the map.
	if b, ok := dto[keyBehaviors]; ok {
		err = json.Unmarshal(b, r.Behaviors)
		if err != nil {
			return err
		}
		delete(dto, keyBehaviors)
	}

	for key, b := range dto {
		r.Type = key
		r.Value = b // defer unmarshaling until protocol is known
	}

	return nil
}

type stubDTO struct {
	Predicates []Predicate `json:"predicates,omitempty"`
	Responses  []Response  `json:"responses"`
}

// MarshalJSON satisfies the json.Marshaler interface.
func (s Stub) MarshalJSON() ([]byte, error) {
	return json.Marshal(stubDTO(s))
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (s *Stub) UnmarshalJSON(b []byte) error {
	var dto stubDTO
	err := json.Unmarshal(b, &dto)
	if err != nil {
		return err
	}

	s.Predicates = dto.Predicates
	s.Responses = dto.Responses

	return nil
}

type imposterRequestDTO struct {
	Proto           string            `json:"protocol"`
	Port            int               `json:"port,omitempty"`
	Name            string            `json:"name,omitempty"`
	RecordRequests  bool              `json:"recordRequests,omitempty"`
	AllowCORS       bool              `json:"allowCORS,omitempty"`
	DefaultResponse json.RawMessage   `json:"defaultResponse,omitempty"`
	Stubs           []json.RawMessage `json:"stubs,omitempty"`
}

// MarshalJSON satisfies the json.Marshaler interface.
func (imp Imposter) MarshalJSON() ([]byte, error) {
	dto := imposterRequestDTO{
		Proto:           imp.Proto,
		Port:            imp.Port,
		Name:            imp.Name,
		RecordRequests:  imp.RecordRequests,
		AllowCORS:       imp.AllowCORS,
		DefaultResponse: nil,
		Stubs:           nil,
	}
	if imp.DefaultResponse != nil {
		jm, ok := imp.DefaultResponse.(json.Marshaler)
		if !ok {
			return nil, errors.New("default response must implemented json.Marshaler")
		}
		b, err := jm.MarshalJSON()
		if err != nil {
			return nil, err
		}
		dto.DefaultResponse = b
	}
	if n := len(imp.Stubs); n > 0 {
		dto.Stubs = make([]json.RawMessage, n)
		for i, stub := range imp.Stubs {
			b, err := stub.MarshalJSON()
			if err != nil {
				return nil, err
			}
			dto.Stubs[i] = b
		}
	}
	return json.Marshal(dto)
}

type imposterResponseDTO struct {
	Port         int               `json:"port"`
	Proto        string            `json:"protocol"`
	Name         string            `json:"name,omitempty"`
	RequestCount int               `json:"numberOfRequests,omitempty"`
	Stubs        []json.RawMessage `json:"stubs,omitempty"`
	Requests     []json.RawMessage `json:"requests,omitempty"`
}

func getRequestUnmarshaler(proto string) (json.Unmarshaler, error) {
	var um json.Unmarshaler
	switch proto {
	case "http":
		um = &HTTPRequest{}
	case "tcp":
		um = &TCPRequest{}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}
	return um, nil
}

func unmarshalPredicateRecurse(proto string, p *Predicate) error {
	switch v := p.Request.(type) {
	case json.RawMessage:
		um, err := getRequestUnmarshaler(proto)
		if err != nil {
			return err
		}
		if err = um.UnmarshalJSON(v); err != nil {
			return err
		}
		p.Request = um

	case Predicate:
		if err := unmarshalPredicateRecurse(proto, &v); err != nil {
			return err
		}
	case []Predicate:
		for i := range v {
			if err := unmarshalPredicateRecurse(proto, &v[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func getResponseUnmarshaler(proto string) (json.Unmarshaler, error) {
	var um json.Unmarshaler
	switch proto {
	case "http":
		um = &HTTPResponse{}
	case "tcp":
		um = &TCPResponse{}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}
	return um, nil
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (imp *Imposter) UnmarshalJSON(b []byte) error {
	var dto imposterResponseDTO
	err := json.Unmarshal(b, &dto)
	if err != nil {
		return err
	}

	imp.Port = dto.Port
	imp.Proto = dto.Proto
	imp.Name = dto.Name
	imp.RequestCount = dto.RequestCount

	if n := len(dto.Stubs); n > 0 {
		imp.Stubs = make([]Stub, n)
		for i, b := range dto.Stubs {
			var s Stub
			err = json.Unmarshal(b, &s)
			if err != nil {
				return err
			}

			for i := range s.Predicates {
				err = unmarshalPredicateRecurse(imp.Proto, &s.Predicates[i])
				if err != nil {
					return err
				}
			}

			for i, r := range s.Responses {
				if raw, ok := r.Value.(json.RawMessage); ok {
					um, err := getResponseUnmarshaler(imp.Proto)
					if err != nil {
						return err
					}
					err = um.UnmarshalJSON(raw)
					if err != nil {
						return err
					}
					s.Responses[i].Value = um
				}
			}

			imp.Stubs[i] = s
		}
	}

	if n := len(dto.Requests); n > 0 {
		imp.Requests = make([]interface{}, n)
		for i, b := range dto.Requests {
			um, err := getRequestUnmarshaler(imp.Proto)
			if err != nil {
				return err
			}
			err = um.UnmarshalJSON(b)
			if err != nil {
				return err
			}
			imp.Requests[i] = um
		}
	}

	return nil
}
