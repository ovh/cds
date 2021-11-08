package swarm

import (
	"time"

	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
)

type Containers []types.Container

func (c Containers) GetByID(id string) *types.Container {
	for i := range c {
		if c[i].ID == id {
			return &c[i]
		}
	}
	return nil
}

func (c Containers) FilterWorkers() Containers {
	res := make(Containers, 0, len(c))
	for i := range c {
		if _, ok := c[i].Labels[LabelWorkerName]; ok {
			res = append(res, c[i])
		}
	}
	return res
}

func (h *HatcherySwarm) getContainers(ctx context.Context, dockerClient *dockerClient, options types.ContainerListOptions) (Containers, error) {
	ctxList, cancelList := context.WithTimeout(ctx, 10*time.Second)
	defer cancelList()

	cs, err := dockerClient.ContainerList(ctxList, options)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list containers on %s", dockerClient.name)
	}

	// Filter hatchery's containers
	res := make(Containers, 0, len(cs))
	for _, c := range res {
		if hatcheryName, ok := c.Labels[LabelHatchery]; ok || hatcheryName == h.Config.Name {
			res = append(res, c)
		}
	}

	return res, nil
}
