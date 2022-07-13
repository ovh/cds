package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const (
	RepositoryAnalyzeStatusInProgress = "InProgress"
	RepositoryAnalyzeStatusSucceed    = "Success"
	RepositoryAnalyzeStatusError      = "Error"
)

type ProjectRepository struct {
	ID                string                `json:"id" db:"id"`
	Name              string                `json:"name" db:"name" cli:"name,key"`
	Created           time.Time             `json:"created" db:"created"`
	CreatedBy         string                `json:"created_by" db:"created_by"`
	VCSProjectID      string                `json:"-" db:"vcs_project_id"`
	HookConfiguration HookConfiguration     `json:"hook_configuration" db:"hook_configuration"`
	CloneURL          string                `json:"clone_url" db:"clone_url"`
	Auth              ProjectRepositoryAuth `json:"auth" db:"auth" gorpmapping:"encrypted,ID,VCSProjectID"`
}

type ProjectRepositoryAuth struct {
	Username   string `json:"username,omitempty" db:"-"`
	Token      string `json:"token,omitempty" db:"-"`
	SSHKeyName string `json:"sshPrivateKey,omitempty" db:"-"`
}

type ProjectRepositoryAnalyze struct {
	ID                  string                `json:"id" db:"id"`
	Created             time.Time             `json:"created" db:"created"`
	LastModified        time.Time             `json:"last_modified" db:"last_modified"`
	ProjectRepositoryID string                `json:"project_repository_id" db:"project_repository_id"`
	Status              string                `json:"status" db:"status"`
	Branch              string                `json:"branch" db:"branch"`
	Commit              string                `json:"commit" db:"commit"`
	Data                ProjectRepositoryData `json:"data" db:"data"`
}

type ProjectRepositoryData struct {
	OperationUUID string                       `json:"operation_uuid"`
	CommitCheck   bool                         `json:"commit_check"`
	Error         string                       `json:"error"`
	Entities      []ProjectRepositoryDataEnity `json:"entities"`
}

type ProjectRepositoryDataEnity struct {
	FileName string `json:"file_name"`
	Path     string `json:"path"`
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
	return WrapError(JSONUnmarshal(source, prd), "cannot unmarshal ProjectRepositoryData")
}
