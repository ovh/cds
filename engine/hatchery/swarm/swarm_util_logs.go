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
				workflowID, runID, nodeRunID, jobID, serviceID := h.GetIdentifiersFromLabels(ctx, cnt)
				if workflowID == 0 || runID == 0 || nodeRunID == 0 || jobID == 0 || serviceID == 0 {
					logsReader.Close()
					continue
				}

				servicesLogs = append(servicesLogs, sdk.ServiceLog{
					WorkflowNodeJobRunID:   jobID,
					WorkflowNodeRunID:      nodeRunID,
					ServiceRequirementID:   serviceID,
					ServiceRequirementName: cnt.Labels[hatchery.LabelServiceReqName],
					Val:                    string(logs),
					WorkerName:             workerName,
					JobName:                cnt.Labels[hatchery.LabelServiceJobName],
					NodeRunName:            cnt.Labels[hatchery.LabelServiceNodeRunName],
					WorkflowName:           cnt.Labels[hatchery.LabelServiceWorkflowName],
					ProjectKey:             cnt.Labels[hatchery.LabelServiceProjectKey],
					RunID:                  runID,
					WorkflowID:             workflowID,
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

func (h *HatcherySwarm) GetIdentifiersFromLabels(ctx context.Context, cnt types.Container) (int64, int64, int64, int64, int64) {
	serviceID, ok := cnt.Labels[hatchery.LabelServiceID]
	if !ok {
		return 0, 0, 0, 0, 0
	}
	serviceJobIDStr, isWorkflowService := cnt.Labels[hatchery.LabelServiceJobID]
	if !isWorkflowService {
		return 0, 0, 0, 0, 0
	}
	serviceNodeRunIDStr, ok := cnt.Labels[hatchery.LabelServiceNodeRunID]
	if !ok {
		return 0, 0, 0, 0, 0
	}
	runIDStr, ok := cnt.Labels[hatchery.LabelServiceRunID]
	if !ok {
		return 0, 0, 0, 0, 0
	}
	workflowIDStr, ok := cnt.Labels[hatchery.LabelServiceWorkflowID]
	if !ok {
		return 0, 0, 0, 0, 0
	}

	reqServiceID, errP := strconv.ParseInt(serviceID, 10, 64)
	if errP != nil {
		return 0, 0, 0, 0, 0
	}
	serviceJobID, errPj := strconv.ParseInt(serviceJobIDStr, 10, 64)
	if errPj != nil {
		return 0, 0, 0, 0, 0
	}
	serviceNodeRunID, err := strconv.ParseInt(serviceNodeRunIDStr, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, 0
	}
	serviceRunID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, 0
	}
	serviceWorkflowID, err := strconv.ParseInt(workflowIDStr, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, 0
	}

	return serviceWorkflowID, serviceRunID, serviceNodeRunID, serviceJobID, reqServiceID
}
