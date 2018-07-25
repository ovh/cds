package swarm

import (
	"context"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcherySwarm) getServicesLogs() error {
	for _, dockerClient := range h.dockerClients {
		containers, err := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if err != nil {
			return sdk.WrapError(err, "hatchery> swarm> getServicesLogs> Cannot get containers list from %s", dockerClient.name)
		}

		servicesLogs := make([]sdk.ServiceLog, 0, len(containers))
		for _, cnt := range containers {
			serviceJobIDStr, isWorkflowService := cnt.Labels["service_job_id"]
			if !isWorkflowService {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
			logsOpts := types.ContainerLogsOptions{
				Details:    true,
				ShowStderr: true,
				ShowStdout: true,
				Timestamps: true,
				Since:      "10s",
			}
			logsReader, errL := dockerClient.ContainerLogs(ctx, cnt.ID, logsOpts)
			if errL != nil {
				log.Error("hatchery> swarm> getServicesLogs> cannot get logs from docker for containers service %s %v : %v", cnt.ID, cnt.Names, errL)
				cancel()
				continue
			}

			logs, errR := ioutil.ReadAll(logsReader)
			defer logsReader.Close()
			if errR != nil {
				log.Error("hatchery> swarm> getServicesLogs> cannot read logs for containers service %s %v : %v", cnt.ID, cnt.Names, errR)
				cancel()
				continue
			}

			cancel()

			if len(logs) > 0 {
				serviceID, ok := cnt.Labels["service_id"]
				if !ok {
					log.Error("hatchery> swarm> getServicesLogs> cannot find label service id for containers service %s %v", cnt.ID, cnt.Names)
					continue
				}

				reqServiceID, errP := strconv.ParseInt(serviceID, 10, 64)
				if errP != nil {
					log.Error("hatchery> swarm> getServicesLogs> cannot parse service id for containers service %s %v id : %s, err : %v", cnt.ID, cnt.Names, serviceID, errP)
					continue
				}
				serviceJobID, errPj := strconv.ParseInt(serviceJobIDStr, 10, 64)
				if errPj != nil {
					log.Error("hatchery> swarm> getServicesLogs> cannot parse service job id for containers service %s %v id : %s, err : %v", cnt.ID, cnt.Names, serviceJobIDStr, errPj)
					continue
				}

				servicesLogs = append(servicesLogs, sdk.ServiceLog{
					WorkflowNodeJobRunID:   serviceJobID,
					ServiceRequirementID:   reqServiceID,
					ServiceRequirementName: cnt.Labels["service_req_name"],
					Val: string(logs),
				})
			}

			if len(servicesLogs) > 0 {
				// Do call api
				if err := h.Client.QueueServiceLogs(servicesLogs); err != nil {
					log.Error("Hatchery> Swarm> Cannot send service logs : %v", err)
				}
			}
		}
	}
	return nil
}
