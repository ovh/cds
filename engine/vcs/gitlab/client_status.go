package gitlab

import (
	"fmt"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getGitlabStateFromStatus(s sdk.Status) gitlab.BuildState {
	switch s {
	case sdk.StatusWaiting:
		return gitlab.Pending
	case sdk.StatusChecking:
		return gitlab.Pending
	case sdk.StatusBuilding:
		return gitlab.Running
	case sdk.StatusSuccess:
		return gitlab.Success
	case sdk.StatusFail:
		return gitlab.Failed
	case sdk.StatusDisabled:
		return gitlab.Canceled
	case sdk.StatusNeverBuilt:
		return gitlab.Canceled
	case sdk.StatusUnknown:
		return gitlab.Failed
	case sdk.StatusSkipped:
		return gitlab.Canceled
	}

	return gitlab.Failed
}

//SetStatus set build status on Gitlab
func (c *gitlabClient) SetStatus(event sdk.Event) error {
	var eventpb sdk.EventPipelineBuild
	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		return err
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

	url := fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%d?envName=%s",
		c.uiURL,
		cdsProject,
		cdsApplication,
		cdsPipelineName,
		cdsBuildNumber,
		url.QueryEscape(cdsEnvironmentName),
	)

	desc := fmt.Sprintf("Build #%d %s", eventpb.BuildNumber, key)

	cds := "CDS"
	opt := &gitlab.SetCommitStatusOptions{
		Name:        &cds,
		Context:     &cds,
		State:       getGitlabStateFromStatus(eventpb.Status),
		Ref:         &eventpb.BranchName,
		TargetURL:   &url,
		Description: &desc,
	}

	if _, _, err := c.client.Commits.SetCommitStatus(eventpb.RepositoryFullname, eventpb.Hash, opt); err != nil {
		return err
	}

	return nil
}
