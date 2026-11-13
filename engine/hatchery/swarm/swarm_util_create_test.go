package swarm

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func TestDockerClientUserError(t *testing.T) {
	c, err := docker.NewClientWithOpts(docker.WithHost("tcp://docker-engine.example.com:2376"))
	require.NoError(t, err)
	engine := &dockerClient{Client: *c, name: "engine-1"}

	urlErr := &url.Error{
		Op:  "Post",
		URL: "https://docker-engine.example.com:2376/v1.44/images/create?fromImage=registry.example.com%2Fimage&tag=latest",
		Err: errors.New("net/http: timeout awaiting response headers"),
	}
	require.Equal(t, "the docker engine did not answer", engine.userError(sdk.WithStack(urlErr)))

	// Errors from the docker daemon are kept, as they can be actionable for users, without the engine address
	daemonErr := fmt.Errorf("Error response from daemon: pull access denied for registry.example.com/image (engine docker-engine.example.com:2376)")
	require.Equal(t, "Error response from daemon: pull access denied for registry.example.com/image (engine engine-1)", engine.userError(daemonErr))

	require.Equal(t, "manifest unknown", engine.userError(sdk.WithStack(errors.New("manifest unknown"))))

	// A real failure of the docker client to reach an engine
	unreachable, err := docker.NewClientWithOpts(docker.WithHost("tcp://127.0.0.1:1"))
	require.NoError(t, err)
	_, err = unreachable.Ping(context.Background())
	require.Error(t, err)
	require.Equal(t, "the docker engine did not answer", (&dockerClient{Client: *unreachable, name: "engine-2"}).userError(sdk.WithStack(err)))
}

func TestHatcherySwarm_createAndStartContainer(t *testing.T) {
	h := testSwarmHatchery(t)
	args := containerArgs{
		name:       "my-nginx",
		image:      "nginx:latest",
		env:        []string{"FROM_CDS", "FROM_CDS"},
		labels:     map[string]string{"FROM_CDS": "FROM_CDS"},
		memory:     256,
		memorySwap: -1,
	}

	// RegisterOnly = true, this will pull image if image is not found
	spawnArgs := hatchery.SpawnArguments{
		RegisterOnly: true,
		Model:        sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{}},
	}
	err := h.createAndStartContainer(context.TODO(), h.dockerClients["default"], args, spawnArgs)
	require.NoError(t, err)

	containers, err := h.getContainers(context.TODO(), h.dockerClients["default"], container.ListOptions{})
	require.NoError(t, err)

	cntr, err := getContainer(h.dockerClients["default"], containers, args.name, container.ListOptions{})
	require.NoError(t, err)

	err = h.killAndRemove(context.TODO(), h.dockerClients["default"], cntr.ID, containers)
	require.NoError(t, err)
}

func TestHatcherySwarm_createAndStartContainerWithNetwork(t *testing.T) {
	h := testSwarmHatchery(t)
	args := containerArgs{
		name:         "my-nginx",
		image:        "nginx:latest",
		cmd:          []string{"uname"},
		env:          []string{"FROM_CDS", "FROM_CDS"},
		labels:       map[string]string{"FROM_CDS": "FROM_CDS"},
		memory:       256,
		memorySwap:   -1,
		network:      "my-network",
		networkAlias: "my-container",
	}

	err := h.createNetwork(context.TODO(), h.dockerClients["default"], args.network)
	require.NoError(t, err)

	spawnArgs := hatchery.SpawnArguments{
		RegisterOnly: false,
		Model:        sdk.WorkerStarterWorkerModel{ModelV1: &sdk.Model{}},
	}
	err = h.createAndStartContainer(context.TODO(), h.dockerClients["default"], args, spawnArgs)
	require.NoError(t, err)

	containers, err := h.getContainers(context.TODO(), h.dockerClients["default"], container.ListOptions{})
	require.NoError(t, err)

	cntr, err := getContainer(h.dockerClients["default"], containers, args.name, container.ListOptions{})
	require.NoError(t, err)

	err = h.killAndRemove(context.TODO(), h.dockerClients["default"], cntr.ID, containers)
	require.NoError(t, err)
}

func getContainer(dockerClient *dockerClient, containers []types.Container, name string, options container.ListOptions) (*types.Container, error) {
	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(name, "/", "", 1) {
			return &containers[i], nil
		}
	}

	return nil, nil
}
