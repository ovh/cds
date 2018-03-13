package swarm

import (
	"strings"
	"sync"
	"time"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//This a embeded cache for containers list
var containersCache = struct {
	mu   sync.RWMutex
	list []types.Container
}{
	mu:   sync.RWMutex{},
	list: []types.Container{},
}

func (h *HatcherySwarm) getContainers() ([]types.Container, error) {
	containersCache.mu.RLock()
	nbServers := len(containersCache.list)
	containersCache.mu.RUnlock()

	if nbServers == 0 {
		s, err := h.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
			All: true,
		})
		if err != nil {
			return nil, sdk.WrapError(err, "getContainers> unable to list containers")
		}
		containersCache.mu.Lock()
		containersCache.list = s
		containersCache.mu.Unlock()

		for _, v := range s {
			log.Debug("getContainers> container ID:%s names:%+v image:%s created:%d state:%s, status:%s", v.ID, v.Names, v.Image, v.Created, v.State, v.Status)
		}
		//Remove data from the cache after 2 seconds
		go func() {
			time.Sleep(2 * time.Second)
			containersCache.mu.Lock()
			containersCache.list = []types.Container{}
			containersCache.mu.Unlock()
		}()
	}

	return containersCache.list, nil
}

func (h *HatcherySwarm) getContainer(name string) (*types.Container, error) {
	containers, err := h.getContainers()
	if err != nil {
		return nil, sdk.WrapError(err, "getContainer> cannot getContainers")
	}

	for i := range containers {
		if strings.Replace(containers[i].Names[0], "/", "", 1) == strings.Replace(name, "/", "", 1) {
			return &containers[i], nil
		}
	}

	return nil, nil
}
