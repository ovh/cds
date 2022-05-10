package bitbucketcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

type statusData struct {
	pipName      string
	desc         string
	status       string
	repoFullName string
	hash         string
	urlPipeline  string
	context      string
}

// DEPRECATED VCS
func (client *bitbucketcloudClient) IsDisableStatusDetails(ctx context.Context) bool {
	return client.DisableStatusDetails
}

//SetStatus Users with push access can create commit statuses for a given ref:
func (client *bitbucketcloudClient) SetStatus(ctx context.Context, event sdk.Event, disableStatusDetails bool) error {
	if client.DisableStatus {
		log.Warn(ctx, "bitbucketcloud.SetStatus>  ⚠ bitbucketcloud statuses are disabled")
		return nil
	}

	var data statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}):
		data, err = processEventWorkflowNodeRun(event, client.uiURL, disableStatusDetails)
	default:
		log.Error(ctx, "bitbucketcloud.SetStatus> Unknown event %v", event)
		return nil
	}
	if err != nil {
		return sdk.WrapError(err, "Cannot process Event")
	}

	if data.status == "" {
		log.Debug(ctx, "bitbucketcloud.SetStatus> Do not process event for current status: %v", event)
		return nil
	}

	bbStatus := Status{
		Description: data.desc,
		URL:         data.urlPipeline,
		State:       data.status,
		Name:        data.context,
		Key:         data.context,
	}

	path := fmt.Sprintf("/repositories/%s/commit/%s/statuses/build", data.repoFullName, data.hash)
	b, err := json.Marshal(bbStatus)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal github status")
	}
	buf := bytes.NewBuffer(b)

	res, err := client.post(ctx, path, "application/json", buf, nil)
	if err != nil {
		return sdk.WrapError(err, "Unable to post status")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "Unable to read body")
	}
	if res.StatusCode != 201 && res.StatusCode != 200 {
		return fmt.Errorf("Unable to create status on bitbucket cloud. Status code : %d - Body: %s - target:%s", res.StatusCode, body, data.urlPipeline)
	}

	var resp Status
	if err := sdk.JSONUnmarshal(body, &resp); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal body")
	}

	log.Debug(ctx, "bitbucketcloud.SetStatus> Status %s %s created at %v", resp.UUID, resp.Links.Self.Href, resp.CreatedOn)

	return nil
}

func (client *bitbucketcloudClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	url := fmt.Sprintf("/repositories/%s/commit/%s/statuses", repo, ref)
	status, body, _, err := client.get(ctx, url)
	if err != nil {
		return []sdk.VCSCommitStatus{}, sdk.WrapError(err, "bitbucketcloudClient.ListStatuses")
	}
	if status >= 400 {
		return []sdk.VCSCommitStatus{}, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	var ss Statuses
	if err := sdk.JSONUnmarshal(body, &ss); err != nil {
		return []sdk.VCSCommitStatus{}, sdk.WrapError(err, "Unable to parse bitbucket cloud commit: %s", ref)
	}

	vcsStatuses := make([]sdk.VCSCommitStatus, 0, ss.Size)
	for _, s := range ss.Values {
		if !strings.HasPrefix(s.Name, "CDS/") {
			continue
		}
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  s.CreatedOn,
			Decription: s.Description,
			Ref:        ref,
			State:      processBbitbucketState(s),
		})
	}

	return vcsStatuses, nil
}

func processBbitbucketState(s Status) string {
	switch s.State {
	case "SUCCESSFUL":
		return sdk.StatusSuccess
	case "FAILED":
		return sdk.StatusFail
	case "STOPPED":
		return sdk.StatusStopped
	default:
		return sdk.StatusBuilding
	}
}

func processEventWorkflowNodeRun(event sdk.Event, cdsUIURL string, disableStatusDetails bool) (statusData, error) {
	data := statusData{}
	var eventNR sdk.EventRunWorkflowNode
	if err := sdk.JSONUnmarshal(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "cannot unmarshal payload")
	}
	//We only manage status Success, Failure and Stopped
	if eventNR.Status == sdk.StatusChecking ||
		eventNR.Status == sdk.StatusDisabled ||
		eventNR.Status == sdk.StatusNeverBuilt ||
		eventNR.Status == sdk.StatusSkipped ||
		eventNR.Status == sdk.StatusUnknown ||
		eventNR.Status == sdk.StatusWaiting {
		return data, nil
	}

	switch eventNR.Status {
	case sdk.StatusFail:
		data.status = "FAILED"
	case sdk.StatusSuccess, sdk.StatusSkipped:
		data.status = "SUCCESSFUL"
	case sdk.StatusStopped:
		data.status = "STOPPED"
	default:
		data.status = "INPROGRESS"
	}
	data.hash = eventNR.Hash
	data.repoFullName = eventNR.RepositoryFullName
	data.pipName = eventNR.NodeName

	if !disableStatusDetails {
		data.urlPipeline = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d",
			cdsUIURL,
			event.ProjectKey,
			event.WorkflowName,
			eventNR.Number,
		)
	} else {
		//CDS can avoid sending bitbucket target url in status, if it's disable
		if disableStatusDetails {
			data.urlPipeline = "https://ovh.github.io/cds/" // because it's mandatory
		}
	}

	data.context = sdk.VCSCommitStatusDescription(event.ProjectKey, event.WorkflowName, eventNR)
	data.desc = eventNR.NodeName + ": " + eventNR.Status
	return data, nil
}
