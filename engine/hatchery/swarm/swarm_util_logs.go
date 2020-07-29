package swarm

import (
	"context"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

func (h *HatcherySwarm) getServicesLogs() error {
	for _, dockerClient := range h.dockerClients {
		containers, err := h.getContainers(dockerClient, types.ContainerListOptions{All: true})
		if err != nil {
			return sdk.WrapError(err, "Cannot get containers list from %s", dockerClient.name)
		}

		servicesLogs := make([]sdk.ServiceLog, 0, len(containers))
		for _, cnt := range containers {
			serviceJobIDStr, isWorkflowService := cnt.Labels[hatchery.LabelServiceJobID]
			if !isWorkflowService {
				continue
			}
			serviceNodeRunIDStr, ok := cnt.Labels[hatchery.LabelServiceNodeRunID]
			if !ok {
				continue
			}
			runIDStr, ok := cnt.Labels[hatchery.LabelServiceRunID]
			if !ok {
				continue
			}
			workflowIDStr, ok := cnt.Labels[hatchery.LabelServiceWorkflowID]
			if !ok {
				continue
			}

			workerName := cnt.Labels["service_worker"]
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
				log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot get logs from docker for containers service %s %v : %v", cnt.ID, cnt.Names, errL)
				cancel()
				continue
			}

			logs, errR := ioutil.ReadAll(logsReader)
			defer logsReader.Close()
			if errR != nil {
				log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot read logs for containers service %s %v : %v", cnt.ID, cnt.Names, errR)
				cancel()
				continue
			}

			cancel()

			if len(logs) > 0 {
				serviceID, ok := cnt.Labels[hatchery.LabelServiceID]
				if !ok {
					log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot find label service id for containers service %s %v", cnt.ID, cnt.Names)
					continue
				}

				reqServiceID, errP := strconv.ParseInt(serviceID, 10, 64)
				if errP != nil {
					log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot parse service id for containers service %s %v id : %s, err : %v", cnt.ID, cnt.Names, serviceID, errP)
					continue
				}
				serviceJobID, errPj := strconv.ParseInt(serviceJobIDStr, 10, 64)
				if errPj != nil {
					log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot parse service job id for containers service %s %v id : %s, err : %v", cnt.ID, cnt.Names, serviceJobIDStr, errPj)
					continue
				}
				serviceNodeRunID, err := strconv.ParseInt(serviceNodeRunIDStr, 10, 64)
				if err != nil {
					log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot parse service node run id for containers service %s %v id : %s, err : %v", cnt.ID, cnt.Names, serviceNodeRunIDStr, errPj)
					continue
				}
				serviceRunID, err := strconv.ParseInt(runIDStr, 10, 64)
				if err != nil {
					log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot parse service run id for containers service %s %v id : %s, err : %v", cnt.ID, cnt.Names, serviceNodeRunIDStr, errPj)
					continue
				}
				serviceWorkflowID, err := strconv.ParseInt(workflowIDStr, 10, 64)
				if err != nil {
					log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot parse service workflow id for containers service %s %v id : %s, err : %v", cnt.ID, cnt.Names, serviceNodeRunIDStr, errPj)
					continue
				}

				servicesLogs = append(servicesLogs, sdk.ServiceLog{
					WorkflowNodeJobRunID:   serviceJobID,
					WorkflowNodeRunID:      serviceNodeRunID,
					ServiceRequirementID:   reqServiceID,
					ServiceRequirementName: cnt.Labels[hatchery.LabelServiceReqName],
					Val:                    string(logs),
					WorkerName:             workerName,
					JobName:                cnt.Labels[hatchery.LabelServiceJobName],
					NodeRunName:            cnt.Labels[hatchery.LabelServiceNodeRunName],
					WorkflowName:           cnt.Labels[hatchery.LabelServiceWorkflowName],
					ProjectKey:             cnt.Labels[hatchery.LabelServiceProjectKey],
					RunID:                  serviceRunID,
					WorkflowID:             serviceWorkflowID,
				})
			}
		}
		if len(servicesLogs) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			h.Common.SendServiceLog(ctx, servicesLogs)
			cancel()
		}
	}
	return nil
}
