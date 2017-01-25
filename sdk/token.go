package sdk

import (
	"encoding/json"
	"fmt"
)

// GenerateWorkerToken creates a key tied to calling user that allow registering workers
func GenerateWorkerToken(group string, e Expiration) (string, error) {

	path := fmt.Sprintf("/group/%s/token/%s", group, e)
	data, code, err := Request("POST", path, nil)
	if err != nil {
		return "", err
	}
	if code > 300 {
		return "", fmt.Errorf("HTTP %d", code)
	}

	s := struct {
		Key string `json:"key"`
	}{}

	err = json.Unmarshal(data, &s)
	if err != nil {
		return "", err
	}

	return s.Key, nil
}
