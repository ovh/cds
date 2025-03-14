package sdk

import (
	"regexp"
	"time"
)

type ConcurrencyOrder string

const (
	ConcurrencyOrderOldestFirst ConcurrencyOrder = "oldest_first"
	ConcurrencyOrderNewestFirst ConcurrencyOrder = "newest_first"
)

func (c ConcurrencyOrder) IsValid() bool {
	return c == ConcurrencyOrderOldestFirst || c == ConcurrencyOrderNewestFirst
}

type ProjectConcurrency struct {
	ID               string           `json:"id" db:"id" cli:"id"`
	ProjectKey       string           `json:"project_key" db:"project_key"`
	Name             string           `json:"name" db:"name" cli:"name"`
	Description      string           `json:"description" db:"description" cli:"description"`
	Order            ConcurrencyOrder `json:"order" db:"order" cli:"order"`
	Pool             int64            `json:"pool" db:"pool" cli:"pool"`
	If               string           `json:"if" db:"if" cli:"if"`
	CancelInProgress bool             `json:"cancel_in_progress" db:"cancel_in_progress" cli:"cancel_in_progress"`
	LastModified     time.Time        `json:"last_modified" db:"last_modified" cli:"last_modified"`
}

func (pc *ProjectConcurrency) ToWorkflowConcurrency() WorkflowConcurrency {
	return WorkflowConcurrency{
		Name:             pc.Name,
		Order:            pc.Order,
		Pool:             pc.Pool,
		CancelInProgress: pc.CancelInProgress,
		If:               pc.If,
	}
}

func (pc *ProjectConcurrency) Check() error {
	if pc.Pool <= 0 {
		pc.Pool = 1
	}
	if !pc.Order.IsValid() {
		return NewErrorFrom(ErrInvalidData, "invalid order, got %q want %s | %s", pc.Order, ConcurrencyOrderOldestFirst, ConcurrencyOrderNewestFirst)
	}

	namePattern, err := regexp.Compile(EntityNamePattern)
	if err != nil {
		return WrapError(err, "unable to compile regexp %s", namePattern)
	}

	if !namePattern.MatchString(pc.Name) {
		return NewErrorFrom(ErrInvalidData, "name %s doesn't match %s", pc.Name, EntityNamePattern)
	}

	return nil
}

type ProjectConcurrencyRunObject struct {
	WorkflowRunID string    `json:"workflow_run_id" db:"workflow_run_id" cli:"workflow_run_id"`
	LastModified  time.Time `json:"last_modified" db:"last_modified" cli:"last_modified"`
	Type          string    `json:"type" db:"type" cli:"type"`
	WorkflowName  string    `json:"workflow_name" db:"workflow_name" cli:"workflow_name"`
	JobName       string    `json:"job_name" db:"job_name" cli:"job_name"`
	Status        string    `json:"status" db:"status" cli:"status"`
	RunNumber     int64     `json:"run_number" db:"run_number" cli:"run_number"`
}
