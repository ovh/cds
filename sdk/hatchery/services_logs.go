package hatchery

import (
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/sdk/cdn"
	cdslog "github.com/ovh/cds/sdk/log"
)

func PrepareCommonLogMessage(hatcheryServiceName string, hatcheryServiceID int64, jobIdentifiers JobIdentifiers, labels map[string]string) cdslog.Message {
	commonMessage := cdslog.Message{}
	if jobIdentifiers.IsJobV2() {
		runNumber, _ := strconv.ParseInt(labels[LabelServiceRunNumber], 10, 64)
		runAttempt, _ := strconv.ParseInt(labels[LabelServiceRunAttempt], 10, 64)

		commonMessage = cdslog.Message{
			Level: logrus.InfoLevel,
			Signature: cdn.Signature{
				ProjectKey:    labels[LabelServiceProjectKey],
				WorkflowName:  labels[LabelServiceWorkflowName],
				JobName:       labels[LabelServiceJobName],
				RunJobID:      jobIdentifiers.JobIdentifiersV2.RunJobID,
				WorkflowRunID: labels[LabelServiceRunID],
				RunNumber:     runNumber,
				RunAttempt:    runAttempt,
			},
		}

		if v, ok := labels[LabelServiceReqName]; ok && v != "" {
			commonMessage.Signature.HatcheryService = &cdn.SignatureHatcheryService{
				HatcheryName: hatcheryServiceName,
				HatcheryID:   strconv.FormatInt(hatcheryServiceID, 10),
				ServiceName:  labels[LabelServiceReqName],
			}
		}
	} else {
		commonMessage = cdslog.Message{
			Level: logrus.InfoLevel,
			Signature: cdn.Signature{
				ProjectKey:   labels[LabelServiceProjectKey],
				WorkflowName: labels[LabelServiceWorkflowName],
				WorkflowID:   jobIdentifiers.JobIdentifiersV1.WorkflowID,
				RunID:        jobIdentifiers.JobIdentifiersV1.RunID,
				NodeRunName:  labels[LabelServiceNodeRunName],
				JobName:      labels[LabelServiceJobName],
				JobID:        jobIdentifiers.JobIdentifiersV1.JobID,
				NodeRunID:    jobIdentifiers.JobIdentifiersV1.NodeRunID,
			},
		}
		if v, ok := labels[LabelServiceReqName]; ok && v != "" {
			commonMessage.Signature.Service = &cdn.SignatureService{
				HatcheryID:      hatcheryServiceID,
				HatcheryName:    hatcheryServiceName,
				RequirementID:   jobIdentifiers.JobIdentifiersV1.ServiceID,
				RequirementName: labels[LabelServiceReqName],
				WorkerName:      labels[LabelServiceWorker],
			}
		}
	}
	return commonMessage
}

func GetServiceIdentifiersFromLabels(labels map[string]string) *JobIdentifiers {
	if labels[LabelServiceVersion] == ValueLabelServiceVersion2 {
		serviceJobID, ok := labels[LabelServiceJobID]
		if !ok {
			return nil
		}
		serviceRunJobID, ok := labels[LabelServiceRunJobID]
		if !ok {
			return nil
		}

		runID, ok := labels[LabelServiceRunID]
		if !ok {
			return nil
		}

		return &JobIdentifiers{
			JobIdentifiersV2: JobIdentifiersV2{
				JobID:    serviceJobID,
				RunJobID: serviceRunJobID,
				RunID:    runID,
			},
		}
	}

	serviceIDStr, ok := labels[LabelServiceID]
	if !ok {
		return nil
	}
	serviceJobIDStr, isWorkflowService := labels[LabelServiceJobID]
	if !isWorkflowService {
		return nil
	}
	serviceNodeRunIDStr, ok := labels[LabelServiceNodeRunID]
	if !ok {
		return nil
	}
	runIDStr, ok := labels[LabelServiceRunID]
	if !ok {
		return nil
	}
	workflowIDStr, ok := labels[LabelServiceWorkflowID]
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

	return &JobIdentifiers{
		JobIdentifiersV1: JobIdentifiersV1{
			WorkflowID: serviceWorkflowID,
			RunID:      serviceRunID,
			NodeRunID:  serviceNodeRunID,
			JobID:      serviceJobID,
			ServiceID:  serviceID,
		},
	}
}
