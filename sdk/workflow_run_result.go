package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
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

type WorkflowRunResults []WorkflowRunResult

// Unique returns the last version of each results
func (w WorkflowRunResults) Unique() (WorkflowRunResults, error) {
	m := make(map[string]WorkflowRunResult, len(w))
	for i := range w {
		key, err := w[i].ComputeUniqueKey()
		if err != nil {
			return nil, err
		}
		if v, ok := m[key]; !ok || v.SubNum < w[i].SubNum {
			m[key] = w[i]
		}
	}
	filtered := make(WorkflowRunResults, 0, len(m))
	for _, v := range m {
		filtered = append(filtered, v)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Created.Before(filtered[j].Created) })
	return filtered, nil
}

type WorkflowRunResult struct {
	ID                string                 `json:"id" db:"id"`
	Created           time.Time              `json:"created" db:"created"`
	WorkflowRunID     int64                  `json:"workflow_run_id" db:"workflow_run_id"`
	WorkflowNodeRunID int64                  `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	WorkflowRunJobID  int64                  `json:"workflow_run_job_id" db:"workflow_run_job_id"`
	SubNum            int64                  `json:"sub_num" db:"sub_num"`
	Type              WorkflowRunResultType  `json:"type" db:"type"`
	DataRaw           json.RawMessage        `json:"data" db:"data"`
	DataSync          *WorkflowRunResultSync `json:"sync,omitmpty" db:"sync"`
}

type WorkflowRunResultSync struct {
	Sync       bool                         `json:"sync"`
	Link       string                       `json:"link"`
	Error      string                       `json:"error"`
	Promotions []WorkflowRunResultPromotion `json:"promotions"`
	Releases   []WorkflowRunResultPromotion `json:"releases"`
}

type WorkflowRunResultPromotionRequest struct {
	WorkflowRunResultPromotion
	IDs []string
}

type WorkflowRunResultPromotion struct {
	FromMaturity string    `json:"from_maturity"`
	ToMaturity   string    `json:"to_maturity"`
	Date         time.Time `json:"date"`
}

func (s *WorkflowRunResultSync) LatestPromotionOrRelease() *WorkflowRunResultPromotion {
	sort.Slice(s.Promotions, func(i, j int) bool {
		return s.Promotions[i].Date.Before(s.Promotions[j].Date)
	})
	sort.Slice(s.Releases, func(i, j int) bool {
		return s.Releases[i].Date.Before(s.Releases[j].Date)
	})
	if len(s.Releases) > 0 {
		return &s.Releases[len(s.Releases)-1]
	}
	if len(s.Promotions) > 0 {
		return &s.Promotions[len(s.Promotions)-1]
	}
	return nil
}

// Value returns driver.Value from WorkflowRunResultSync
func (s WorkflowRunResultSync) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal WorkflowRunResultSync")
}

// Scan WorkflowRunResultSync
func (s *WorkflowRunResultSync) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, s), "cannot unmarshal WorkflowRunResultSync")
}

func (r WorkflowRunResult) ComputeUniqueKey() (string, error) {
	key := fmt.Sprintf("%d-%s", r.WorkflowRunID, r.Type)
	switch r.Type {
	case WorkflowRunResultTypeArtifactManager:
		var data WorkflowRunResultArtifactManager
		if err := json.Unmarshal(r.DataRaw, &data); err != nil {
			return "", WithStack(err)
		}
		key = key + "-" + data.Name + "-" + data.RepoType
	default:
		var data WorkflowRunResultArtifactCommon
		if err := json.Unmarshal(r.DataRaw, &data); err != nil {
			return "", WithStack(err)
		}
		key = key + "-" + data.Name
	}
	return key, nil
}

func (r WorkflowRunResult) ComputeName() (string, error) {
	switch r.Type {
	case WorkflowRunResultTypeArtifactManager:
		var data WorkflowRunResultArtifactManager
		if err := json.Unmarshal(r.DataRaw, &data); err != nil {
			return "", WithStack(err)
		}
		return fmt.Sprintf("%s (%s: %s)", data.Name, r.Type, data.RepoType), nil
	default:
		var data WorkflowRunResultArtifactCommon
		if err := json.Unmarshal(r.DataRaw, &data); err != nil {
			return "", WithStack(err)
		}
		return fmt.Sprintf("%s (%s)", data.Name, r.Type), nil
	}
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

type WorkflowRunResultArtifactCommon struct {
	Name string `json:"name"`
}

type WorkflowRunResultArtifactManager struct {
	WorkflowRunResultArtifactCommon
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
	WorkflowRunResultArtifactCommon
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
	WorkflowRunResultArtifactCommon
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
	WorkflowRunResultArtifactCommon
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
