package gitlab

import (
	"fmt"
	"net/url"

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
func (c *gitlabClient) SetStatus(event sdk.Event) error {
	if c.disableStatus {
		log.Warning("disableStatus.SetStatus>  ⚠ Gitlab statuses are disabled")
		return nil
	}

	var data statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventPipelineBuild{}):
		data, err = processPipelineBuildEvent(event, c.uiURL)
	case fmt.Sprintf("%T", sdk.EventWorkflowNodeRun{}):
		data, err = processWorkflowNodeRunEvent(event, c.uiURL)
	default:
		log.Debug("gitlabClient.SetStatus> Unknown event %s", event)
		return nil
	}

	if err != nil {
		return sdk.WrapError(err, "gitlabClient.SetStatus> Cannot process event %s", event)
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

	if _, _, err := c.client.Commits.SetCommitStatus(data.repoFullName, data.hash, opt); err != nil {
		return err
	}

	return nil
}

func processWorkflowNodeRunEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventNR sdk.EventWorkflowNodeRun
	if err := mapstructure.Decode(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "gitlabClient.processPipelineBuildEvent> cannot read payload")
	}

	log.Debug("Process event:%+v", event)

	key := fmt.Sprintf("%s-%s-%s",
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

	data.desc = fmt.Sprintf("Build #%d.%d %s", eventNR.Number, eventNR.SubNumber, key)
	data.hash = eventNR.Hash
	data.repoFullName = eventNR.RepositoryFullName
	data.status = eventNR.Status
	data.branchName = eventNR.BranchName
	return data, nil
}

func processPipelineBuildEvent(event sdk.Event, uiURL string) (statusData, error) {
	data := statusData{}
	var eventpb sdk.EventPipelineBuild
	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		return data, sdk.WrapError(err, "gitlabClient.processPipelineBuildEvent> cannot read payload")
	}

	log.Debug("Process event:%+v", event)

	cdsProject := eventpb.ProjectKey
	cdsApplication := eventpb.ApplicationName
	cdsPipelineName := eventpb.PipelineName
	cdsBuildNumber := eventpb.BuildNumber
	cdsEnvironmentName := eventpb.EnvironmentName

	key := fmt.Sprintf("%s-%s-%s",
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

	data.desc = fmt.Sprintf("Build #%d %s", eventpb.BuildNumber, key)
	data.hash = eventpb.Hash
	data.repoFullName = eventpb.RepositoryFullname
	data.status = eventpb.Status.String()
	data.branchName = eventpb.BranchName
	return data, nil
}
