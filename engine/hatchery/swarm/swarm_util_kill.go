package swarm

import (
	"strconv"
	"strings"
	"time"

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

func (h *HatcherySwarm) killAndRemove(dockerClient *dockerClient, ID string) error {
	container, err := dockerClient.ContainerInspect(context.Background(), ID)
	if err != nil {
		//If there is an error, we try to remove the container
		if strings.Contains(err.Error(), "No such container") {
			log.Debug("hatchery> swarm> killAndRemove> cannot InspectContainer: %v", err)
			return nil
		}
		log.Info("hatchery> swarm> killAndRemove> cannot InspectContainer: %v", err)
	} else {
		// If its a worker "register", check registration before deleting it
		if strings.Contains(container.Name, "register-") {
			modelID, err := strconv.ParseInt(container.Config.Labels["worker_model"], 10, 64)
			if err != nil {
				log.Error("hatchery> swarm> killAndRemove> unable to get model from registering container %s", container.Name)
			} else {
				hatchery.CheckWorkerModelRegister(h, modelID)
			}
		}
	}

	if err := h.killAndRemoveContainer(dockerClient, ID); err != nil {
		return sdk.WrapError(err, "hatchery> swarm> killAndRemove> %s on %s", ID[:7], dockerClient.name)
	}

	//If there is no network settings, stop here
	if container.NetworkSettings == nil {
		return nil
	}

	for _, cnetwork := range container.NetworkSettings.Networks {
		//Get the network
		network, err := dockerClient.NetworkInspect(context.Background(), cnetwork.NetworkID, types.NetworkInspectOptions{})
		if err != nil {
			if !strings.Contains(err.Error(), "No such network") {
				return sdk.WrapError(err, "hatchery> swarm> killAndRemove> unable to get network for % on %s", ID[:7], dockerClient.name)
			}
			continue
		}

		//If it's the default docker bridge... skip
		if network.Driver != bridge || network.Name == docker0 || network.Name == bridge {
			continue
		}

		// If we succeed to get the network, kill and remove all the container on the network
		if netname, ok := network.Labels["worker_net"]; ok {
			log.Debug("hatchery> swarm> killAndRemove> Remove network %s", netname)
			for id := range network.Containers {
				if err := h.killAndRemoveContainer(dockerClient, id); err != nil {
					log.Error("hatchery> swarm> killAndRemove> unable to kill and remove container %s on %s err:%s", id[:12], dockerClient.name, err)
				}
			}
		}

		//Finally remove the network
		if err := dockerClient.NetworkRemove(context.Background(), network.ID); err != nil {
			log.Error("hatchery> swarm> killAndRemove> unable to kill and remove network %s from %s err:%s", network.ID[:12], dockerClient.name, err)
		}
	}
	return nil
}

func (h *HatcherySwarm) killAndRemoveContainer(dockerClient *dockerClient, ID string) error {
	log.Debug("hatchery> swarm> killAndRemove> remove container %s on %s", ID, dockerClient.name)
	if err := dockerClient.ContainerKill(context.Background(), ID, "SIGKILL"); err != nil {
		if !strings.Contains(err.Error(), "is not running") && !strings.Contains(err.Error(), "No such container") {
			return sdk.WrapError(err, "hatchery> swarm> killAndRemove> err on kill container %v from %s", err, dockerClient.name)
		}
	}

	if err := dockerClient.ContainerRemove(context.Background(), ID, types.ContainerRemoveOptions{Force: true}); err != nil {
		// container could be already removed by a previous call to docker
		if !strings.Contains(err.Error(), "No such container") {
			return sdk.WrapError(err, "hatchery> swarm> killAndRemove> Unable to remove container %s form %s", ID, dockerClient.name)
		}
	}

	return nil
}

func (h *HatcherySwarm) killAwolNetworks() error {
	for _, dockerClient := range h.dockerClients {
		//Checking networks
		nets, errLN := dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
		if errLN != nil {
			log.Warning("hatchery> swarm> killAwolNetworks> Cannot get networks on %s: %s", dockerClient.name, errLN)
			return errLN
		}

		for i := range nets {
			n, err := dockerClient.NetworkInspect(context.Background(), nets[i].ID, types.NetworkInspectOptions{})
			if err != nil {
				log.Warning("hatchery> swarm> killAwolNetworks> Unable to get network info: %v", err)
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

			// if network created less than 10 min, keep it alive for now
			if time.Since(nets[i].Created) < 10*time.Minute {
				continue
			}

			log.Debug("hatchery> swarm> killAwolNetworks> Delete network %s from %s", n.Name, dockerClient.name)
			if err := dockerClient.NetworkRemove(context.Background(), n.ID); err != nil {
				log.Warning("hatchery> swarm> killAwolNetworks> Unable to delete network %s err:%s", n.Name, err)
			}
		}
	}
	return nil
}
