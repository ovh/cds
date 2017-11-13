package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Stage Pipeline step that parallelize actions by order
type Stage struct {
	ID                int64                  `json:"id" yaml:"pipeline_stage_id"`
	Name              string                 `json:"name"`
	PipelineID        int64                  `json:"-" yaml:"-"`
	BuildOrder        int                    `json:"build_order"`
	Enabled           bool                   `json:"enabled"`
	PipelineBuildJobs []PipelineBuildJob     `json:"builds"`
	RunJobs           []WorkflowNodeJobRun   `json:"run_jobs"`
	Prerequisites     []Prerequisite         `json:"prerequisites"`
	LastModified      int64                  `json:"last_modified"`
	Jobs              []Job                  `json:"jobs"`
	Status            Status                 `json:"status"`
	Warnings          []PipelineBuildWarning `json:"warnings"`
}

// Conditions returns stage prerequisites as a set of WorkflowTriggerCondition regex
func (s *Stage) Conditions() []WorkflowNodeCondition {
	res := []WorkflowNodeCondition{}
	for _, p := range s.Prerequisites {
		res = append(res, WorkflowNodeCondition{
			Value:    p.ExpectedValue,
			Variable: p.Parameter,
			Operator: WorkflowConditionsOperatorRegex,
		})
	}
	return res
}

// NewStage instanciate a new Stage
func NewStage(name string) *Stage {
	s := &Stage{
		Name: name,
	}
	return s
}

// AddStage creates a new stage
func AddStage(projectKey, pipelineName, name string) error {

	s := NewStage(name)

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/pipeline/%s/stage", projectKey, pipelineName)
	data, code, err := Request("POST", url, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}

	return nil
}

// GetStage Get stage by ID
func GetStage(projectKey, pipelineName, pipelineStageID string) (*Stage, error) {
	s := &Stage{}
	url := fmt.Sprintf("/project/%s/pipeline/%s/stage/%s", projectKey, pipelineName, pipelineStageID)
	data, code, err := Request("GET", url, nil)
	if err != nil {
		return s, err
	}
	if code != http.StatusCreated && code != http.StatusOK {
		return s, fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return s, e
	}
	err = json.Unmarshal(data, s)
	return s, err
}

func updateStage(projectKey, pipelineName, pipelineStageID string, stageData *Stage) error {
	data, err := json.Marshal(stageData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/pipeline/%s/stage/%s", projectKey, pipelineName, pipelineStageID)
	data, code, err := Request("PUT", url, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}

// RenameStage Rename a stage
func RenameStage(projectKey, pipelineName, pipelineStageID, newName string) error {

	s, err := GetStage(projectKey, pipelineName, pipelineStageID)
	if err != nil {
		return err
	}
	s.Name = newName
	return updateStage(projectKey, pipelineName, pipelineStageID, s)
}

// ChangeStageState Enabled/Disabled a stage
func ChangeStageState(projectKey, pipelineName, pipelineStageID string, enabled bool) error {

	s, err := GetStage(projectKey, pipelineName, pipelineStageID)
	if err != nil {
		return err
	}
	s.Enabled = enabled
	return updateStage(projectKey, pipelineName, pipelineStageID, s)
}

// MoveStage Change stage buildOrder
func MoveStage(projectKey, pipelineName string, pipelineStageID int64, buildOrder int) error {

	s := &Stage{
		ID:         pipelineStageID,
		BuildOrder: buildOrder,
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/pipeline/%s/stage/move", projectKey, pipelineName)
	data, code, err := Request("POST", url, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}

	return nil
}

// DeleteStage Call API to delete the given stage from the given pipeline
func DeleteStage(projectKey, pipelineName, pipelineStageID string) error {

	url := fmt.Sprintf("/project/%s/pipeline/%s/stage/%s", projectKey, pipelineName, pipelineStageID)
	data, code, err := Request("DELETE", url, nil)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}

	return nil
}
