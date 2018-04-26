package mbgo_test

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"github.com/senseyeio/mbgo"
)

func TestImposter_MarshalJSON(t *testing.T) {
	cases := []struct {
		Description string
		Imposter    mbgo.Imposter
		Expected    map[string]interface{}
		Err         error
	}{
		{
			Description: "should marshal into the expected JSON structure when stubbing TCP",
			Imposter: mbgo.Imposter{
				Port:           8080,
				Proto:          "tcp",
				Name:           "tcp_imposter",
				RecordRequests: true,
				AllowCORS:      true,
				Stubs: []mbgo.Stub{
					{
						Predicates: []mbgo.Predicate{
							{
								Operator: "equals",
								Request: &mbgo.TCPRequest{
									RequestFrom: &net.TCPAddr{
										IP:   net.IPv4(172, 17, 0, 1),
										Port: 58112,
									},
									Data: "SGVsbG8sIHdvcmxkIQ==",
								},
							},
						},
						Responses: []mbgo.Response{
							{
								Type: "is",
								Value: &mbgo.TCPResponse{
									Data: "Z2l0aHViLmNvbS9zZW5zZXllaW8vbWJnbw==",
								},
							},
						},
					},
				},
			},
			Expected: map[string]interface{}{
				"port":           float64(8080),
				"protocol":       "tcp",
				"name":           "tcp_imposter",
				"recordRequests": true,
				"allowCORS":      true,
				"stubs": []interface{}{
					map[string]interface{}{
						"predicates": []interface{}{
							map[string]interface{}{
								"equals": map[string]interface{}{
									"data":        "SGVsbG8sIHdvcmxkIQ==",
									"requestFrom": "172.17.0.1:58112",
								},
							},
						},
						"responses": []interface{}{
							map[string]interface{}{
								"is": map[string]interface{}{
									"data": "Z2l0aHViLmNvbS9zZW5zZXllaW8vbWJnbw==",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Description, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(c.Imposter)
			expectEqual(t, err, c.Err)

			actual := make(map[string]interface{})
			if err := json.Unmarshal(b, &actual); err != nil {
				t.Fatal(err)
			}
			for key := range actual {
				expectEqual(t, actual[key], c.Expected[key])
			}
		})
	}
}

func TestImposter_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		Description string
		JSON        map[string]interface{}
		Expected    mbgo.Imposter
		Err         error
	}{
		{
			Description: "should unmarshal into the expected structure when given the JSON for an HTTP Imposter",
			JSON: map[string]interface{}{
				"port":             8080,
				"protocol":         "http",
				"name":             "http_imposter",
				"numberOfRequests": 42,
				"stubs": []interface{}{
					map[string]interface{}{
						"predicates": []interface{}{
							map[string]interface{}{
								"equals": map[string]interface{}{
									"requestFrom": "172.17.0.1:58112",
									"method":      "POST",
									"path":        "/foo",
									"query": map[string]string{
										"bar": "baz",
									},
									"headers": map[string]string{
										"Content-Type": "application/json",
									},
									"body": `{"predicate":true}`,
								},
							},
						},
						"responses": []interface{}{
							map[string]interface{}{
								"is": map[string]interface{}{
									"statusCode": 200,
									"_mode":      "text",
									"headers": map[string]string{
										"Accept":       "application/json",
										"Content-Type": "application/json",
									},
									"body": `{"response":true}`,
								},
							},
						},
					},
				},
			},
			Expected: mbgo.Imposter{
				Port:         8080,
				Proto:        "http",
				Name:         "http_imposter",
				RequestCount: 42,
				Stubs: []mbgo.Stub{
					{
						Predicates: []mbgo.Predicate{
							{
								Operator: "equals",
								Request: mbgo.HTTPRequest{
									RequestFrom: &net.TCPAddr{
										IP:   net.IPv4(172, 17, 0, 1),
										Port: 58112,
									},
									Method: "POST",
									Path:   "/foo",
									Query: map[string]string{
										"bar": "baz",
									},
									Headers: map[string]string{
										"Content-Type": "application/json",
									},
									Body: `{"predicate":true}`,
								},
							},
						},
						Responses: []mbgo.Response{
							{
								Type: "is",
								Value: mbgo.HTTPResponse{
									StatusCode: http.StatusOK,
									Mode:       "text",
									Headers: map[string]string{
										"Accept":       "application/json",
										"Content-Type": "application/json",
									},
									Body: `{"response":true}`,
								},
							},
						},
					},
				},
			},
		},
		{
			Description: "should unmarshal into the expected structure when given the JSON for a TCP Imposter",
			JSON: map[string]interface{}{
				"port":             8080,
				"protocol":         "tcp",
				"name":             "tcp_imposter",
				"numberOfRequests": 4,
				"stubs": []interface{}{
					map[string]interface{}{
						"predicates": []interface{}{
							map[string]interface{}{
								"equals": map[string]interface{}{
									"requestFrom": "172.17.0.1:58112",
									"data":        "SGVsbG8sIHdvcmxkIQ==",
								},
							},
						},
						"responses": []interface{}{
							map[string]interface{}{
								"is": map[string]interface{}{
									"data": "Z2l0aHViLmNvbS9zZW5zZXllaW8vbWJnbw==",
								},
							},
						},
					},
				},
			},
			Expected: mbgo.Imposter{
				Port:         8080,
				Proto:        "tcp",
				Name:         "tcp_imposter",
				RequestCount: 4,
				Stubs: []mbgo.Stub{
					{
						Predicates: []mbgo.Predicate{
							{
								Operator: "equals",
								Request: mbgo.TCPRequest{
									RequestFrom: &net.TCPAddr{
										IP:   net.IPv4(172, 17, 0, 1),
										Port: 58112,
									},
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
		c := c

		t.Run(c.Description, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(c.JSON)
			if err != nil {
				t.Fatal(err)
			}

			actual := mbgo.Imposter{}
			err = json.Unmarshal(b, &actual)
			expectEqual(t, err, c.Err)
			expectEqual(t, actual, c.Expected)
		})
	}
}
