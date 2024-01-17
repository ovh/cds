package bitbucketserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

type statusData struct {
	key         string
	buildNumber int64
	status      string
	url         string
	hash        string
	description string
}

func (client *bitbucketClient) SetDisableStatusDetails(disableStatusDetails bool) {
	client.disableStatusDetails = disableStatusDetails
}

func (client *bitbucketClient) SetStatus(ctx context.Context, event sdk.Event) error {
	ctx, end := telemetry.Span(ctx, "bitbucketserver.SetStatus")
	defer end()

	var statusData statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}):
		statusData, err = client.processWorkflowNodeRunEvent(event, client.consumer.uiURL)
	default:
		return nil
	}

	if err != nil {
		return sdk.WrapError(err, "bitbucketClient.SetStatus: Cannot process Event")
	}

	state := getBitbucketStateFromStatus(statusData.status)
	status := Status{
		Key:         statusData.key,
		Name:        fmt.Sprintf("%s%d", statusData.key, statusData.buildNumber),
		State:       state,
		URL:         statusData.url,
		Description: statusData.description,
	}

	values, err := json.Marshal(status)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshall status")
	}

	log.Info(ctx, "sending build status for %s : %s %s - %s", statusData.hash, status.Key, status.Name, state)

	if err := client.do(ctx, "POST", "build-status", fmt.Sprintf("/commits/%s", statusData.hash), nil, values, nil); err != nil {
		return sdk.WrapError(err, "Unable to post build-status name:%s status:%s", status.Name, state)
	}
	return nil
}

func (client *bitbucketClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	ss := []Status{}

	path := fmt.Sprintf("/commits/%s", ref)
	params := url.Values{}
	nextPage := 0
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response ResponseStatus
		if err := client.do(ctx, "GET", "build-status", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get statuses")
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
		return sdk.StatusSuccess
	case failed:
		return sdk.StatusFail
	default:
		return sdk.StatusDisabled
	}
}

const (
	// "state": "<INPROGRESS|SUCCESSFUL|FAILED>"
	// doc from https://developer.atlassian.com/server/bitbucket/how-tos/updating-build-status-for-commits/
	inProgress = "INPROGRESS"
	successful = "SUCCESSFUL"
	failed     = "FAILED"
)

func (client *bitbucketClient) processWorkflowNodeRunEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventNR sdk.EventRunWorkflowNode
	if err := sdk.JSONUnmarshal(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "cannot unmarshal payload")
	}
	data.key = fmt.Sprintf("%s-%s-%s",
		event.ProjectKey,
		event.WorkflowName,
		eventNR.NodeName,
	)
	if !client.disableStatusDetails {
		data.url = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d",
			uiURL,
			event.ProjectKey,
			event.WorkflowName,
			eventNR.Number,
		)
	}
	data.buildNumber = eventNR.Number
	data.status = eventNR.Status
	data.hash = eventNR.Hash
	data.description = sdk.VCSCommitStatusDescription(event.ProjectKey, event.WorkflowName, eventNR)

	return data, nil
}

func getBitbucketStateFromStatus(status string) string {
	switch status {
	case sdk.StatusSuccess, sdk.StatusSkipped, sdk.StatusDisabled:
		return successful
	case sdk.StatusWaiting, sdk.StatusBuilding:
		return inProgress
	case sdk.StatusFail:
		return failed
	default:
		return failed
	}
}
