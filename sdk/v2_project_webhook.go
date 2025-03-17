package sdk

import (
	"time"
)

const (
	ProjectWebHookTypeRepository ProjectWebHookType = "repository"
	ProjectWebHookTypeWorkflow   ProjectWebHookType = "workflow"
)

type ProjectWebHookType string
type ProjectWebHook struct {
	ID         string             `json:"id" db:"id" cli:"id"`
	ProjectKey string             `json:"project_key" db:"project_key" cli:"project_key"`
	VCSServer  string             `json:"vcs_server" db:"vcs_server" cli:"vcs_server"`
	Repository string             `json:"repository" db:"repository" cli:"repository"`
	Workflow   string             `json:"workflow" db:"workflow" cli:"workflow"`
	Created    time.Time          `json:"created" db:"created" cli:"created"`
	Type       ProjectWebHookType `json:"type" db:"type" cli:"type"`
	Username   string             `json:"username" db:"username" cli:"username"`
}

type PostProjectWebHook struct {
	VCSServer  string             `json:"vcs_server"`
	Repository string             `json:"repository"`
	Workflow   string             `json:"workflow"`
	Type       ProjectWebHookType `json:"type"`
}

func (p PostProjectWebHook) Valid() error {
	if p.VCSServer == "" {
		return NewErrorFrom(ErrInvalidData, "missing vcs_server")
	}
	if p.Repository == "" {
		return NewErrorFrom(ErrInvalidData, "missing repository")
	}

	if p.Type == ProjectWebHookTypeWorkflow {
		if p.Workflow == "" {
			return NewErrorFrom(ErrInvalidData, "missing workflow")
		}
	}
	return nil
}
