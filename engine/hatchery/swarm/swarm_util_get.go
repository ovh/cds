package swarm

import (
	"time"

	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"

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

func (h *HatcherySwarm) getContainer(ctns []types.Container, id string) *types.Container {
	for i := range ctns {
		ctn := &ctns[i]
		if ctn.ID == id {
			return ctn
		}
	}
	return nil
}
