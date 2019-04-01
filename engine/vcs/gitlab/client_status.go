package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type statusData struct {
	status       string
	branchName   string
	url          string
	desc         string
	repoFullName string
	hash         string
}

func getGitlabStateFromStatus(s string) gitlab.BuildState {
	switch s {
	case sdk.StatusWaiting.String():
		return gitlab.Pending
	case sdk.StatusChecking.String():
		return gitlab.Pending
	case sdk.StatusBuilding.String():
		return gitlab.Running
	case sdk.StatusSuccess.String():
		return gitlab.Success
	case sdk.StatusFail.String():
		return gitlab.Failed
	case sdk.StatusDisabled.String():
		return gitlab.Canceled
	case sdk.StatusNeverBuilt.String():
		return gitlab.Canceled
	case sdk.StatusUnknown.String():
		return gitlab.Failed
	case sdk.StatusSkipped.String():
		return gitlab.Canceled
	}

	return gitlab.Failed
}

//SetStatus set build status on Gitlab
func (c *gitlabClient) SetStatus(ctx context.Context, event sdk.Event) error {
	if c.disableStatus {
		log.Warning("disableStatus.SetStatus>  ⚠ Gitlab statuses are disabled")
		return nil
	}

	var data statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}):
		data, err = processWorkflowNodeRunEvent(event, c.uiURL)
	default:
		log.Debug("gitlabClient.SetStatus> Unknown event %v", event)
		return nil
	}

	if err != nil {
		return sdk.WrapError(err, "cannot process event %v", event)
	}

	if c.disableStatusDetail {
		data.url = ""
	}

	cds := "CDS"
	opt := &gitlab.SetCommitStatusOptions{
		Name:        &cds,
		Context:     &cds,
		State:       getGitlabStateFromStatus(data.status),
		Ref:         &data.branchName,
		TargetURL:   &data.url,
		Description: &data.desc,
	}

	val, _, err := c.client.Commits.GetCommitStatuses(data.repoFullName, data.hash, nil)
	if err != nil {
		return sdk.WrapError(err, "unable to get commit statuses - repo:%s hash:%s", data.repoFullName, data.hash)
	}

	found := false
	for _, s := range val {
		sameRequest := s.TargetURL == *opt.TargetURL && // Comparing TargetURL as there is the workflow run number inside
			s.Status == string(opt.State) && // Comparing Status to avoid duplicate entries
			s.Ref == *opt.Ref && // Comparing branches name
			s.SHA == data.hash && // Comparing commit SHA to match the right commit
			s.Name == *opt.Name && // Comparing app name (CDS)
			s.Description == *opt.Description // Comparing Description as there are the pipelines names inside

		if sameRequest {
			log.Debug("gitlabClient.SetStatus> Duplicate commit status, ignoring request - repo:%s hash:%s", data.repoFullName, data.hash)
			found = true
			break
		}
	}
	if !found {
		if _, _, err := c.client.Commits.SetCommitStatus(data.repoFullName, data.hash, opt); err != nil {
			return sdk.WrapError(err, "cannot process event %v - repo:%s hash:%s", event, data.repoFullName, data.hash)
		}
	}
	return nil
}

func (c *gitlabClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	ss, _, err := c.client.Commits.GetCommitStatuses(repo, ref, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get commit statuses hash:%s", ref)
	}

	vcsStatuses := []sdk.VCSCommitStatus{}
	for _, s := range ss {
		if !strings.HasPrefix(s.Description, "CDS/") {
			continue
		}
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  *s.CreatedAt,
			Decription: s.Description,
			Ref:        ref,
			State:      processGitlabState(*s),
		})
	}

	return vcsStatuses, nil
}

func processGitlabState(s gitlab.CommitStatus) string {
	switch s.Status {
	case string(gitlab.Success):
		return sdk.StatusSuccess.String()
	case string(gitlab.Failed):
		return sdk.StatusFail.String()
	case string(gitlab.Canceled):
		return sdk.StatusSkipped.String()
	default:
		return sdk.StatusBuilding.String()
	}
}

func processWorkflowNodeRunEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventNR sdk.EventRunWorkflowNode
	if err := mapstructure.Decode(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "cannot read payload")
	}

	data.url = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d",
		uiURL,
		event.ProjectKey,
		event.WorkflowName,
		eventNR.Number,
	)

	data.desc = sdk.VCSCommitStatusDescription(event.ProjectKey, event.WorkflowName, eventNR)
	data.hash = eventNR.Hash
	data.repoFullName = eventNR.RepositoryFullName
	data.status = eventNR.Status
	data.branchName = eventNR.BranchName
	return data, nil
}
