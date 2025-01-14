package swarm

import (
	"strings"
	"testing"

	types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

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
