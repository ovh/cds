package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// ActionBuild represents an action to be run
type ActionBuild struct {
	ID               int64         `json:"id"`
	BuildNumber      int           `json:"build_number"`
	PipelineBuildID  int64         `json:"pipeline_build_id"`
	PipelineID       int64         `json:"pipeline_id"`
	ActionName       string        `json:"action_name"`
	PipelineActionID int64         `json:"pipeline_action_id"`
	PipelineStageID  int64         `json:"-"`
	Args             []Parameter   `json:"args"`
	Status           Status        `json:"status"`
	Requirements     []Requirement `json:"requirements"`
	Queued           time.Time     `json:"queued,omitempty"`
	Start            time.Time     `json:"start,omitempty"`
	Done             time.Time     `json:"done,omitempty"`
	Logs             string        `json:"logs,omitempty"`
	Model            string        `json:"model,omitempty"`
}

// BuildState define struct returned when looking for build state informations
type BuildState struct {
	Stages []Stage `json:"stages"`
	Logs   []Log   `json:"logs"`
	Status Status  `json:"status"`
}

// Status reprensents a Build Action or Build Pipeline Status
type Status string

// StatusFromString returns a Status from a given string
func StatusFromString(in string) Status {
	switch in {
	case StatusWaiting.String():
		return StatusWaiting
	case StatusBuilding.String():
		return StatusBuilding
	case StatusChecking.String():
		return StatusChecking
	case StatusSuccess.String():
		return StatusSuccess
	case StatusNeverBuilt.String():
		return StatusNeverBuilt
	case StatusFail.String():
		return StatusFail
	case StatusDisabled.String():
		return StatusDisabled
	case StatusSkipped.String():
		return StatusSkipped
	default:
		return StatusUnknown
	}
}

func (t Status) String() string {
	return string(t)
}

// Action status in queue
const (
	StatusWaiting    Status = "Waiting"
	StatusChecking   Status = "Checking"
	StatusBuilding   Status = "Building"
	StatusSuccess    Status = "Success"
	StatusFail       Status = "Fail"
	StatusDisabled   Status = "Disabled"
	StatusNeverBuilt Status = "Never Built"
	StatusUnknown    Status = "Unknown"
	StatusSkipped    Status = "Skipped"
)

// GetBuildQueue retrieves current CDS build in queue
func GetBuildQueue() ([]ActionBuild, error) {
	var q []ActionBuild

	path := "/queue?status=all"

	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &q)
	if err != nil {
		return nil, err
	}

	return q, nil
}

// GetBuildState Get the state of given build
func GetBuildState(projectKey, appName, pipelineName, env, buildID string) (PipelineBuild, error) {
	var buildState PipelineBuild

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s?envName=%s", projectKey, appName, pipelineName, buildID, env)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return buildState, err
	}
	if code >= 300 {
		return buildState, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &buildState)
	if err != nil {
		return buildState, err
	}
	return buildState, nil
}

// GetBuildActionLog Get the log of the given action for the given build
func GetBuildActionLog(projectKey, appName, pipelineName, buildID, pipelineActionID string) (BuildState, error) {
	var buildState BuildState

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/action/%s/log", projectKey, appName, pipelineName, buildID, pipelineActionID)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return buildState, err
	}
	if code >= 300 {
		return buildState, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &buildState)
	if err != nil {
		return buildState, err
	}
	return buildState, nil
}
