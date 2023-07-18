package swarm

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func TestHatcherySwarm_KillAwolNetwork(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.Config.Name = "swarmy"

	containers := []types.Container{}
	gock.New("https://lolcat.local").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)
	workers := []sdk.Worker{}
	gock.New("https://cds-api.local").Get("/worker").Reply(200).JSON(workers)

	// AWOL Network
	net := []types.NetworkResource{
		{
			ID:     "net-1",
			Driver: "coucou",
		},
		{
			ID:     "net-2",
			Driver: bridge,
		},
		{
			ID:     "net-3",
			Driver: bridge,
			Labels: map[string]string{
				"worker_net": "net-3",
			},
			Containers: map[string]types.EndpointResource{
				"id-1": {},
			},
		},
		{
			ID:     "net-4",
			Driver: bridge,
			Labels: map[string]string{
				"worker_net": "net-4",
			},
			Containers: map[string]types.EndpointResource{},
			Created:    time.Now(),
		},
		{
			ID:     "net-5",
			Driver: bridge,
			Labels: map[string]string{
				"worker_net": "net-5",
			},
			Containers: map[string]types.EndpointResource{},
			Created:    time.Now().Add(-11 * time.Minute),
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/networks").Reply(http.StatusOK).JSON(net)
	gock.New("https://lolcat.local").Get("/v6.66/networks/net-1").Reply(http.StatusOK).JSON(net[0])
	gock.New("https://lolcat.local").Get("/v6.66/networks/net-2").Reply(http.StatusOK).JSON(net[1])
	gock.New("https://lolcat.local").Get("/v6.66/networks/net-3").Reply(http.StatusOK).JSON(net[2])
	gock.New("https://lolcat.local").Get("/v6.66/networks/net-4").Reply(http.StatusOK).JSON(net[3])
	gock.New("https://lolcat.local").Get("/v6.66/networks/net-5").Reply(http.StatusOK).JSON(net[4])

	// JUJST DELETE NET-5
	gock.New("https://lolcat.local").Delete("/v6.66/networks/net-5").Reply(http.StatusOK).JSON(nil)

	err := h.killAwolWorker(context.TODO())
	require.NoError(t, err)
	require.True(t, gock.IsDone())
}

func TestHatcherySwarm_ListAwolWorker(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.Config.Name = "swarmy"

	now := time.Now()
	d1h := now.Add(-5 * time.Minute)
	containers := []types.Container{
		{
			ID:    "swarmy-model1-w1",
			Names: []string{"swarmy-model1-w1"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy",
				LabelWorkerName: "swarmy-model1-w1",
			},
			Created: d1h.Unix(),
		},
		{
			ID:    "swarmy-model1-w2",
			Names: []string{"swarmy-model1-w2"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy",
				LabelWorkerName: "swarmy-model1-w2",
			},
			Created: d1h.Unix(),
		},
		{
			ID:    "swarmy-model1-w3",
			Names: []string{"swarmy-model1-w3"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy",
				LabelWorkerName: "swarmy-model1-w3",
			},
			Created: time.Now().Unix(),
		},
		{
			ID:    "swarmy2-model1-w4",
			Names: []string{"swarmy2-model1-w4"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy2",
				LabelWorkerName: "swarmy2-model1-w4",
			},
			Created: d1h.Unix(),
		},
		{
			ID:    "swarmy-model1-w4",
			Names: []string{"swarmy-model1-w4"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy",
				LabelWorkerName: "swarmy-model1-w4",
			},
			Created: d1h.Unix(),
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"net-1": {NetworkID: "net-1"},
					"net-2": {NetworkID: "net-2"},
				},
			},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	workers := []sdk.Worker{
		{
			Name:   "swarmy-model1-w1",
			Status: sdk.StatusDisabled,
		},
		{
			Name:   "swarmy-model1-w2",
			Status: sdk.StatusSuccess,
		},
		{
			Name:   "swarmy-model1-w3",
			Status: sdk.StatusDisabled,
		},
	}
	gock.New("https://cds-api.local").Get("/worker").Reply(200).JSON(workers)

	c1 := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			Name: "swarmy-model1-w1",
		},
		Config: &container.Config{
			Labels: map[string]string{
				"worker_model_path": "aa",
			},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/containers/swarmy-model1-w1/json").Reply(http.StatusOK).JSON(c1)
	gock.New("https://lolcat.local").Post("/v6.66/containers/swarmy-model1-w1/kill").Reply(http.StatusOK).JSON(nil)
	gock.New("https://lolcat.local").Delete("/v6.66/containers/swarmy-model1-w1").Reply(http.StatusOK).JSON(nil)

	c4 := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			Name: "swarmy-model1-w4",
		},
		Config: &container.Config{
			Labels: map[string]string{
				"worker_model_path": "aa",
			},
		},
		NetworkSettings: &types.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"net-1": {NetworkID: "net-1"},
				"net-2": {NetworkID: "net-2"},
			},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/containers/swarmy-model1-w4/json").Reply(http.StatusOK).JSON(c4)
	gock.New("https://lolcat.local").Post("/v6.66/containers/swarmy-model1-w4/kill").Reply(http.StatusOK).JSON(nil)
	gock.New("https://lolcat.local").Delete("/v6.66/containers/swarmy-model1-w4").Reply(http.StatusOK).JSON(nil)

	// Network
	ns := []types.NetworkResource{
		{
			ID:     "net-1",
			Driver: "toto",
		},
		{
			ID:     "net-2",
			Driver: bridge,
			Containers: map[string]types.EndpointResource{
				"net-1-ctn-1": {},
				"net-1-ctn-2": {},
			},
			Labels: map[string]string{
				"worker_net": "coucou",
			},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/networks/net-1").Reply(http.StatusOK).JSON(ns[0])
	gock.New("https://lolcat.local").Get("/v6.66/networks/net-2").Reply(http.StatusOK).JSON(ns[1])

	gock.New("https://lolcat.local").Post("/v6.66/containers/net-1-ctn-1/kill").Reply(http.StatusOK).JSON(nil)
	gock.New("https://lolcat.local").Delete("/v6.66/containers/net-1-ctn-1").Reply(http.StatusOK).JSON(nil)
	gock.New("https://lolcat.local").Post("/v6.66/containers/net-1-ctn-2/kill").Reply(http.StatusOK).JSON(nil)
	gock.New("https://lolcat.local").Delete("/v6.66/containers/net-1-ctn-2").Reply(http.StatusOK).JSON(nil)
	gock.New("https://lolcat.local").Delete("/v6.66/networks/net-2").Reply(http.StatusOK).JSON(nil)

	net := []types.NetworkResource{}
	gock.New("https://lolcat.local").Get("/v6.66/networks").Reply(http.StatusOK).JSON(net)

	// Must keep: swarmy-model1-w2, swarmy-model1-w3, swarmy2-model1-w4
	// Must delete: swarmy-model1-w1, swarmy-model1-w4
	// Must delete only network net-2 and containers
	err := h.killAwolWorker(context.TODO())
	require.NoError(t, err)
	require.True(t, gock.IsDone())
}

func TestHatcherySwarm_WorkersStarted(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.Config.Name = "swarmy"
	containers := []types.Container{
		{
			Names: []string{"postgresql"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy",
				LabelWorkerName: "w1",
			},
		},
		{
			Names: []string{"postgresql"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy2",
				LabelWorkerName: "w3",
			},
		},
		{
			Names: []string{"postgresql"},
			Labels: map[string]string{
				LabelHatchery:   "swarmy",
				LabelWorkerName: "w2",
			},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	s, err := h.WorkersStarted(context.TODO())
	require.NoError(t, err)
	require.Equal(t, 2, len(s))
	require.Equal(t, "w1", s[0])
	require.Equal(t, "w2", s[1])
	require.True(t, gock.IsDone())
}

func TestHatcherySwarm_Spawn(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.Config.Name = "swarmy"
	h.dockerClients["default"].MaxContainers = 2

	m := sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
		ModelDocker: sdk.ModelDocker{
			Image: "model:9",
		},
	}

	containers := []types.Container{
		{
			Names: []string{"postgresql"},
			Labels: map[string]string{
				LabelHatchery: "swarmy",
			},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	// SERVICE
	n := types.NetworkCreateResponse{
		ID: "net-666",
	}
	gock.New("https://lolcat.local").Post("/v6.66/networks/create").Reply(http.StatusOK).JSON(n)
	gock.New("https://lolcat.local").Post("/v6.66/images/create").MatchParam("fromImage", "postgresql").MatchParam("tag", "5.6.6").Reply(http.StatusOK).JSON(nil)
	cService := container.ContainerCreateCreatedBody{
		ID: "serviceIdContainer",
	}
	gock.New("https://lolcat.local").Post("/v6.66/containers/create").MatchParam("name", "pg-*").Reply(http.StatusOK).JSON(cService)
	gock.New("https://lolcat.local").Post("/v6.66/containers/serviceIdContainer/start").Reply(http.StatusOK).JSON(nil)

	// WORKER
	gock.New("https://lolcat.local").Post("/v6.66/images/create").MatchParam("fromImage", "model").MatchParam("tag", "9").Reply(http.StatusOK).JSON(nil)
	cWorker := container.ContainerCreateCreatedBody{
		ID: "workerIDContainer",
	}
	gock.New("https://lolcat.local").Post("/v6.66/containers/create").MatchParam("name", "swarmy-*").Reply(http.StatusOK).JSON(cWorker)
	gock.New("https://lolcat.local").Post("/v6.66/containers/workerIDContainer/start").Reply(http.StatusOK).JSON(nil)

	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{
		JobID:      "1",
		Model:      sdk.WorkerStarterWorkerModel{ModelV1: &m},
		WorkerName: "swarmy-worker1",
		Requirements: []sdk.Requirement{
			{
				Name:  "Mem",
				Type:  sdk.MemoryRequirement,
				Value: "4096",
			},
			{
				Name:  "pg",
				Type:  sdk.ServiceRequirement,
				Value: "postgresql:5.6.6",
			},
		},
	})
	assert.NoError(t, err)
	require.True(t, gock.IsDone())
}

func TestHatcherySwarm_SpawnMaxContainerReached(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.Config.Name = "swarmy"
	h.dockerClients["default"].MaxContainers = 1

	m := sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
	}

	containers := []types.Container{
		{
			Names: []string{"postgresql"},
			Labels: map[string]string{
				LabelHatchery: "swarmy",
			},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{
		JobID:      "666",
		Model:      sdk.WorkerStarterWorkerModel{ModelV1: &m},
		WorkerName: "swarmy-workerReached",
	})
	assert.Error(t, err)
	require.Contains(t, "unable to found suitable docker engine", err.Error())
	require.True(t, gock.IsDone())
}

func TestHatcherySwarm_CanSpawn(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.dockerClients["default"].MaxContainers = 1

	m := sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
	}
	jobID := int64(1)

	containers := []types.Container{
		{
			Names:  []string{"postgresql"},
			Labels: map[string]string{},
		},
	}
	gock.New("https://lolcat.local").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	b := h.CanSpawn(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m}, fmt.Sprintf("%d", jobID), []sdk.Requirement{})
	assert.True(t, b)
	assert.True(t, gock.IsDone())
}

func TestHatcherySwarm_MaxContainerReached(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.Config.Name = "swarmy"
	h.dockerClients["default"].MaxContainers = 2
	m := sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
	}
	jobID := int64(1)

	containers := []types.Container{
		{
			Names: []string{"worker1"},
			Labels: map[string]string{
				LabelHatchery: "swarmy",
			},
		},
		{
			Names: []string{"worker2"},
			Labels: map[string]string{
				LabelHatchery: "swarmy",
			},
		},
	}

	gock.New("https://lolcat.local").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)
	b := h.CanSpawn(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m}, fmt.Sprintf("%d", jobID), []sdk.Requirement{})
	assert.False(t, b)
	assert.True(t, gock.IsDone())
}

func TestHatcherySwarm_CanSpawnNoDockerClient(t *testing.T) {
	defer gock.Off()
	h := InitTestHatcherySwarm(t)
	h.dockerClients = nil
	m := sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
	}
	jobID := int64(1)
	b := h.CanSpawn(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m}, fmt.Sprintf("%d", jobID), []sdk.Requirement{})
	assert.False(t, b)
	assert.True(t, gock.IsDone())
}
