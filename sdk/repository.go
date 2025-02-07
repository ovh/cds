package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const (
	RepositoryAnalysisStatusInProgress = "InProgress"
	RepositoryAnalysisStatusSucceed    = "Success"
	RepositoryAnalysisStatusError      = "Error"
	RepositoryAnalysisStatusSkipped    = "Skipped"

	GitRefTagPrefix    = "refs/tags/"
	GitRefBranchPrefix = "refs/heads/"
	GitRefTypeBranch   = "branch"
	GitRefTypeTag      = "tag"
)

type ProjectRepository struct {
	ID           string    `json:"id" db:"id"`
	ProjectKey   string    `json:"project_key" db:"project_key"`
	Name         string    `json:"name" db:"name" cli:"name,key" action_metadata_name:"name"`
	Created      time.Time `json:"created" db:"created"`
	CreatedBy    string    `json:"created_by" db:"created_by"`
	VCSProjectID string    `json:"-" db:"vcs_project_id"`
	CloneURL     string    `json:"clone_url" db:"clone_url"`
}

type ProjectRepositoryAnalysis struct {
	ID                  string                `json:"id" db:"id" cli:"id"`
	Created             time.Time             `json:"created" db:"created" cli:"created"`
	LastModified        time.Time             `json:"last_modified" db:"last_modified"`
	ProjectRepositoryID string                `json:"project_repository_id" db:"project_repository_id"`
	VCSProjectID        string                `json:"vcs_project_id" db:"vcs_project_id"`
	ProjectKey          string                `json:"project_key" db:"project_key"`
	Status              string                `json:"status" db:"status" cli:"status"`
	Ref                 string                `json:"ref" db:"ref" cli:"ref"`
	Commit              string                `json:"commit" db:"commit" cli:"commit"`
	Data                ProjectRepositoryData `json:"data" db:"data"`
}

type ProjectRepositoryData struct {
	HookEventUUID             string                        `json:"hook_event_uuid"`
	HookEventKey              string                        `json:"hook_event_key"`
	OperationUUID             string                        `json:"operation_uuid"`
	CommitCheck               bool                          `json:"commit_check"`
	SignKeyID                 string                        `json:"sign_key_id"`
	DeprecatedCDSUserName     string                        `json:"cds_username"`    // Deprecated
	DeprecatedCDSUserID       string                        `json:"cds_username_id"` // Deprecated
	DeprecatedCDSAdminWithMFA bool                          `json:"cds_admin_mfa"`   // Deprecated
	Error                     string                        `json:"error"`
	Entities                  []ProjectRepositoryDataEntity `json:"entities"`
	Initiator                 *V2Initiator                  `json:"initiator"`
}

type ProjectRepositoryDataEntity struct {
	FileName string `json:"file_name"`
	Path     string `json:"path"`
	Status   string `json:"status"`
}

func (prd ProjectRepositoryData) Value() (driver.Value, error) {
	j, err := json.Marshal(prd)
	return j, WrapError(err, "cannot marshal ProjectRepositoryData")
}

func (prd *ProjectRepositoryData) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}

	if err := JSONUnmarshal(source, prd); err != nil {
		return WrapError(err, "cannot unmarshal ProjectRepositoryData")
	}

	if prd.Initiator == nil {
		prd.Initiator = &V2Initiator{
			UserID:         prd.DeprecatedCDSUserID,
			IsAdminWithMFA: prd.DeprecatedCDSAdminWithMFA,
		}
	}

	return nil
}
