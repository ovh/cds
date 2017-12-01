package sdk

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Prerequisite defines a expected value to one triggering pipeline parameter
type Prerequisite struct {
	Parameter     string `json:"parameter"`
	ExpectedValue string `json:"expected_value"`
}

// PipelineTrigger represent a pipeline trigger
type PipelineTrigger struct {
	ID int64 `json:"id"`

	SrcProject     Project     `json:"src_project" yaml:"-"`
	SrcApplication Application `json:"src_application" yaml:"-"`
	SrcPipeline    Pipeline    `json:"src_pipeline" yaml:"-"`
	SrcEnvironment Environment `json:"src_environment" yaml:"-"`

	DestProject     Project     `json:"dest_project" yaml:"-"`
	DestApplication Application `json:"dest_application" yaml:"-"`
	DestPipeline    Pipeline    `json:"dest_pipeline" yaml:"-"`
	DestEnvironment Environment `json:"dest_environment" yaml:"-"`

	Manual        bool           `json:"manual"`
	Parameters    []Parameter    `json:"parameters"`
	Prerequisites []Prerequisite `json:"prerequisites"`
	LastModified  int64          `json:"last_modified"`
}

// GetTriggers retrieves all output triggers of a pipeline
func GetTriggers(project, app, pipeline, env string) ([]PipelineTrigger, error) {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger", project, app, pipeline)

	if env != "" {
		uri = fmt.Sprintf("%s?env=%s", uri, env)
	}

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var triggers []PipelineTrigger
	err = json.Unmarshal(data, &triggers)
	if err != nil {
		return nil, err
	}

	return triggers, nil
}

// GetTriggersAsSource retrieves all output triggers of a pipeline
func GetTriggersAsSource(project, app, pipeline, env string) ([]PipelineTrigger, error) {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger/source", project, app, pipeline)

	if env != "" {
		uri = fmt.Sprintf("%s?env=%s", uri, env)
	}

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var triggers []PipelineTrigger
	err = json.Unmarshal(data, &triggers)
	if err != nil {
		return nil, err
	}

	return triggers, nil
}

// AddTrigger adds a trigger between two pipelines
func AddTrigger(t *PipelineTrigger) error {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name)

	if t.SrcEnvironment.Name != "" {
		uri = fmt.Sprintf("%s?env=%s", uri, t.SrcEnvironment.Name)
	}

	fmt.Printf("Adding trigger %s/%s/%s[%s] -> %s/%s/%s[%s]\n",
		t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name,
		t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name)

	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	data, code, err := Request("POST", uri, data)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	srcApp := &Application{}
	if err := json.Unmarshal(data, srcApp); err != nil {
		return err
	}

	return nil
}

// GetTrigger gets an existing trigger
func GetTrigger(proj, app, pip string, id int64) (*PipelineTrigger, error) {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger/%d", proj, app, pip, id)

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var trigger *PipelineTrigger
	err = json.Unmarshal(data, &trigger)
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

// DeleteTrigger removes a trigger between two pipelines
func DeleteTrigger(proj, app, pip string, id int64) error {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger/%d", proj, app, pip, id)

	_, code, err := Request("DELETE", uri, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// UpdateTrigger adds a trigger between two pipelines
func UpdateTrigger(t *PipelineTrigger) error {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger/%d", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.ID)
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	_, code, err := Request("PUT", uri, data)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// NewPrerequisite creates a Prerequisite from a string with <name>=<expectedValue> format
func NewPrerequisite(s string) (Prerequisite, error) {
	var p Prerequisite

	t := strings.SplitN(s, "=", 2)
	if len(t) != 2 {
		return p, fmt.Errorf("cds: wrong format parameter")
	}
	p.Parameter = t[0]
	p.ExpectedValue = t[1]

	return p, nil
}
