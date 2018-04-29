package mbgo_test

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/senseyeio/mbgo"
	"github.com/senseyeio/mbgo/internal/testutil"

	// Docker client dependencies must be vendored in order for
	// their internal package imports to resolve properly.
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

	if !testing.Short() {
		ctx := context.Background()
		cli := mustNewDockerClient()
		image := "andyrbell/mountebank:1.14.0"

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
	}

	// run the main test cases
	code = m.Run()
}

func TestClient_Create(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	mb := newMountebankClient(nil)

	cases := []struct {
		Description string
		Before      func(*mbgo.Client)
		After       func(*mbgo.Client)
		Input       mbgo.Imposter
		Expected    *mbgo.Imposter
		Err         error
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
	if testing.Short() {
		t.Skip()
	}

	mb := newMountebankClient(nil)

	cases := []struct {
		Description string
		Before      func(*mbgo.Client)
		After       func(*mbgo.Client)
		Port        int
		Replay      bool
		Expected    *mbgo.Imposter
		Err         error
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
				_, err := mb.Delete(8081, false)
				testutil.ExpectEqual(t, err, nil)

				imp, err := mb.Create(mbgo.Imposter{
					Port:           8081,
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
				imp, err := mb.Delete(8081, false)
				testutil.ExpectEqual(t, err, nil)
				testutil.ExpectEqual(t, imp.Name, "imposter_test")
			},
			Port:   8081,
			Replay: false,
			Expected: &mbgo.Imposter{
				Port:           8081,
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
	if testing.Short() {
		t.Skip()
	}

	mb := newMountebankClient(nil)

	cases := []struct {
		Description string
		Before      func(*mbgo.Client)
		After       func(*mbgo.Client)
		Port        int
		Replay      bool
		Expected    *mbgo.Imposter
		Err         error
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

// mustNewDockerClient creates a new Docker client instance from
// the the system's environment variables.
//
// Use DOCKER_HOST to set the url to the docker server.
// Use DOCKER_API_VERSION to set the version of the API to reach, leave empty for latest.
// Use DOCKER_CERT_PATH to load the tls certificates from.
// Use DOCKER_TLS_VERIFY to enable or disable TLS verification, off by default.
func mustNewDockerClient() *client.Client {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	return cli
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
			// ports used by imposter fixtures
			"8080/tcp": struct{}{}, // http
			"8081/tcp": struct{}{}, // https
			"8082/tcp": struct{}{}, // tcp
			"8083/tcp": struct{}{}, // smtp
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
			"8081/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "8081"},
			},
			"8082/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "8082"},
			},
			"8083/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "8083"},
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

// newMountebankClient creates a new *mbgo.Client instance given its
// API base URL u; localhost:2525 used in integration tests.
func newMountebankClient(u *url.URL) *mbgo.Client {
	return mbgo.NewClient(&http.Client{
		Timeout: time.Second,
	}, u)
}
