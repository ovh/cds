package swarm

import (
	"github.com/docker/docker/api/types"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
	"net/http"
	"testing"
)

func TestHatcherySwarm_CanSpawn(t *testing.T) {
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
	gock.New("https://lolcat.host").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	b := h.CanSpawn(&m, jobID, []sdk.Requirement{})
	assert.True(t, b)
}

func TestHatcherySwarm_MaxContainerReached(t *testing.T) {
	h := InitTestHatcherySwarm(t)
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
				"hatchery": "swarmy",
			},
		},
		{
			Names: []string{"worker2"},
			Labels: map[string]string{
				"hatchery": "swarmy",
			},
		},
	}

	gock.New("https://lolcat.host").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)
	b := h.CanSpawn(&m, jobID, []sdk.Requirement{})
	assert.False(t, b)
}

func TestHatcherySwarm_MaxContainerRatioService100(t *testing.T) {
	h := InitTestHatcherySwarm(t)
	h.dockerClients["default"].MaxContainers = 2

	ratio := 100
	h.Config.Provision.RatioService = &ratio
	m := sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
	}
	jobID := int64(1)

	containers := []types.Container{}

	gock.New("https://lolcat.host").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)

	b := h.CanSpawn(&m, jobID, []sdk.Requirement{})
	assert.False(t, b)
}

func TestHatcherySwarm_MaxContainerRatioPercentReached(t *testing.T) {
	h := InitTestHatcherySwarm(t)
	h.Config.Name = "swarmy"
	h.dockerClients["default"].MaxContainers = 5

	ratio := 20
	h.Config.Provision.RatioService = &ratio
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
				"hatchery":    "swarmy",
				"worker_name": "w1",
			},
		},
		{
			Names: []string{"worker2"},
			Labels: map[string]string{
				"hatchery":    "swarmy",
				"worker_name": "w2",
			},
		},
		{
			Names: []string{"worker3"},
			Labels: map[string]string{
				"hatchery":    "swarmy",
				"worker_name": "w3",
			},
		},
		{
			Names: []string{"worker4"},
			Labels: map[string]string{
				"hatchery":    "swarmy",
				"worker_name": "w4",
			},
		},
	}

	gock.New("https://lolcat.host").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)
	b := h.CanSpawn(&m, jobID, []sdk.Requirement{})
	assert.False(t, b)
}

func TestHatcherySwarm_MaxContainerRatioPercentOK(t *testing.T) {
	h := InitTestHatcherySwarm(t)
	h.dockerClients["default"].MaxContainers = 3
	h.Config.Provision.MaxWorker = 3

	ratio := 90
	h.Config.Provision.RatioService = &ratio
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
				"hatchery": "swarmy",
			},
		},
		{
			Names: []string{"worker2"},
			Labels: map[string]string{
				"hatchery": "swarmy",
			},
		},
	}

	gock.New("https://lolcat.host").Get("/v6.66/containers/json").Reply(http.StatusOK).JSON(containers)
	b := h.CanSpawn(&m, jobID, []sdk.Requirement{{Name: "pg", Type: sdk.ServiceRequirement, Value: "postgresql"}})
	assert.True(t, b)
}

func TestHatcherySwarm_CanSpawnNoDockerClient(t *testing.T) {
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
	b := h.CanSpawn(&m, jobID, []sdk.Requirement{})
	assert.False(t, b)
}
