package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// Token describes tokens used by worker to access the API
// on behalf of a group.
type Token struct {
	GroupID    int64      `json:"group_id"`
	Token      string     `json:"token" cli:"token"`
	Expiration Expiration `json:"expiration" cli:"expiration"`
	Created    time.Time  `json:"created" cli:"created"`
}

// GenerateWorkerToken creates a key tied to calling user that allow registering workers
func GenerateWorkerToken(group string, e Expiration) (*Token, error) {
	path := fmt.Sprintf("/group/%s/token/%s", group, e)
	data, code, err := Request("POST", path, nil)
	if err != nil {
		return nil, err
	}
	if code > 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	tk := &Token{}
	if err = json.Unmarshal(data, &tk); err != nil {
		return nil, err
	}

	return tk, nil
}
