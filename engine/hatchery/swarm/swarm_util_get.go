package swarm

import (
	"strings"
	"time"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
)

func (h *HatcherySwarm) getContainers(dockerClient *dockerClient, options types.ContainerListOptions) ([]types.Container, error) {
	ctxList, cancelList := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelList()
	s, err := dockerClient.ContainerList(ctxList, options)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list containers on %s", dockerClient.name)
	}
	return s, nil
}

func (h *HatcherySwarm) getContainer(dockerClient *dockerClient, name string, options types.ContainerListOptions) (*types.Container, error) {
	containers, err := h.getContainers(dockerClient, options)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot getContainers on %s", dockerClient.name)
	}

	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(name, "/", "", 1) {
			return &containers[i], nil
		}
	}

	return nil, nil
}
