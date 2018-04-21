package mbgo_test

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/senseyeio/mbgo"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/senseyeio/mbgo/internal/rest"
)

// containerURL points to the local mountebank Docker container used in the
// integration tests. Only available when not testing in short mode.
var containerURL *url.URL

func TestMain(m *testing.M) {
	// must parse flags to get -short flag; not parsed before TestMain by default
	flag.Parse()

	var (
		code int
		cli  *client.Client
		id   string
	)

	if !testing.Short() {
		cli = mustNewDockerClient()
		image := "andyrbell/mountebank:1.14.0"

		// create/start a test container, then wait for it to be healthy
		id, containerURL = mustStartDockerContainer(cli, image)
	}

	// run the main test cases
	code = m.Run()

	if !testing.Short() {
		// Always stop/remove the test container, even on failure.
		// This function cannot be deferred since program will Exit
		// before it resolves.
		mustStopDockerContainer(cli, id, time.Second)
	}

	os.Exit(code)
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

// mustStartDockerContainer pulls the mountebank image, creates and
// starts the container, then waits for it to be healthy.
//
// The container exposes port 2525 for the mountebank API and the port
// range 8080-8084 to be used by imposter fixtures of the http, https,
// tcp and smtp protocols, respectively.
func mustStartDockerContainer(cli *client.Client, image string) (string, *url.URL) {
	ctx := context.Background()
	_, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

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
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	for {
		dto, err := cli.ContainerInspect(ctx, resp.ID)
		if err != nil {
			panic(err)
		}
		// block until the container is healthy
		if !dto.State.Running || dto.State.Health.Status != "healthy" {
			continue
		}
		time.Sleep(time.Millisecond * 100)
		break
	}

	return resp.ID, &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("0.0.0.0", "2525"),
	}
}

// mustStopDockerContainer stops and removes the Docker container
// specified by the given id.
func mustStopDockerContainer(cli *client.Client, id string, timeout time.Duration) {
	ctx := context.Background()

	if err := cli.ContainerStop(ctx, id, &timeout); err != nil {
		panic(err)
	}

	if err := cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{}); err != nil {
		panic(err)
	}
}

// newMountebankClient creates a new *mbgo.Client instance given its
// API base URL u; localhost:2525 used in integration tests.
func newMountebankClient(u *url.URL) *mbgo.Client {
	return mbgo.NewClient(&http.Client{
		Timeout: time.Second,
	}, u)
}

// expectEqual is a helper function used throughout the unit and integration
// tests to assert deep quality between an actual and expected value.
func expectEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected %v to equal %v", actual, expected)
	}
}

func TestPredicate_MarshalJSON(t *testing.T) {
	cases := []struct {
		Description string
		Predicate   mbgo.Predicate
		Expected    string
		Err         error
	}{
		{
			Description: "OperatorEquals",
			Predicate: mbgo.Predicate{
				Operator: "equals",
				Request: mbgo.Request{
					Method: http.MethodGet,
				},
			},
			Expected: `{"equals":{"method":"GET"}}`,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Description, func(t *testing.T) {
			t.Parallel()

			actual, err := json.Marshal(c.Predicate)
			expectEqual(t, err, c.Err)
			expectEqual(t, string(actual), c.Expected)
		})
	}
}

func TestClient_Create(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	mb := newMountebankClient(containerURL)

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
			Err: rest.Error{
				Code:    "bad data",
				Message: "invalid value for 'port'",
			},
		},
		{
			Description: "should error if an invalid protocol is provided",
			Input: mbgo.Imposter{
				Proto: "udp",
				Port:  8080,
			},
			Err: rest.Error{
				Code:    "bad data",
				Message: "the udp protocol is not yet supported",
			},
		},
		{
			Description: "should create the Imposter if the provided data is valid",
			Input: mbgo.Imposter{
				Proto: "http",
				Port:  8080,
				Name:  "create_test",
			},
			Before: func(mb *mbgo.Client) {
				_, err := mb.Delete(8080, false)
				expectEqual(t, err, nil)
			},
			After: func(mb *mbgo.Client) {
				imp, err := mb.Delete(8080, false)
				expectEqual(t, err, nil)
				expectEqual(t, imp.Name, "create_test")
			},
			Expected: &mbgo.Imposter{
				Proto: "http",
				Port:  8080,
				Name:  "create_test",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(mb)
			}

			actual, err := mb.Create(c.Input)
			expectEqual(t, err, c.Err)
			expectEqual(t, actual, c.Expected)

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

	mb := newMountebankClient(containerURL)

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
				expectEqual(t, err, nil)
			},
			Err: rest.Error{
				Code:    "no such resource",
				Message: "Try POSTing to /imposters first?",
			},
		},
		{
			Description: "should return the expected Imposter if it exists on the specified port",
			Before: func(mb *mbgo.Client) {
				imp, err := mb.Create(mbgo.Imposter{
					Port:  8080,
					Proto: "http",
					Name:  "imposter_test",
				})
				expectEqual(t, err, nil)
				expectEqual(t, imp.Name, "imposter_test")
			},
			After: func(mb *mbgo.Client) {
				imp, err := mb.Delete(8080, false)
				expectEqual(t, err, nil)
				expectEqual(t, imp.Name, "imposter_test")
			},
			Port:   8080,
			Replay: false,
			Expected: &mbgo.Imposter{
				Port:  8080,
				Proto: "http",
				Name:  "imposter_test",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			if c.Before != nil {
				c.Before(mb)
			}

			actual, err := mb.Imposter(c.Port, c.Replay)
			expectEqual(t, err, c.Err)
			expectEqual(t, actual, c.Expected)

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

	mb := newMountebankClient(containerURL)

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
				expectEqual(t, err, nil)
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
			expectEqual(t, err, c.Err)
			expectEqual(t, actual, c.Expected)

			if c.After != nil {
				c.After(mb)
			}
		})
	}
}
