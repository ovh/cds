package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

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
	description string
}

func (b *bitbucketClient) SetStatus(event sdk.Event) error {
	if b.consumer.disableStatus {
		log.Warning("bitbucketClient.SetStatus>  ⚠ Bitbucket statuses are disabled")
		return nil
	}

	var statusData statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventPipelineBuild{}):
		statusData, err = processPipelineBuildEvent(event, b.consumer.uiURL)
	case fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}):
		statusData, err = processWorkflowNodeRunEvent(event, b.consumer.uiURL)
	default:
		return nil
	}

	if err != nil {
		return sdk.WrapError(err, "bitbucketClient.SetStatus: Cannot process Event")
	}

	status := Status{
		Key:         statusData.key,
		Name:        fmt.Sprintf("%s%d", statusData.key, statusData.buildNumber),
		State:       getBitbucketStateFromStatus(statusData.status),
		URL:         statusData.url,
		Description: statusData.description,
	}

	values, err := json.Marshal(status)
	if err != nil {
		return sdk.WrapError(err, "bitbucketClient.SetStatus> Unable to marshall status")
	}
	return b.do("POST", "build-status", fmt.Sprintf("/commits/%s", statusData.hash), nil, values, nil, nil)
}

func (b *bitbucketClient) ListStatuses(repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	ss := []Status{}

	path := fmt.Sprintf("/commits/%s", ref)
	params := url.Values{}
	nextPage := 0
	for {
		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response ResponseStatus
		if err := b.do("GET", "build-status", path, nil, nil, &response, nil); err != nil {
			return nil, sdk.WrapError(err, "vcs> bitbucket> Repos> Unable to get statuses")
		}

		ss = append(ss, response.Values...)

		if response.IsLastPage {
			break
		} else {
			nextPage = response.NextPageStart
		}
	}

	vcsStatuses := []sdk.VCSCommitStatus{}
	for _, s := range ss {
		if !strings.HasPrefix(s.Description, "CDS/") {
			continue
		}
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  time.Unix(s.Timestamp/1000, 0),
			Decription: s.Description,
			Ref:        ref,
			State:      processBitbucketState(s),
		})
	}

	return vcsStatuses, nil
}

func processBitbucketState(s Status) string {
	switch s.State {
	case successful:
		return sdk.StatusSuccess.String()
	case failed:
		return sdk.StatusFail.String()
	default:
		return sdk.StatusBuilding.String()
	}
}

const (
	inProgress = "INPROGRESS"
	successful = "SUCCESSFUL"
	failed     = "FAILED"
)

func processWorkflowNodeRunEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventNR sdk.EventRunWorkflowNode
	if err := mapstructure.Decode(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "bitbucketClient.processWorkflowNodeRunEvent> Error during consumption")
	}
	data.key = fmt.Sprintf("%s-%s-%s",
		event.ProjectKey,
		event.WorkflowName,
		eventNR.NodeName,
	)
	data.url = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d",
		uiURL,
		event.ProjectKey,
		event.WorkflowName,
		eventNR.Number,
	)
	data.buildNumber = eventNR.Number
	data.status = eventNR.Status
	data.hash = eventNR.Hash
	data.description = sdk.VCSCommitStatusDescription(event.ProjectKey, event.WorkflowName, eventNR)

	return data, nil
}

func processPipelineBuildEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventpb sdk.EventPipelineBuild
	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		return data, sdk.WrapError(err, "bitbucketClient.processPipelineBuildEvent> Error during consumption")
	}
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
