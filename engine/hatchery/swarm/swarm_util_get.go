package swarm

import (
	"strings"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
)

func (h *HatcherySwarm) getContainers(dockerClient *dockerClient, options types.ContainerListOptions) ([]types.Container, error) {
	s, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		return nil, sdk.WrapError(err, "hatchery> swarm> getContainers> unable to list containers")
	}
	return s, nil
}

func (h *HatcherySwarm) getContainer(dockerClient *dockerClient, name string, options types.ContainerListOptions) (*types.Container, error) {
	containers, err := h.getContainers(dockerClient, options)
	if err != nil {
		return nil, sdk.WrapError(err, "hatchery> swarm> getContainer> cannot getContainers")
	}

	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(name, "/", "", 1) {
			return &containers[i], nil
		}
	}

	return nil, nil
}
