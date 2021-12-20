package sdk

import (
	"encoding/json"
	"time"
)

const (
	WorkflowRunResultTypeArtifact        WorkflowRunResultType = "artifact"
	WorkflowRunResultTypeCoverage        WorkflowRunResultType = "coverage"
	WorkflowRunResultTypeArtifactManager WorkflowRunResultType = "artifact-manager"
	WorkflowRunResultTypeStaticFile      WorkflowRunResultType = "static-file"
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
	if err := JSONUnmarshal(r.DataRaw, &data); err != nil {
		return data, WithStack(err)
	}

	return data, nil
}

func (r *WorkflowRunResult) GetCoverage() (WorkflowRunResultCoverage, error) {
	var data WorkflowRunResultCoverage
	if err := JSONUnmarshal(r.DataRaw, &data); err != nil {
		return data, WithStack(err)
	}
	return data, nil
}

func (r *WorkflowRunResult) GetArtifactManager() (WorkflowRunResultArtifactManager, error) {
	var data WorkflowRunResultArtifactManager
	if err := JSONUnmarshal(r.DataRaw, &data); err != nil {
		return data, WithStack(err)
	}
	if data.FileType == "" {
		data.FileType = data.RepoType
	}
	return data, nil
}

func (r *WorkflowRunResult) GetStaticFile() (WorkflowRunResultStaticFile, error) {
	var data WorkflowRunResultStaticFile
	if err := JSONUnmarshal(r.DataRaw, &data); err != nil {
		return data, WithStack(err)
	}
	return data, nil
}

type WorkflowRunResultCheck struct {
	Name       string                `json:"name"`
	RunID      int64                 `json:"run_id"`
	RunNodeID  int64                 `json:"run_node_id"`
	RunJobID   int64                 `json:"run_job_id"`
	ResultType WorkflowRunResultType `json:"result_type"`
}

type WorkflowRunResultArtifactManager struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MD5      string `json:"md5"`
	Path     string `json:"path"`
	Perm     uint32 `json:"perm"`
	RepoName string `json:"repository_name"`
	RepoType string `json:"repository_type"`
	FileType string `json:"file_type"`
}

func (a *WorkflowRunResultArtifactManager) IsValid() error {
	if a.Name == "" {
		return WrapError(ErrInvalidData, "missing artifact name")
	}
	if a.Path == "" {
		return WrapError(ErrInvalidData, "missing cdn item hash")
	}
	if a.RepoName == "" {
		return WrapError(ErrInvalidData, "missing repository_name")
	}
	if a.RepoType == "" {
		return WrapError(ErrInvalidData, "missing repository_type")
	}
	return nil
}

type WorkflowRunResultStaticFile struct {
	Name      string `json:"name"`
	RemoteURL string `json:"remote_url"`
}

func (a *WorkflowRunResultStaticFile) IsValid() error {
	if a.Name == "" {
		return WrapError(ErrInvalidData, "missing static-file name")
	}
	if a.RemoteURL == "" {
		return WrapError(ErrInvalidData, "missing remote url")
	}
	return nil
}

type WorkflowRunResultArtifact struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	MD5        string `json:"md5"`
	CDNRefHash string `json:"cdn_hash"`
	Perm       uint32 `json:"perm"`
	FileType   string `json:"file_type"`
}

func (a *WorkflowRunResultArtifact) IsValid() error {
	if a.Name == "" {
		return WrapError(ErrInvalidData, "missing artifact name")
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
	Perm       uint32 `json:"perm"`
}

func (a *WorkflowRunResultCoverage) IsValid() error {
	if a.Name == "" {
		return WrapError(ErrInvalidData, "missing file name")
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
