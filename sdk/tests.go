package sdk

import (
	"encoding/json"
	"fmt"
)

// GetTestResults retrieves tests results for a specific build
func GetTestResults(proj, app, pip, env string, bn int) (Tests, error) {
	if env == "" {
		env = DefaultEnv.Name
	}
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/test?env=%s", proj, app, pip, bn, env)
	var t Tests

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return t, err
	}
	if code > 300 {
		return t, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal([]byte(data), &t)
	if err != nil {
		return t, err
	}

	return t, nil
}
