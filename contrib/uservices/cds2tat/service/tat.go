package main

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/tat"
)

const (
	failed   = "#a94442"
	success  = "#3c763d"
	waiting  = "#8a6d3b"
	building = "#31708f"
	unknown  = "#333"
)

// Process send message to all notifications backend
func process(event sdk.Event) error {

	log.Debugf("process> receive: type:%s all: %+v", event.EventType, event)

	if event.EventType == fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		var e sdk.EventPipelineBuild
		if err := mapstructure.Decode(event.Payload, &e); err != nil {
			log.Errorf("Error during consumption EventPipelineBuild: %s", err)
		} else {
			return processEventPipelineBuild(&e)
		}
	} else if event.EventType == fmt.Sprintf("%T", sdk.EventJob{}) {
		var e sdk.EventJob
		if err := mapstructure.Decode(event.Payload, &e); err != nil {
			log.Errorf("Error during consumption EventJob: %s", err)
		} else {
			return processEventJob(&e)
		}
	}
	return nil
}

func processEventPipelineBuild(e *sdk.EventPipelineBuild) error {
	eventType := "pipelineBuild"
	cdsProject := e.ProjectKey
	cdsApp := e.ApplicationName
	cdsPipeline := e.PipelineName
	cdsEnvironment := e.EnvironmentName
	version := e.Version
	branch := e.BranchName

	return processMsg(eventType, cdsProject, cdsApp, cdsPipeline, cdsEnvironment, version, branch, e.Status)
}

func processEventJob(e *sdk.EventJob) error {
	eventType := "job"
	cdsProject := e.ProjectKey
	cdsApp := e.ApplicationName
	cdsPipeline := e.PipelineName
	cdsEnvironment := e.EnvironmentName
	version := e.Version
	branch := e.BranchName

	return processMsg(eventType, cdsProject, cdsApp, cdsPipeline, cdsEnvironment, version, branch, e.Status)
}

func processMsg(eventType, cdsProject, cdsApp, cdsPipeline, cdsEnvironment string, version int64, branch string, cdsStatus sdk.Status) error {

	text := fmt.Sprintf("#cds #type:%s #project:%s #app:%s #pipeline:%s #environment:%s #version:%d #branch:%s",
		eventType, cdsProject, cdsApp, cdsPipeline, cdsEnvironment, version, branch)

	tagsReference := fmt.Sprintf("cds,project:%s,app:%s,pipeline:%s,environment:%s,version:%d,branch:%s",
		cdsProject, cdsApp, cdsPipeline, cdsEnvironment, version, branch)

	msg := tat.MessageJSON{
		Text:         text,
		Labels:       getLabelsFromStatus(cdsStatus),
		TagReference: tagsReference,
		Topic:        viper.GetString("topic_tat_engine"),
	}

	if _, err := getClient().MessageRelabelOrCreate(msg); err != nil {
		return fmt.Errorf("Error while MessageAdd:%s", err)
	}

	return nil
}

func getLabelsFromStatus(status sdk.Status) []tat.Label {
	switch status {
	case sdk.StatusSuccess:
		return []tat.Label{tat.Label{Text: status.String(), Color: success}}
	case sdk.StatusWaiting:
		return []tat.Label{tat.Label{Text: status.String(), Color: waiting}}
	case sdk.StatusBuilding:
		return []tat.Label{tat.Label{Text: status.String(), Color: building}}
	case sdk.StatusFail:
		return []tat.Label{tat.Label{Text: status.String(), Color: failed}}
	default:
		return []tat.Label{tat.Label{Text: status.String(), Color: unknown}}
	}
}
