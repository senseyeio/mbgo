package mbgo_test

import (
	"context"
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

func TestMain(m *testing.M) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	// create/start a test container, then wait for it to be healthy
	id := mustStartDockerContainer(cli, "andyrbell/mountebank")

	// run the test cases
	code := m.Run()

	// always stop the test container, even on failure
	mustStopDockerContainer(cli, id, time.Second)

	os.Exit(code)
}

func TestClient_Create(t *testing.T) {
	cases := []struct {
		Description string
		Input       mbgo.Imposter
		Expected    *mbgo.Imposter
		Err         error
	}{
		{
			Description: "bad data: invalid port",
			Err: rest.Error{
				Code:    "bad data",
				Message: "invalid value for 'port'",
			},
		},
		{
			Description: "bad data: unsupported protocol",
			Input: mbgo.Imposter{
				Port: 8080,
			},
			Err: rest.Error{
				Code:    "bad data",
				Message: "the  protocol is not yet supported",
			},
		},
		{
			Description: "success",
			Input: mbgo.Imposter{
				Port:  8080,
				Proto: "http",
			},
			Expected: &mbgo.Imposter{
				Port:  8080,
				Proto: "http",
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Description, func(t *testing.T) {
			t.Parallel()

			mb := mbgo.NewClient(&http.Client{}, &url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort("0.0.0.0", "2525"),
			})

			actual, err := mb.Create(c.Input)
			if !reflect.DeepEqual(err, c.Err) {
				t.Errorf(`expected "%v" to equal "%v"`, err, c.Err)
			}
			if !reflect.DeepEqual(actual, c.Expected) {
				t.Errorf("expected %v to equal %v", actual, c.Expected)
			}
		})
	}
}

func TestClient_Config(t *testing.T) {
	mb := mbgo.NewClient(&http.Client{}, &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("0.0.0.0", "2525"),
	})

	cfg, err := mb.Config()
	if err != nil {
		t.Fatal(err)
	}
	if expected := "1.14.0"; cfg.Version != expected {
		t.Errorf("expected %v to equal %v", cfg.Version, expected)
	}
}

// mustStartDockerContainer pulls the mountebank image, creates and
// starts the container, then waits for it to be healthy.
func mustStartDockerContainer(cli *client.Client, image string) string {
	ctx := context.Background()
	_, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   []string{"mb"},
		ExposedPorts: nat.PortSet{
			"2525/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "nc", "-z", "localhost", "2525"},
			Interval: time.Second,
			Retries:  10,
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"2525/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "2525",
				},
			},
		},
	}, nil, "mountebank_integration_test")
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
		// wait for the container to be healthy
		if !dto.State.Running || dto.State.Health.Status != "healthy" {
			continue
		}
		time.Sleep(time.Millisecond * 100)
		break
	}

	return resp.ID
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
