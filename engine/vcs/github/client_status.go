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

type statusData struct {
	pipName      string
	desc         string
	status       string
	repoFullName string
	hash         string
	urlPipeline  string
}

//SetStatus Users with push access can create commit statuses for a given ref:
//https://developer.github.com/v3/repos/statuses/#create-a-status
func (g *githubClient) SetStatus(event sdk.Event) error {
	log.Debug("github.SetStatus> receive: type:%s all: %+v", event.EventType, event)

	if g.DisableStatus {
		log.Warning("github.SetStatus>  ⚠ Github statuses are disabled")
		return nil
	}

	var data statusData
	var err error
	switch event.EventType {
	case fmt.Sprintf("%T", sdk.EventPipelineBuild{}):
		data, err = processEventPipelineBuild(event, g.uiURL, g.DisableStatusDetail)
	case fmt.Sprintf("%T", sdk.EventWorkflowNodeRun{}):
		data, err = processEventWorkflowNodeRun(event, g.uiURL, g.DisableStatusDetail)
	default:
		log.Error("github.SetStatus> Unknown event %s", event)
		return nil
	}
	if err != nil {
		return sdk.WrapError(err, "githubClient.SetStatus> Cannot process Event")
	}

	if data.status == "" {
		log.Debug("github.SetStatus> Do not process event for current status: %s", event)
		return nil
	}

	var context = fmt.Sprintf("continuous-delivery/CDS/%s", data.pipName)

	ghStatus := CreateStatus{
		Description: data.desc,
		TargetURL:   data.urlPipeline,
		State:       data.status,
		Context:     context,
	}

	path := fmt.Sprintf("/repos/%s/statuses/%s", data.repoFullName, data.hash)

	b, err := json.Marshal(ghStatus)
	if err != nil {
		return sdk.WrapError(err, "github.SetStatus> Unable to marshal github status")
	}
	buf := bytes.NewBuffer(b)

	res, err := g.post(path, "application/json", buf, false)
	if err != nil {
		return sdk.WrapError(err, "github.SetStatus> Unable to post status")
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return sdk.WrapError(err, "github.SetStatus> Unable to read body")
	}

	if res.StatusCode != 201 {
		return sdk.WrapError(err, "github.SetStatus>  Unable to create status on github. Status code : %d - Body: %s", res.StatusCode, body)
	}

	s := &Status{}
	if err := json.Unmarshal(body, s); err != nil {
		return sdk.WrapError(err, "github.SetStatus> Unable to unmarshal body")
	}

	log.Debug("SetStatus> Status %d %s created at %v", s.ID, s.URL, s.CreatedAt)

	return nil
}

func processEventWorkflowNodeRun(event sdk.Event, githubURL string, disabledStatusDetail bool) (statusData, error) {
	data := statusData{}
	var eventNR sdk.EventWorkflowNodeRun
	if err := mapstructure.Decode(event.Payload, &eventNR); err != nil {
		return data, sdk.WrapError(err, "githubClient.processEventWorkflowNodeRun> Error durring consumption")
	}

	log.Debug("Process event:%+v", event)
	//We only manage status Success and Failure
	if eventNR.Status == sdk.StatusChecking.String() ||
		eventNR.Status == sdk.StatusDisabled.String() ||
		eventNR.Status == sdk.StatusNeverBuilt.String() ||
		eventNR.Status == sdk.StatusSkipped.String() ||
		eventNR.Status == sdk.StatusUnknown.String() ||
		eventNR.Status == sdk.StatusWaiting.String() {
		return data, nil
	}

	switch eventNR.Status {
	case sdk.StatusFail.String():
		data.status = "error"
	case sdk.StatusSuccess.String():
		data.status = "success"
	default:
		data.status = "pending"
	}
	data.hash = eventNR.Hash
	data.repoFullName = eventNR.RepositoryFullName
	data.pipName = eventNR.NodeName

	data.urlPipeline = fmt.Sprintf("%s/project/%s/workflow/%s/run/%d",
		githubURL,
		eventNR.ProjectKey,
		eventNR.WorkflowName,
		eventNR.Number,
	)

	//CDS can avoid sending github targer url in status, if it's disable
	if disabledStatusDetail {
		data.urlPipeline = ""
	}

	data.desc = fmt.Sprintf("Pipeline %s: %s", eventNR.PipelineName, eventNR.Status)
	return data, nil
}

func processEventPipelineBuild(event sdk.Event, githubURL string, disabledStatusDetail bool) (statusData, error) {
	data := statusData{}
	var eventpb sdk.EventPipelineBuild
	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		return data, sdk.WrapError(err, "githubClient.processEventPipelineBuild> Error durring consumption")
	}

	log.Debug("Process event:%+v", event)
	//We only manage status Success and Failure
	if eventpb.Status == sdk.StatusChecking ||
		eventpb.Status == sdk.StatusDisabled ||
		eventpb.Status == sdk.StatusNeverBuilt ||
		eventpb.Status == sdk.StatusSkipped ||
		eventpb.Status == sdk.StatusUnknown ||
		eventpb.Status == sdk.StatusWaiting {
		return data, nil
	}

	switch eventpb.Status {
	case sdk.StatusFail:
		data.status = "error"
	case sdk.StatusSuccess:
		data.status = "success"
	default:
		data.status = "pending"
	}
	data.urlPipeline = fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%d?envName=%s",
		githubURL,
		eventpb.ProjectKey,
		eventpb.ApplicationName,
		eventpb.PipelineName,
		eventpb.BuildNumber,
		url.QueryEscape(eventpb.EnvironmentName),
	)
	data.hash = eventpb.Hash
	data.repoFullName = eventpb.RepositoryFullname
	//CDS can avoid sending github targer url in status, if it's disable
	if disabledStatusDetail {
		data.urlPipeline = ""
	}

	switch eventpb.PipelineType {
	case sdk.BuildPipeline:
		data.desc = fmt.Sprintf("Build pipeline %s: %s", eventpb.PipelineName, eventpb.Status.String())
	case sdk.TestingPipeline:
		data.desc = fmt.Sprintf("Testing pipeline %s: %s", eventpb.PipelineName, eventpb.Status.String())
		if eventpb.Status == sdk.StatusFail {
			data.status = "failure"
		}
	case sdk.DeploymentPipeline:
		data.desc = fmt.Sprintf("Deployment pipeline %s: %s", eventpb.PipelineName, eventpb.Status.String())
	default:
		log.Warning("Unrecognized pipeline type : %v", eventpb.PipelineType)
	}

	return data, nil
}
