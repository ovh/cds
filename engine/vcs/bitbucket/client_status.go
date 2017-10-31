package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (b *bitbucketClient) SetStatus(event sdk.Event) error {
	log.Info("process> receive: type:%s all: %+v", event.EventType, event)
	var eventpb sdk.EventPipelineBuild

	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		return sdk.WrapError(err, "Error during consumption")
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
		b.consumer.uiURL,
		cdsProject,
		cdsApplication,
		cdsPipelineName,
		cdsBuildNumber,
		url.QueryEscape(cdsEnvironmentName),
	)

	status := Status{
		Key:   key,
		Name:  fmt.Sprintf("%s%d", key, cdsBuildNumber),
		State: getBitbucketStateFromStatus(eventpb.Status),
		URL:   url,
	}

	log.Debug("SetStatus> hash:%s status:%+v", eventpb.Hash, status)

	values, err := json.Marshal(status)
	if err != nil {
		return sdk.WrapError(err, "VCS> Bitbucket> Unable to marshall status")
	}
	return b.do("POST", "build-status", fmt.Sprintf("/commits/%s", eventpb.Hash), nil, values, nil)
}

const (
	inProgress = "INPROGRESS"
	successful = "SUCCESSFUL"
	failed     = "FAILED"
)

func getBitbucketStateFromStatus(status sdk.Status) string {
	switch status {
	case sdk.StatusSuccess:
		return successful
	case sdk.StatusWaiting:
		return inProgress
	case sdk.StatusBuilding:
		return inProgress
	case sdk.StatusFail:
		return failed
	default:
		return failed
	}
}
