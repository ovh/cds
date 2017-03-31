package sdk

import (
	"encoding/json"
	"fmt"
)

var ConfigURLUIKey = "url.ui"

// GetConfigUser retrieve 'common' configuration CDS
func GetConfigUser() (map[string]string, error) {
	data, code, err := Request("GET", "/config/user", nil)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var output map[string]string
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}

	return output, nil
}
