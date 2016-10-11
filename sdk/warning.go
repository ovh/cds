package sdk

import (
	"encoding/json"
	"fmt"
)

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
