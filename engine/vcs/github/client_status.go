package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//SetStatus Users with push access can create commit statuses for a given ref:
//https://developer.github.com/v3/repos/statuses/#create-a-status
func (g *githubClient) SetStatus(event sdk.Event) error {
	log.Debug("github.SetStatus> receive: type:%s all: %+v", event.EventType, event)
	var eventpb sdk.EventPipelineBuild

	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	if g.DisableStatus {
		log.Warning("⚠ Github statuses are disabled")
		return nil
	}

	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		log.Warning("Error during consumption: %s", err)
		return err
	}

	log.Debug("Process event:%+v", event)

	//We only manage status Success and Failure
	if eventpb.Status == sdk.StatusChecking ||
		eventpb.Status == sdk.StatusDisabled ||
		eventpb.Status == sdk.StatusNeverBuilt ||
		eventpb.Status == sdk.StatusSkipped ||
		eventpb.Status == sdk.StatusUnknown ||
		eventpb.Status == sdk.StatusWaiting {
		return nil
	}

	var status string
	switch eventpb.Status {
	case sdk.StatusFail:
		status = "error"
	case sdk.StatusSuccess:
		status = "success"
	default:
		status = "pending"
	}

	url := fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%d?envName=%s",
		g.uiURL,
		eventpb.ProjectKey,
		eventpb.ApplicationName,
		eventpb.PipelineName,
		eventpb.BuildNumber,
		url.QueryEscape(eventpb.EnvironmentName),
	)

	//CDS can avoid sending github targer url in status, if it's disable
	if g.DisableStatusDetail {
		url = ""
	}

	var desc string
	switch eventpb.PipelineType {
	case sdk.BuildPipeline:
		desc = fmt.Sprintf("Build pipeline %s: %s", eventpb.PipelineName, eventpb.Status.String())
	case sdk.TestingPipeline:
		desc = fmt.Sprintf("Testing pipeline %s: %s", eventpb.PipelineName, eventpb.Status.String())
		if eventpb.Status == sdk.StatusFail {
			status = "failure"
		}
	case sdk.DeploymentPipeline:
		desc = fmt.Sprintf("Deployment pipeline %s: %s", eventpb.PipelineName, eventpb.Status.String())
	default:
		log.Warning("Unrecognized pipeline type : %v", eventpb.PipelineType)
		return nil
	}

	var context = fmt.Sprintf("continuous-delivery/CDS/%s", eventpb.PipelineName)

	ghStatus := CreateStatus{
		Description: desc,
		TargetURL:   url,
		State:       status,
		Context:     context,
	}

	path := fmt.Sprintf("/repos/%s/statuses/%s", eventpb.RepositoryFullname, eventpb.Hash)

	b, err := json.Marshal(ghStatus)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(b)

	res, err := g.post(path, "application/json", buf, false)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 201 {
		err := fmt.Errorf("Unable to create status on github. Status code : %d - Body: %s", res.StatusCode, body)
		log.Warning("SetStatus> %s", err)
		log.Warning("SetStatus> Sent data %s", b)
		return err
	}

	s := &Status{}
	if err := json.Unmarshal(body, s); err != nil {
		return err
	}

	log.Debug("SetStatus> Status %d %s created at %v", s.ID, s.URL, s.CreatedAt)

	return nil
}
