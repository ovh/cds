package sdk

import (
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
	CancelInProgress bool             `json:"cancel_in_progress" db:"cancel_in_progress" cli:"cancel_in_progress"`
	LastModified     time.Time        `json:"last_modified" db:"last_modified" cli:"last_modified"`
}

func (pc *ProjectConcurrency) ToWorkflowConcurrency() WorkflowConcurrency {
	return WorkflowConcurrency{
		Name:             pc.Name,
		Order:            pc.Order,
		Pool:             pc.Pool,
		CancelInProgress: pc.CancelInProgress,
	}
}

func (pc *ProjectConcurrency) Check() error {
	if pc.Pool <= 0 {
		pc.Pool = 1
	}
	if !pc.Order.IsValid() {
		return NewErrorFrom(ErrInvalidData, "invalid order, got %q want %s | %s", pc.Order, ConcurrencyOrderOldestFirst, ConcurrencyOrderNewestFirst)
	}
	return nil
}
