package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// WarningV2 Represents warning database structure
type WarningV2 struct {
	ID            int64             `json:"id" db:"id"`
	Key           string            `json:"key" db:"project_key"`
	AppName       string            `json:"application_name" db:"application_name"`
	PipName       string            `json:"pipeline_name" db:"pipeline_name"`
	WorkflowName  string            `json:"workflow_name" db:"workflow_name"`
	EnvName       string            `json:"environment_name" db:"environment_name"`
	Type          string            `json:"type" db:"type"`
	Element       string            `json:"element" db:"element"`
	Created       time.Time         `json:"created" db:"created"`
	MessageParams map[string]string `json:"message_params" db:"-"`
	Message       string            `json:"message" db:"-"`
}

// Warning contains information about user action configuration
type Warning struct {
	ID           int64             `json:"id"`
	Message      string            `json:"message"`
	MessageParam map[string]string `json:"message_param"`

	Action      Action      `json:"action"`
	StageID     int64       `json:"stage_id"`
	Project     Project     `json:"project"`
	Application Application `json:"application"`
	Pipeline    Pipeline    `json:"pipeline"`
	Environment Environment `json:"environment"`
}

// GetWarnings retrieves warnings related to Action accessible to caller
func GetWarnings() ([]Warning, error) {
	uri := "/mon/warning"

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	if code > 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var warnings []Warning
	err = json.Unmarshal(data, &warnings)
	if err != nil {
		return nil, err
	}

	return warnings, nil
}
