package sdk

import (
	"encoding/json"
	"fmt"
)

// GetStatus retrieve generic health infos to CDS
func GetStatus() ([]string, error) {
	var output []string

	data, code, err := Request("GET", "/mon/status", nil)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &output)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// GetVersion returns API version
func GetVersion() (string, error) {
	data, code, err := Request("GET", "/mon/version", nil)
	if err != nil {
		return "", err
	}

	if code >= 300 {
		return "", fmt.Errorf("HTTP %d", code)
	}

	s := struct {
		Version string `json:"version"`
	}{}

	err = json.Unmarshal(data, &s)
	if err != nil {
		return "", err
	}

	return s.Version, nil
}
