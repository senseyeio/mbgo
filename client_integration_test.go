// +build integration

package mbgo_test

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/senseyeio/mbgo"
	"github.com/senseyeio/mbgo/internal/testutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

func TestMain(m *testing.M) {
	// must parse flags to get -short flag; not parsed before TestMain by default
	flag.Parse()

	var code int
	defer func() {
		if err := recover(); err != nil {
			log.Printf("caught test panic: %v", err)
			code = 1
		}
		os.Exit(code)
	}()

	ctx := context.Background()
	cli := mustNewDockerClient()
	image := "andyrbell/mountebank:1.14.1"

	var (
		id  string
		err error
	)

	// setup a mountebank Docker container for integration tests
	if err = pullDockerImage(ctx, cli, image, time.Second*45); err != nil {
		panic(err)
	}
	id, err = createDockerContainer(ctx, cli, image, time.Second*5)
	if err != nil {
		panic(err)
	}
	if err = startDockerContainer(ctx, cli, id, time.Second*3); err != nil {
		panic(err)
	}
	if err = waitHealthyDockerContainer(ctx, cli, id, time.Second*10); err != nil {
		panic(err)
	}

	// Always stop/remove the test container, even on test failure or panic.
	defer func() {
		if err = stopDockerContainer(ctx, cli, id, time.Second*3); err != nil {
			panic(err)
		}
		if err = removeDockerContainer(ctx, cli, id, time.Second*3); err != nil {
			panic(err)
		}
	}()

	// run the main test cases
	code = m.Run()
}

func TestClient_Logs(t *testing.T) {
	mb := newMountebankClient(nil)

	vs, err := mb.Logs(-1, -1)
	testutil.ExpectEqual(t, err, nil)
	testutil.ExpectEqual(t, len(vs) >= 2, true)
	testutil.ExpectEqual(t, vs[0].Message, "[mb:2525] mountebank v1.14.1 now taking orders - point your browser to http://localhost:2525 for help")
	testutil.ExpectEqual(t, vs[1].Message, "[mb:2525] GET /logs")
}

func TestClient_Create(t *testing.T) {
	mb := newMountebankClient(nil)

	cases := []struct {
		// general
		Description string
		Before      func(*mbgo.Client)
		After       func(*mbgo.Client)

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
									Query: map[string]string{
										"page": "3",
									},
									Headers: map[string]string{
										"Accept": "application/json",
									},
								},
							},
						},
						Responses: []mbgo.Response{
							{
								Type: "is",
								Value: mbgo.HTTPResponse{
									StatusCode: http.StatusOK,
									Headers: map[string]string{
										"Content-Type": "application/json",
									},
									Body: `{"test":true}`,
								},
							},
						},
					},
				},
			},
			Before: func(mb *mbgo.Client) {
				_, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)
			},
			After: func(mb *mbgo.Client) {
				imp, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)
				testutil.ExpectEqual(t, imp.Name, "create_test")
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
									Query: map[string]string{
										"page": "3",
									},
									Headers: map[string]string{
										"Accept": "application/json",
									},
								},
							},
						},
						Responses: []mbgo.Response{
							{
								Type: "is",
								Value: mbgo.HTTPResponse{
									StatusCode: http.StatusOK,
									Headers: map[string]string{
										"Content-Type": "application/json",
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
				c.Before(mb)
			}

			actual, err := mb.Create(c.Input)
			testutil.ExpectEqual(t, err, c.Err)
			testutil.ExpectEqual(t, actual, c.Expected)

			if c.After != nil {
				c.After(mb)
			}
		})
	}
}

func TestClient_Imposter(t *testing.T) {
	mb := newMountebankClient(nil)

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
			Description: "should error if an Imposter does not exist on the specified port",
			Port:        8080,
			Before: func(mb *mbgo.Client) {
				_, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)
			},
			Err: errors.New("no such resource: Try POSTing to /imposters first?"),
		},
		{
			Description: "should return the expected TCP Imposter if it exists on the specified port",
			Before: func(mb *mbgo.Client) {
				_, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)

				imp, err := mb.Create(mbgo.Imposter{
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
				testutil.ExpectEqual(t, err, nil)
				testutil.ExpectEqual(t, imp.Name, "imposter_test")
			},
			After: func(mb *mbgo.Client) {
				imp, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)
				testutil.ExpectEqual(t, imp.Name, "imposter_test")
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
				c.Before(mb)
			}

			actual, err := mb.Imposter(c.Port, c.Replay)
			testutil.ExpectEqual(t, err, c.Err)
			testutil.ExpectEqual(t, actual, c.Expected)

			if c.After != nil {
				c.After(mb)
			}
		})
	}
}

func TestClient_Delete(t *testing.T) {
	mb := newMountebankClient(nil)

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
				_, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)
			},
			Expected: &mbgo.Imposter{},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(mb)
			}

			actual, err := mb.Delete(c.Port, c.Replay)
			testutil.ExpectEqual(t, err, c.Err)
			testutil.ExpectEqual(t, actual, c.Expected)

			if c.After != nil {
				c.After(mb)
			}
		})
	}
}

func TestClient_DeleteRequests(t *testing.T) {
	mb := newMountebankClient(nil)

	cases := []struct {
		// general
		Description string
		Before      func(*mbgo.Client)
		After       func(*mbgo.Client)

		// input
		Port int

		// output expectations
		Expected *mbgo.Imposter
		Err      error
	}{
		{
			Description: "should return an empty Imposter struct if one is not configured on the specified port",
			Before: func(mb *mbgo.Client) {
				_, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)
			},
			Port:     8080,
			Expected: &mbgo.Imposter{},
		},
		{
			Description: "should return the expected Imposter if it exists on successful deletion",
			Before: func(mb *mbgo.Client) {
				_, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)

				_, err = mb.Create(mbgo.Imposter{
					Port:           8080,
					Proto:          "http",
					Name:           "delete_requests_test",
					RecordRequests: true,
				})
				testutil.ExpectEqual(t, err, nil)

				// make some HTTP requests to the new Imposter
				for i := 0; i < 1; i++ {
					resp, err := http.Get("http://localhost:8080/foo?bar=true")
					testutil.ExpectEqual(t, err, nil)
					testutil.ExpectEqual(t, resp.StatusCode, http.StatusOK)
				}
			},
			After: func(mb *mbgo.Client) {
				imp, err := mb.Delete(8080, false)
				testutil.ExpectEqual(t, err, nil)
				testutil.ExpectEqual(t, imp.Name, "delete_requests_test")
			},
			Port: 8080,
			Expected: &mbgo.Imposter{
				Port:         8080,
				Proto:        "http",
				Name:         "delete_requests_test",
				RequestCount: 0,
				Requests: []interface{}{
					mbgo.HTTPRequest{
						RequestFrom: net.IPv4(172, 17, 0, 1),
						Method:      http.MethodGet,
						Path:        "/foo",
						Headers: map[string]string{
							"Host":            "localhost:8080",
							"User-Agent":      "Go-http-client/1.1",
							"Accept-Encoding": "gzip",
						},
						Query: map[string]string{
							"bar": "true",
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(mb)
			}

			actual, err := mb.DeleteRequests(c.Port)
			testutil.ExpectEqual(t, err, c.Err)
			testutil.ExpectEqual(t, actual, c.Expected)

			if c.After != nil {
				c.After(mb)
			}
		})
	}
}

func TestClient_Config(t *testing.T) {
	mb := newMountebankClient(nil)

	cfg, err := mb.Config()
	testutil.ExpectEqual(t, err, nil)
	testutil.ExpectEqual(t, cfg.Version, "1.14.1")
}

func TestClient_Imposters(t *testing.T) {
	cases := []struct {
		// general
		Description string
		Before      func(*mbgo.Client)
		After       func(*mbgo.Client)

		// input
		Replay bool

		// output expectations
		Expected []mbgo.Imposter
		Err      error
	}{
		{
			Description: "should return a minimal representation of all registered Imposters",
			Before: func(mb *mbgo.Client) {
				_, err := mb.DeleteAll(false)
				testutil.ExpectEqual(t, err, nil)

				// create a tcp imposter
				imp, err := mb.Create(mbgo.Imposter{
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
				testutil.ExpectEqual(t, err, nil)
				testutil.ExpectEqual(t, imp.Name, "imposters_tcp_test")

				// and an http imposter
				imp, err = mb.Create(mbgo.Imposter{
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
										Query: map[string]string{
											"page": "3",
										},
										Headers: map[string]string{
											"Accept": "application/json",
										},
									},
								},
							},
							Responses: []mbgo.Response{
								{
									Type: "is",
									Value: mbgo.HTTPResponse{
										StatusCode: http.StatusOK,
										Headers: map[string]string{
											"Content-Type": "application/json",
										},
										Body: `{"test":true}`,
									},
								},
							},
						},
					},
				})
				testutil.ExpectEqual(t, err, nil)
				testutil.ExpectEqual(t, imp.Name, "imposters_http_test")
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

	mb := newMountebankClient(nil)

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(mb)
			}

			actual, err := mb.Imposters(c.Replay)
			testutil.ExpectEqual(t, err, c.Err)
			testutil.ExpectEqual(t, actual, c.Expected)

			if c.After != nil {
				c.After(mb)
			}
		})
	}
}

func mustNewDockerClient() *client.Client {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	return cli
}

func newMountebankClient(u *url.URL) *mbgo.Client {
	return mbgo.NewClient(&http.Client{
		Timeout: time.Second,
	}, u)
}

func pullDockerImage(ctx context.Context, cli *client.Client, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := cli.ImagePull(ctx, name, types.ImagePullOptions{})
	return err
}

func createDockerContainer(ctx context.Context, cli *client.Client, image string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   []string{"mb", "--mock", "--debug"},
		ExposedPorts: nat.PortSet{
			"2525/tcp": struct{}{},
			"8080/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "nc", "-z", "localhost", "2525"},
			Interval: time.Millisecond * 100,
			Retries:  50,
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"2525/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "2525"},
			},
			"8080/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "8080"},
			},
		},
	}, nil, "mbgo_integration_test")
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func startDockerContainer(ctx context.Context, cli *client.Client, id string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
}

func waitHealthyDockerContainer(ctx context.Context, cli *client.Client, id string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		dto, err := cli.ContainerInspect(ctx, id)
		if err != nil {
			return err
		}

		// block until the container is healthy
		if !dto.State.Running || dto.State.Health.Status != "healthy" {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		break
	}
	return nil
}

func stopDockerContainer(ctx context.Context, cli *client.Client, id string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return cli.ContainerStop(ctx, id, &timeout)
}

func removeDockerContainer(ctx context.Context, cli *client.Client, id string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{})
}
