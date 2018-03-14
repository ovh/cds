package swarm

import (
	"strings"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcherySwarm) killAndRemove(ID string) error {
	container, err := h.dockerClient.ContainerInspect(context.Background(), ID)
	if err != nil {
		//If there is an error, we try to remove the container
		if strings.Contains(err.Error(), "No such container") {
			log.Debug("killAndRemove> cannot InspectContainer: %v", err)
			return nil
		}
		log.Info("killAndRemove> cannot InspectContainer: %v", err)
	}
	if err := h.killAndRemoveContainer(ID); err != nil {
		return sdk.WrapError(err, "killAndRemove> %s", ID[:7])
	}

	//If there is no network settings, stop here
	if container.NetworkSettings == nil {
		return nil
	}

	for _, cnetwork := range container.NetworkSettings.Networks {
		//Get the network
		network, err := h.dockerClient.NetworkInspect(context.Background(), cnetwork.NetworkID)
		if err != nil {
			log.Info("killAndRemove> cannot NetworkInfo: %v", err)
			return sdk.WrapError(err, "killAndRemove> unable to get network for %s", ID[:7])
		}

		// If we succeed to get the network, kill and remove all the container on the network
		if netname, ok := network.Labels["worker_net"]; ok {
			log.Debug("killAndRemove> Remove network %s", netname)
			for id := range network.Containers {
				if err := h.killAndRemoveContainer(id); err != nil {
					log.Error("killAndRemove> unable to kill and remove %d", id[:12])
				}
			}
		}

		//Finally remove the network
		if err := h.dockerClient.NetworkRemove(context.Background(), network.ID); err != nil {
			log.Error("killAndRemove> unable to kill and remove network %d", network.ID[:12])
		}
	}
	return nil
}

func (h *HatcherySwarm) killAndRemoveContainer(ID string) error {
	log.Debug("killAndRemove> remove container %s", ID)
	if err := h.dockerClient.ContainerKill(context.Background(), ID, "SIGKILL"); err != nil {
		if strings.Contains(err.Error(), "is not running") || strings.Contains(err.Error(), "No such container") {
			log.Debug("killAndRemove> container already killed %v", err)
		} else {
			return err
		}
	}

	if err := h.dockerClient.ContainerRemove(context.Background(), ID, types.ContainerRemoveOptions{Force: true}); err != nil {
		// container could be already removed by a previous call to docker
		if !strings.Contains(err.Error(), "No such container") {
			log.Warning("killAndRemove> Unable to remove container %s: %v", ID, err)
			return err
		}
	}

	return nil
}

func (h *HatcherySwarm) killAwolNetworks() error {
	//Checking networks
	nets, errLN := h.dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
	if errLN != nil {
		log.Warning("killAwolWorker> Cannot get networks: %s", errLN)
		return errLN
	}

	for i := range nets {
		n, err := h.dockerClient.NetworkInspect(context.Background(), nets[i].ID)
		if err != nil {
			log.Warning("killAwolWorker> Unable to get network info: %v", err)
			continue
		}

		if n.Driver != "bridge" || n.Name == "docker0" || n.Name == "bridge" {
			continue
		}

		if _, ok := n.Labels["worker_net"]; !ok {
			continue
		}

		if len(n.Containers) > 0 {
			continue
		}

		log.Debug("killAwolWorker> Delete network %s", n.Name)
		if err := h.dockerClient.NetworkRemove(context.Background(), n.ID); err != nil {
			log.Warning("killAwolWorker> Unable to delete network %s err:%s", n.Name, err)
		}
	}
	return nil
}
