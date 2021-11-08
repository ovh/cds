package swarm

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/hatchery"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (h *HatcherySwarm) getServicesLogs() error {
	ctx := context.Background()

	ctxList, cancelList := context.WithTimeout(ctx, 10*time.Second)
	defer cancelList()

	apiWorkers, err := h.CDSClient().WorkerList(ctxList)
	if err != nil {
		return sdk.WrapError(err, "cannot get worker list from CDS api")
	}
	apiWorkerNames := make(map[string]struct{}, len(apiWorkers))
	for i := range apiWorkers {
		apiWorkerNames[apiWorkers[i].Name] = struct{}{}
	}

	for _, dockerClient := range h.dockerClients {
		containers, err := h.getContainers(ctx, dockerClient, types.ContainerListOptions{All: true})
		if err != nil {
			return sdk.WrapError(err, "Cannot get containers list from %s", dockerClient.name)
		}

		servicesLogs := make([]cdslog.Message, 0, len(containers))
		for _, cnt := range containers {
			if _, has := cnt.Labels[hatchery.LabelServiceID]; !has {
				continue
			}

			workerName := cnt.Labels[LabelServiceWorker]
			// Check if there is a known worker in CDS api results for given worker name
			// If not we skip sending logs as the worker is not ready.
			// This will avoid problems validating log signature by the CDN service.
			if _, ok := apiWorkerNames[workerName]; !ok {
				continue
			}

			ctxLogs, cancelLogs := context.WithTimeout(ctx, time.Minute*2)
			logsOpts := types.ContainerLogsOptions{
				Details:    true,
				ShowStderr: true,
				ShowStdout: true,
				Since:      "10s",
			}
			logsReader, err := dockerClient.ContainerLogs(ctxLogs, cnt.ID, logsOpts)
			if err != nil {
				err = sdk.WrapError(err, "cannot get logs from docker for containers service %s %v", cnt.ID, cnt.Names)
				ctx := sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, err.Error())
				cancelLogs()
				continue
			}

			logs, err := io.ReadAll(logsReader)
			if err != nil {
				logsReader.Close() // nolint
				err = sdk.WrapError(err, "cannot read logs for containers service %s %v", cnt.ID, cnt.Names)
				ctx := sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, err.Error())
				cancelLogs()
				continue
			}

			cancelLogs()

			if len(logs) > 0 {
				jobIdentifiers := h.GetIdentifiersFromLabels(cnt)
				if jobIdentifiers == nil {
					logsReader.Close()
					continue
				}

				commonMessage := cdslog.Message{
					Level: logrus.InfoLevel,
					Signature: cdn.Signature{
						Service: &cdn.SignatureService{
							HatcheryID:      h.Service().ID,
							HatcheryName:    h.ServiceName(),
							RequirementID:   jobIdentifiers.ServiceID,
							RequirementName: cnt.Labels[hatchery.LabelServiceReqName],
							WorkerName:      workerName,
						},
						ProjectKey:   cnt.Labels[hatchery.LabelServiceProjectKey],
						WorkflowName: cnt.Labels[hatchery.LabelServiceWorkflowName],
						WorkflowID:   jobIdentifiers.WorkflowID,
						RunID:        jobIdentifiers.RunID,
						NodeRunName:  cnt.Labels[hatchery.LabelServiceNodeRunName],
						JobName:      cnt.Labels[hatchery.LabelServiceJobName],
						JobID:        jobIdentifiers.JobID,
						NodeRunID:    jobIdentifiers.NodeRunID,
					},
				}

				logsSplitted := strings.Split(string(logs), "\n")
				for i := range logsSplitted {
					if i == len(logsSplitted)-1 && logsSplitted[i] == "" {
						break
					}
					msg := commonMessage
					msg.Signature.Timestamp = time.Now().UnixNano()
					msg.Value = sdk.RemoveNotPrintableChar(logsSplitted[i])
					servicesLogs = append(servicesLogs, msg)
				}
			}
			logsReader.Close()
		}
		if len(servicesLogs) > 0 {
			ctxSend, cancelSend := context.WithTimeout(ctx, 10*time.Second)
			h.Common.SendServiceLog(ctxSend, servicesLogs, sdk.StatusNotTerminated)
			cancelSend()
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
