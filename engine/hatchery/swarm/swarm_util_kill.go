package swarm

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	types "github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	context "golang.org/x/net/context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

const (
	bridge  = "bridge"
	docker0 = "docker0"
)

func (h *HatcherySwarm) killAndRemove(ctx context.Context, dockerClient *dockerClient, ID string, containers []types.Container) error {
	ctxList, cancelList := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelList()
	container, err := dockerClient.ContainerInspect(ctxList, ID)
	if err != nil {
		//If there is an error, we try to remove the container
		if strings.Contains(err.Error(), "No such container") {
			log.Debug("hatchery> swarm> killAndRemove> cannot InspectContainer: %v on %s", err, dockerClient.name)
			return nil
		}
		log.Info(ctx, "hatchery> swarm> killAndRemove> cannot InspectContainer: %v on %s", err, dockerClient.name)
	} else {
		// If its a worker "register", check registration before deleting it
		if strings.HasPrefix(container.Name, "register-") {
			modelPath := container.Config.Labels["worker_model_path"]

			if err := hatchery.CheckWorkerModelRegister(h, modelPath); err != nil {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
				defer cancel()
				logsOpts := types.ContainerLogsOptions{
					Details:    true,
					ShowStderr: true,
					ShowStdout: true,
					Timestamps: true,
					Since:      "10s",
				}
				var spawnErr = sdk.SpawnErrorForm{
					Error: err.Error(),
				}

				logsReader, errL := dockerClient.ContainerLogs(ctx, container.ID, logsOpts)
				if errL != nil {
					log.Error(ctx, "hatchery> swarm> killAndRemove> cannot get logs from docker for containers service %s %v : %v", container.ID, container.Name, errL)
					spawnErr.Logs = []byte(fmt.Sprintf("unable to get container logs: %v", errL))

				} else if logsReader != nil {
					defer logsReader.Close()
					logs, errR := ioutil.ReadAll(logsReader)
					if errR != nil {
						log.Error(ctx, "hatchery> swarm> killAndRemove> cannot read logs for containers service %s %v : %v", container.ID, container.Name, errR)
					} else if logs != nil {
						spawnErr.Logs = logs
					}
				}

				tuple := strings.SplitN(modelPath, "/", 2)
				if err := h.CDSClient().WorkerModelSpawnError(tuple[0], tuple[1], spawnErr); err != nil {
					log.Error(ctx, "hatchery> swarm> killAndRemove> error on call client.WorkerModelSpawnError on worker model %s for register: %s", modelPath, err)
				}
			}
		}

	}

	if err := h.killAndRemoveContainer(ctx, dockerClient, ID); err != nil {
		return sdk.WrapError(err, "%s on %s", sdk.StringFirstN(ID, 7), dockerClient.name)
	}

	//If there is no network settings, stop here
	if container.NetworkSettings == nil {
		return nil
	}

	for _, cnetwork := range container.NetworkSettings.Networks {
		//Get the network
		ctxList, cancelList := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelList()
		network, err := dockerClient.NetworkInspect(ctxList, cnetwork.NetworkID, types.NetworkInspectOptions{})
		if err != nil {
			if !strings.Contains(err.Error(), "No such network") {
				return sdk.WrapError(err, "unable to get network for %s on %s", sdk.StringFirstN(ID, 7), dockerClient.name)
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

				c := h.getContainer(containers, id)
				if c != nil {
					// Send final logs before deleting service container
					jobIdentifiers := h.GetIdentifiersFromLabels(*c)
					if jobIdentifiers == nil {
						log.Error(ctx, "killAwolWorker> unable to get identifiers from containers labels")
						continue
					}
					endLog := log.Message{
						Level: logrus.InfoLevel,
						Value: string("End of Job"),
						Signature: log.Signature{
							Service: &log.SignatureService{
								HatcheryID:      h.Service().ID,
								HatcheryName:    h.ServiceName(),
								RequirementID:   jobIdentifiers.ServiceID,
								RequirementName: c.Labels[hatchery.LabelServiceReqName],
								WorkerName:      c.Labels["service_worker"],
							},
							ProjectKey:   c.Labels[hatchery.LabelServiceProjectKey],
							WorkflowName: c.Labels[hatchery.LabelServiceWorkflowName],
							WorkflowID:   jobIdentifiers.WorkflowID,
							RunID:        jobIdentifiers.RunID,
							NodeRunName:  c.Labels[hatchery.LabelServiceNodeRunName],
							JobName:      c.Labels[hatchery.LabelServiceJobName],
							JobID:        jobIdentifiers.JobID,
							NodeRunID:    jobIdentifiers.NodeRunID,
							Timestamp:    time.Now().UnixNano(),
						},
					}
					h.Common.SendServiceLog(ctx, []log.Message{endLog}, sdk.StatusTerminated)
				}

				if err := h.killAndRemoveContainer(ctx, dockerClient, id); err != nil {
					log.Error(ctx, "hatchery> swarm> killAndRemove> unable to kill and remove container %s on %s err:%s", sdk.StringFirstN(id, 12), dockerClient.name, err)
				}
			}
		}

		//Finally remove the network
		log.Info(ctx, "hatchery> swarm> remove network %s (%s)", network.Name, network.ID)
		ctxDocker, cancelList := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelList()
		if err := dockerClient.NetworkRemove(ctxDocker, network.ID); err != nil {
			log.Error(ctx, "hatchery> swarm> killAndRemove> unable to kill and remove network %s from %s err:%s", sdk.StringFirstN(network.ID, 12), dockerClient.name, err)
		}
	}
	return nil
}

func (h *HatcherySwarm) killAndRemoveContainer(ctx context.Context, dockerClient *dockerClient, ID string) error {
	log.Debug("hatchery> swarm> killAndRemove> remove container %s on %s", ID, dockerClient.name)
	ctxDocker, cancelList := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelList()
	if err := dockerClient.ContainerKill(ctxDocker, ID, "SIGKILL"); err != nil {
		if !strings.Contains(err.Error(), "is not running") && !strings.Contains(err.Error(), "No such container") {
			return sdk.WrapError(err, "err on kill container %v from %s", err, dockerClient.name)
		}
	}

	ctxDockerRemove, cancelList := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelList()
	if err := dockerClient.ContainerRemove(ctxDockerRemove, ID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true}); err != nil {
		// container could be already removed by a previous call to docker
		if !strings.Contains(err.Error(), "No such container") && !strings.Contains(err.Error(), "is already in progress") {
			log.Error(ctx, "Unable to remove container %s from %s: %v", ID, dockerClient.name, err)
		}
	}

	return nil
}

func (h *HatcherySwarm) killAwolNetworks(ctx context.Context) error {
	for _, dockerClient := range h.dockerClients {
		//Checking networks
		ctxDocker, cancelList := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelList()
		nets, errLN := dockerClient.NetworkList(ctxDocker, types.NetworkListOptions{})
		if errLN != nil {
			log.Warning(ctx, "hatchery> swarm> killAwolNetworks> Cannot get networks on %s: %s", dockerClient.name, errLN)
			return errLN
		}

		for i := range nets {
			ctxDocker, cancelList := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelList()
			n, err := dockerClient.NetworkInspect(ctxDocker, nets[i].ID, types.NetworkInspectOptions{})
			if err != nil {
				log.Warning(ctx, "hatchery> swarm> killAwolNetworks> Unable to get network info: %v", err)
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
			if time.Since(n.Created) < 10*time.Minute {
				continue
			}

			log.Info(ctx, "hatchery> swarm> killAwolNetworks> remove network[%s] %s on %s (created on %v)", n.ID, n.Name, dockerClient.name, n.Created)
			ctxDocker2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := dockerClient.NetworkRemove(ctxDocker2, n.ID); err != nil {
				log.Warning(ctx, "hatchery> swarm> killAwolNetworks> Unable to delete network %s err:%s", n.Name, err)
			}
		}
	}
	return nil
}
