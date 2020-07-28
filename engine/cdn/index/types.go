package index

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"time"
)

var (
	TypeItemStepLog    = "StepLog"
	TypeItemServiceLog = "ServiceLog"

	StatusItemIncoming  = "Incoming"
	StatusItemCompleted = "Completed"
)

type Item struct {
	gorpmapper.SignedEntity
	ID           string    `json:"id" db:"id"`
	Created      time.Time `json:"created" db:"created"`
	LastModified time.Time `json:"last_modified" db:"last_modified"`
	Hash         string    `json:"-" db:"hash"`
	ApiRef       ApiRef    `json:"api_ref" db:"api_ref"`
	ApiRefHash   string    `json:"api_ref_hash" db:"api_ref_hash"`
	Status       string    `json:"status" db:"status"`
	Type         string    `json:"type" db:"type"`
}

type ApiRef struct {
	ProjectKey     string `json:"project_key,omitempty"`
	WorkflowName   string `json:"workflow_name,omitempty"`
	WorkflowID     int64  `json:"workflow_id,omitempty"`
	RunID          int64  `json:"run_id,omitempty"`
	NodeRunID      int64  `json:"node_run_id,omitempty"`
	NodeRunName    string `json:"node_run_name,omitempty"`
	NodeRunJobID   int64  `json:"node_run_job_id,omitempty"`
	NodeRunJobName string `json:"node_run_job_name,omitempty"`
	StepOrder      int64  `json:"step_order,omitempty"`
	StepName       string `json:"step_name,omitempty"`
}

// Value returns driver.Value from ApiRef.
func (a ApiRef) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, sdk.WrapError(err, "cannot marshal ApiRef")
}

// Scan ApiRef.
func (a *ApiRef) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return sdk.WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return sdk.WrapError(json.Unmarshal(source, a), "cannot unmarshal ApiRef")
}