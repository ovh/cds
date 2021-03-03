package sdk

import (
	"encoding/json"
	"time"
)

const (
	WorkflowRunResultTypeArtifact WorkflowRunResultType = "artifact"
	WorkflowRunResultTypeCoverage WorkflowRunResultType = "coverage"
)

type WorkflowRunResultType string
type WorkflowRunResultDataKey string

type WorkflowRunResult struct {
	ID                string                `json:"id" db:"id"`
	Created           time.Time             `json:"created" db:"created"`
	WorkflowRunID     int64                 `json:"workflow_run_id" db:"workflow_run_id"`
	WorkflowNodeRunID int64                 `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	WorkflowRunJobID  int64                 `json:"workflow_run_job_id" db:"workflow_run_job_id"`
	SubNum            int64                 `json:"sub_num" db:"sub_num"`
	Type              WorkflowRunResultType `json:"type" db:"type"`
	DataRaw           json.RawMessage       `json:"data" db:"data"`
}

func (r *WorkflowRunResult) GetArtifact() (WorkflowRunResultArtifact, error) {
	var data WorkflowRunResultArtifact
	if err := json.Unmarshal(r.DataRaw, &data); err != nil {
		return data, WithStack(err)
	}
	return data, nil
}

func (r *WorkflowRunResult) GetCoverage() (WorkflowRunResultCoverage, error) {
	var data WorkflowRunResultCoverage
	if err := json.Unmarshal(r.DataRaw, &data); err != nil {
		return data, WithStack(err)
	}
	return data, nil
}

type WorkflowRunResultArtifact struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	MD5        string `json:"md5"`
	CDNRefHash string `json:"cdn_hash"`
	Perm       uint32 `json:"perm"`
}

func (a *WorkflowRunResultArtifact) IsValid() error {
	if a.Name == "" {
		return WrapError(ErrInvalidData, "missing artifact name")
	}
	if a.Size == 0 {
		return WrapError(ErrInvalidData, "missing artifact size")
	}
	if a.MD5 == "" {
		return WrapError(ErrInvalidData, "missing md5Sum")
	}
	if a.CDNRefHash == "" {
		return WrapError(ErrInvalidData, "missing cdn item hash")
	}
	if a.Perm == 0 {
		return WrapError(ErrInvalidData, "missing file permission")
	}
	return nil
}

type WorkflowRunResultCoverage struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	MD5        string `json:"md5"`
	CDNRefHash string `json:"cdn_hash"`
}

func (a *WorkflowRunResultCoverage) IsValid() error {
	if a.Name == "" {
		return WrapError(ErrInvalidData, "missing file name")
	}
	if a.Size == 0 {
		return WrapError(ErrInvalidData, "missing file size")
	}
	if a.MD5 == "" {
		return WrapError(ErrInvalidData, "missing md5Sum")
	}
	if a.CDNRefHash == "" {
		return WrapError(ErrInvalidData, "missing cdn item hash")
	}
	return nil
}
