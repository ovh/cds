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
			if _, has := cnt.Labels[hatchery.LabelServiceID]; !has {
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
			if errR != nil {
				logsReader.Close()
				log.Error(ctx, "hatchery> swarm> getServicesLogs> cannot read logs for containers service %s %v : %v", cnt.ID, cnt.Names, errR)
				cancel()
				continue
			}

			cancel()

			if len(logs) > 0 {
				jobIdentifiers := h.GetIdentifiersFromLabels(cnt)
				if jobIdentifiers == nil {
					logsReader.Close()
					continue
				}

				servicesLogs = append(servicesLogs, sdk.ServiceLog{
					WorkflowNodeJobRunID:   jobIdentifiers.JobID,
					WorkflowNodeRunID:      jobIdentifiers.NodeRunID,
					ServiceRequirementID:   jobIdentifiers.ServiceID,
					ServiceRequirementName: cnt.Labels[hatchery.LabelServiceReqName],
					Val:                    string(logs),
					WorkerName:             workerName,
					JobName:                cnt.Labels[hatchery.LabelServiceJobName],
					NodeRunName:            cnt.Labels[hatchery.LabelServiceNodeRunName],
					WorkflowName:           cnt.Labels[hatchery.LabelServiceWorkflowName],
					ProjectKey:             cnt.Labels[hatchery.LabelServiceProjectKey],
					RunID:                  jobIdentifiers.RunID,
					WorkflowID:             jobIdentifiers.WorkflowID,
				})
			}
			logsReader.Close()
		}
		if len(servicesLogs) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			h.Common.SendServiceLog(ctx, servicesLogs, sdk.StatusBuilding)
			cancel()
		}
	}
	return nil
}

func (h *HatcherySwarm) GetIdentifiersFromLabels(cnt types.Container) *hatchery.JobIdentifiers {
	serviceIDStr, ok := cnt.Labels[hatchery.LabelServiceID]
	if !ok {
		return nil
	}
	serviceJobIDStr, isWorkflowService := cnt.Labels[hatchery.LabelServiceJobID]
	if !isWorkflowService {
		return nil
	}
	serviceNodeRunIDStr, ok := cnt.Labels[hatchery.LabelServiceNodeRunID]
	if !ok {
		return nil
	}
	runIDStr, ok := cnt.Labels[hatchery.LabelServiceRunID]
	if !ok {
		return nil
	}
	workflowIDStr, ok := cnt.Labels[hatchery.LabelServiceWorkflowID]
	if !ok {
		return nil
	}

	serviceID, errP := strconv.ParseInt(serviceIDStr, 10, 64)
	if errP != nil {
		return nil
	}
	serviceJobID, errPj := strconv.ParseInt(serviceJobIDStr, 10, 64)
	if errPj != nil {
		return nil
	}
	serviceNodeRunID, err := strconv.ParseInt(serviceNodeRunIDStr, 10, 64)
	if err != nil {
		return nil
	}
	serviceRunID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		return nil
	}
	serviceWorkflowID, err := strconv.ParseInt(workflowIDStr, 10, 64)
	if err != nil {
		return nil
	}

	return &hatchery.JobIdentifiers{
		WorkflowID: serviceWorkflowID,
		RunID:      serviceRunID,
		NodeRunID:  serviceNodeRunID,
		JobID:      serviceJobID,
		ServiceID:  serviceID,
	}
}
