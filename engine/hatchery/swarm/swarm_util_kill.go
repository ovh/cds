package swarm

import (
	"strconv"
	"strings"

	types "github.com/docker/docker/api/types"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

const (
	bridge  = "bridge"
	docker0 = "docker0"
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
	} else {
		// If its a worker "register", check registration before deleting it
		if strings.Contains(container.Name, "register-") {
			modelID, err := strconv.ParseInt(container.Config.Labels["worker_model"], 10, 64)
			if err != nil {
				log.Error("killAndRemove> unable to get model from registering container %s", container.Name)
			} else {
				hatchery.CheckWorkerModelRegister(h, modelID)
			}
		}
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
		network, err := h.dockerClient.NetworkInspect(context.Background(), cnetwork.NetworkID, types.NetworkInspectOptions{Scope: "swarm"})
		if err != nil {
			if !strings.Contains(err.Error(), "No such network") {
				return sdk.WrapError(err, "killAndRemove> unable to get network for %s", ID[:7])
			}
			continue
		}

		//If it's the default docker bridge... skip
		if network.Driver != bridge || network.Name == docker0 || network.Name == bridge {
			continue
		}

		// If we succeed to get the network, kill and remove all the container on the network
		if netname, ok := network.Labels["worker_net"]; ok {
			log.Debug("killAndRemove> Remove network %s", netname)
			for id := range network.Containers {
				if err := h.killAndRemoveContainer(id); err != nil {
					log.Error("killAndRemove> unable to kill and remove container %s err:%s", id[:12], err)
				}
			}
		}

		//Finally remove the network
		if err := h.dockerClient.NetworkRemove(context.Background(), network.ID); err != nil {
			log.Error("killAndRemove> unable to kill and remove network %s err:%s", network.ID[:12], err)
		}
	}
	return nil
}

func (h *HatcherySwarm) killAndRemoveContainer(ID string) error {
	log.Debug("killAndRemove> remove container %s", ID)
	if err := h.dockerClient.ContainerKill(context.Background(), ID, "SIGKILL"); err != nil {
		if !strings.Contains(err.Error(), "is not running") && !strings.Contains(err.Error(), "No such container") {
			return sdk.WrapError(err, "killAndRemove> err on kill container %v", err)
		}
	}

	if err := h.dockerClient.ContainerRemove(context.Background(), ID, types.ContainerRemoveOptions{Force: true}); err != nil {
		// container could be already removed by a previous call to docker
		if !strings.Contains(err.Error(), "No such container") {
			return sdk.WrapError(err, "killAndRemove> Unable to remove container %s", ID)
		}
	}

	return nil
}

func (h *HatcherySwarm) killAwolNetworks() error {
	//Checking networks
	nets, errLN := h.dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
	if errLN != nil {
		log.Warning("killAwolNetworks> Cannot get networks: %s", errLN)
		return errLN
	}

	for i := range nets {
		n, err := h.dockerClient.NetworkInspect(context.Background(), nets[i].ID, types.NetworkInspectOptions{Scope: "swarm"})
		if err != nil {
			log.Warning("killAwolNetworks> Unable to get network info: %v", err)
			continue
		}

		if n.Driver != bridge || n.Name == docker0 || n.Name == bridge {
			continue
		}

		if _, ok := n.Labels["worker_net"]; !ok {
			continue
		}

		if len(n.Containers) > 0 {
			continue
		}

		log.Debug("killAwolNetworks> Delete network %s", n.Name)
		if err := h.dockerClient.NetworkRemove(context.Background(), n.ID); err != nil {
			log.Warning("killAwolNetworks> Unable to delete network %s err:%s", n.Name, err)
		}
	}
	return nil
}
