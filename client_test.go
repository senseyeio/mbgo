package mbgo_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/senseyeio/mbgo"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// containerURL points to the local mountebank Docker container used in the integration examples.
var containerURL *url.URL

func TestMain(m *testing.M) {
	// must parse flags to get -short flag; not parsed before TestMain by default
	flag.Parse()

	// skip all Docker integration examples in short mode
	if testing.Short() {
		log.Printf("skipping integration tests")
		return
	}

	cli := mustNewDockerClient()
	image := "andyrbell/mountebank:1.14.0"

	// create/start a test container, then wait for it to be healthy
	id, u := mustStartDockerContainer(cli, image)
	containerURL = u

	var code int
	defer func() {
		if err := recover(); err != nil {
			log.Printf("test panic caught: %v", err)
			code = 1
		}

		// always stop/remove the test container, even on failure
		mustStopDockerContainer(cli, id, time.Second)
		os.Exit(code)
	}()

	code = m.Run()
}

func ExampleClient_Create() {
	mb := newMountebankClient(containerURL)

	imp, err := mb.Create(mbgo.Imposter{
		Port:  8080,
		Proto: "http",
		Name:  "example_create",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d, %s, %s, %d", imp.Port, imp.Proto, imp.Name, imp.RequestCount)
	// Output: 8080, http, example_create, 0
}

func ExampleClient_Imposter() {
	mb := newMountebankClient(containerURL)
	port := 8081

	// create an imposter fixture prior to retrieval
	_, err := mb.Create(mbgo.Imposter{
		Port:  port,
		Proto: "https",
		Name:  "example_imposter",
	})
	if err != nil {
		panic(err)
	}

	imp, err := mb.Imposter(port, false)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d, %s, %s", imp.Port, imp.Proto, imp.Name)
	// Output: 8081, https, example_imposter
}

func ExampleClient_Delete() {
	mb := newMountebankClient(containerURL)
	port := 8082

	// create an imposter fixture prior to deletion
	_, err := mb.Create(mbgo.Imposter{
		Port:  port,
		Proto: "tcp",
		Name:  "example_delete",
	})
	if err != nil {
		panic(err)
	}

	imp, err := mb.Delete(port, false)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d, %s, %s", imp.Port, imp.Proto, imp.Name)
	// Output: 8082, tcp, example_delete
}

func ExampleClient_DeleteRequests() {
	mb := newMountebankClient(containerURL)
	port := 8080

	// delete any previous imposters on the ports under test
	if _, err := mb.DeleteAll(false); err != nil {
		panic(err)
	}

	// create an imposter fixture to record HTTP requests
	_, err := mb.Create(mbgo.Imposter{
		Port:           port,
		Proto:          "http",
		Name:           "example_delete_requests",
		RecordRequests: true,
	})
	if err != nil {
		panic(err)
	}

	// make a couple of HTTP requests on the imposter port
	u := *containerURL
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		panic(err)
	}
	u.Host = net.JoinHostPort(host, fmt.Sprintf("%d", port))
	if _, err = http.Get(u.String()); err != nil {
		panic(err)
	}
	if _, err = http.Get(u.String()); err != nil {
		panic(err)
	}

	// verify the requests were recorded, then deleted
	before, err := mb.Imposter(port, false)
	if err != nil {
		panic(err)
	}
	after, err := mb.DeleteRequests(port)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d, %d", before.RequestCount, after.RequestCount)
	// Output: 2, 0
}

func ExampleClient_Overwrite() {
	mb := newMountebankClient(containerURL)

	// delete any previous imposters on the ports under test
	if _, err := mb.DeleteAll(false); err != nil {
		panic(err)
	}

	// create a few imposter fixtures to overwrite later
	before1, err := mb.Create(mbgo.Imposter{
		Port:  8080,
		Proto: "http",
		Name:  "example_overwrite_before_1",
	})
	if err != nil {
		panic(err)
	}
	before2, err := mb.Create(mbgo.Imposter{
		Port:  8081,
		Proto: "https",
		Name:  "example_overwrite_before_2",
	})
	if err != nil {
		panic(err)
	}

	after, err := mb.Overwrite([]mbgo.Imposter{
		{
			Port:  8080,
			Proto: "tcp",
			Name:  "example_overwrite_after_1",
		},
		{
			Port:  8081,
			Proto: "smtp",
			Name:  "example_overwrite_after_2",
		},
	}, false)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s, %s, %s, %s",
		before1.Proto, before2.Proto, after[0].Proto, after[1].Proto)
	// Output: http, https, tcp, smtp
}

func ExampleClient_Imposters() {
	mb := newMountebankClient(containerURL)

	// delete any previous imposters on the ports under test
	if _, err := mb.DeleteAll(false); err != nil {
		panic(err)
	}

	// create a few imposter fixtures to list later
	_, err := mb.Create(mbgo.Imposter{
		Port:  8080,
		Proto: "http",
		Name:  "example_imposters_1",
	})
	if err != nil {
		panic(err)
	}
	_, err = mb.Create(mbgo.Imposter{
		Port:  8081,
		Proto: "https",
		Name:  "example_imposters_2",
	})
	if err != nil {
		panic(err)
	}

	imps, err := mb.Imposters(true)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d, %s, %s", len(imps), imps[0].Name, imps[1].Name)
	// Output: 2, example_imposters_1, example_imposters_2
}

func ExampleClient_DeleteAll() {
	mb := newMountebankClient(containerURL)

	// delete any previous imposters on the ports under test
	if _, err := mb.DeleteAll(false); err != nil {
		panic(err)
	}

	// create a few imposter fixtures to delete
	if _, err := mb.Create(mbgo.Imposter{
		Port:  8080,
		Proto: "http",
		Name:  "example_delete_all_1",
	}); err != nil {
		panic(err)
	}
	if _, err := mb.Create(mbgo.Imposter{
		Port:  8081,
		Proto: "https",
		Name:  "example_delete_all_2",
	}); err != nil {
		panic(err)
	}

	before, err := mb.Imposters(false)
	if err != nil {
		panic(err)
	}
	deleted, err := mb.DeleteAll(false)
	if err != nil {
		panic(err)
	}
	after, err := mb.Imposters(false)
	if err != nil {
		panic(err)
	}

	fmt.Printf("before: %d, after: %d, deleted: %d", len(before), len(after), len(deleted))
	// Output: before: 2, after: 0, deleted: 2
}

func ExampleClient_Config() {
	mb := newMountebankClient(containerURL)

	cfg, err := mb.Config()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s, %s", cfg.Version, cfg.Process.NodeVersion)
	// Output: 1.14.0, v8.9.3
}

func ExampleClient_Logs() {
	mb := newMountebankClient(containerURL)

	logs, err := mb.Logs(-1, -1)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d", len(logs))
	// Output: 52
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
		// wait for the container to be healthy
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
// API base URL u; localhost:2525 used in integration examples.
func newMountebankClient(u *url.URL) *mbgo.Client {
	return mbgo.NewClient(&http.Client{
		Timeout: time.Second,
	}, u)
}
