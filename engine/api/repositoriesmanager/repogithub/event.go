package repogithub

import (
	"fmt"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//SetStatus set build status on github
func (g *GithubClient) SetStatus(event sdk.Event) error {

	log.Debug("github.SetStatus> receive: type:%s all: %+v", event.EventType, event)
	var eventpb sdk.EventPipelineBuild

	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		log.Warning("Error during consumption: %s", err)
		return err
	}

	log.Debug("Process event:%+v", event)

	//We only manage status Success and Failure
	if eventpb.Status == sdk.StatusBuilding ||
		eventpb.Status == sdk.StatusChecking ||
		eventpb.Status == sdk.StatusDisabled ||
		eventpb.Status == sdk.StatusNeverBuilt ||
		eventpb.Status == sdk.StatusSkipped ||
		eventpb.Status == sdk.StatusUnknown ||
		eventpb.Status == sdk.StatusWaiting {
		return nil
	}

	var status string
	if eventpb.Status == sdk.StatusSuccess {
		status = "success"
	} else {
		status = "error"
	}

	// project/CDS/application/cds2tat/pipeline/monPipeline/build/855?env=monEnvi
	url := fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%d?env=%s",
		viper.GetString("base_url"),
		eventpb.ProjectKey,
		eventpb.ApplicationName,
		eventpb.PipelineName,
		eventpb.BuildNumber,
		url.QueryEscape(eventpb.EnvironmentName),
	)

	//CDS can avoid sending github targer url in status, if it's disable
	if viper.GetBool("no_github_target_url") {
		url = ""
	}

	var desc string
	switch eventpb.PipelineType {
	case sdk.BuildPipeline:
		desc = fmt.Sprintf("Build pipeline %s on %s: %s", eventpb.PipelineName, eventpb.ApplicationName, eventpb.Status.String())
	case sdk.TestingPipeline:
		desc = fmt.Sprintf("Testing pipeline %s on %s %s: %s", eventpb.PipelineName, eventpb.ApplicationName, eventpb.EnvironmentName, eventpb.Status.String())
		if eventpb.Status == sdk.StatusFail {
			status = "failure"
		}
	case sdk.DeploymentPipeline:
		desc = fmt.Sprintf("Deployment pipeline %s on %s %s: %s", eventpb.PipelineName, eventpb.ApplicationName, eventpb.EnvironmentName, eventpb.Status.String())
	default:
		log.Warning("Unrecognized pipeline type : %v", eventpb.PipelineType)
		return nil
	}

	var context = fmt.Sprintf("continuous-delivery/cds/%s/%s/%s/%s", eventpb.ProjectKey, eventpb.ApplicationName, eventpb.PipelineName, eventpb.EnvironmentName)

	ghStatus := CreateStatus{
		Description: desc,
		TargetURL:   url,
		State:       context,
		Context:     status,
	}

	return nil
}
