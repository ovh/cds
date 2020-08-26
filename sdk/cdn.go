package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
	"strconv"

	"github.com/mitchellh/hashstructure"
)

type CDNAuthToken struct {
	APIRefHash string `json:"api_ref_hash"`
}

type CDNLogAccess struct {
	Exists      bool   `json:"exists"`
	Token       string `json:"token,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

type CDNLogAPIRef struct {
	ProjectKey     string `json:"project_key"`
	WorkflowName   string `json:"workflow_name"`
	WorkflowID     int64  `json:"workflow_id"`
	RunID          int64  `json:"run_id"`
	NodeRunID      int64  `json:"node_run_id"`
	NodeRunName    string `json:"node_run_name"`
	NodeRunJobID   int64  `json:"node_run_job_id"`
	NodeRunJobName string `json:"node_run_job_name"`

	// for workers
	StepOrder int64  `json:"step_order"`
	StepName  string `json:"step_name,omitempty"`

	// for hatcheries
	RequirementServiceID   int64  `json:"service_id,omitempty"`
	RequirementServiceName string `json:"service_name,omitempty"`
}

func (a CDNLogAPIRef) ToHash() (string, error) {
	hashRefU, err := hashstructure.Hash(a, nil)
	if err != nil {
		return "", WithStack(err)
	}
	return strconv.FormatUint(hashRefU, 10), nil
}

// Value returns driver.Value from CDNLogAPIRef.
func (a CDNLogAPIRef) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, WrapError(err, "cannot marshal CDNLogAPIRef")
}

// Scan CDNLogAPIRef.
func (a *CDNLogAPIRef) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, a), "cannot unmarshal CDNLogAPIRef")
}
