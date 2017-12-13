package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type statusData struct {
	key         string
	buildNumber int64
	status      string
	url         string
	hash        string
}

func (b *bitbucketClient) SetStatus(event sdk.Event) error {
	log.Info("bitbucketClient.SetStatus> receive: type:%s all: %+v", event.EventType, event)

	if b.consumer.disableStatus {
		log.Warning("bitbucketClient.SetStatus>  ⚠ Bitbucket statuses are disabled")
		return nil
	}

	var statusData statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventPipelineBuild{}):
		statusData, err = processPipelineBuildEvent(event, b.consumer.uiURL)
	case fmt.Sprintf("%T", sdk.EventWorkflowNodeRun{}):
		statusData, err = processWorkflowNodeRunEvent(event, b.consumer.uiURL)
	default:
		return nil
	}

	if err != nil {
		return sdk.WrapError(err, "bitbucketClient.SetStatus: Cannot process Event")
	}

	status := Status{
		Key:   statusData.key,
		Name:  fmt.Sprintf("%s%d", statusData.key, statusData.buildNumber),
		State: getBitbucketStateFromStatus(statusData.status),
		URL:   statusData.url,
	}

	log.Debug("SetStatus> hash:%s status:%+v", statusData.hash, status)

	values, err := json.Marshal(status)
	if err != nil {
		return sdk.WrapError(err, "bitbucketClient.SetStatus> Unable to marshall status")
	}
	log.Debug("SetStatus> Values: %+v", values)
	return b.do("POST", "build-status", fmt.Sprintf("/commits/%s", statusData.hash), nil, values, nil)
}

const (
	inProgress = "INPROGRESS"
	successful = "SUCCESSFUL"
	failed     = "FAILED"
)

func processWorkflowNodeRunEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventNR sdk.EventWorkflowNodeRun
	if err := mapstructure.Decode(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "bitbucketClient.processWorkflowNodeRunEvent> Error during consumption")
	}
	log.Debug("bitbucketClient.processWorkflowNodeRunEvent>Process event:%+v", event)
	data.key = fmt.Sprintf("%s-%s-%s",
		eventNR.ProjectKey,
		eventNR.WorkflowName,
		eventNR.NodeName,
	)
	data.url = fmt.Sprintf("%s/project/%s/workflow/%s/run/%s",
		uiURL,
		eventNR.ProjectKey,
		eventNR.WorkflowName,
		eventNR.Number,
	)
	data.buildNumber = eventNR.Number
	data.status = eventNR.Status
	data.hash = eventNR.Hash

	return data, nil
}

func processPipelineBuildEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventpb sdk.EventPipelineBuild
	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		return data, sdk.WrapError(err, "bitbucketClient.processPipelineBuildEvent> Error during consumption")
	}

	log.Debug("bitbucketClient.processPipelineBuildEvent> Process event:%+v", event)

	cdsProject := eventpb.ProjectKey
	cdsApplication := eventpb.ApplicationName
	cdsPipelineName := eventpb.PipelineName
	cdsBuildNumber := eventpb.BuildNumber
	cdsEnvironmentName := eventpb.EnvironmentName

	data.key = fmt.Sprintf("%s-%s-%s",
		cdsProject,
		cdsApplication,
		cdsPipelineName,
	)

	data.url = fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%d?envName=%s",
		uiURL,
		cdsProject,
		cdsApplication,
		cdsPipelineName,
		cdsBuildNumber,
		url.QueryEscape(cdsEnvironmentName),
	)
	data.buildNumber = cdsBuildNumber
	data.status = eventpb.Status.String()
	data.hash = eventpb.Hash
	return data, nil
}

func getBitbucketStateFromStatus(status string) string {
	switch status {
	case sdk.StatusSuccess.String():
		return successful
	case sdk.StatusWaiting.String():
		return inProgress
	case sdk.StatusBuilding.String():
		return inProgress
	case sdk.StatusFail.String():
		return failed
	default:
		return failed
	}
}
