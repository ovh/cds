package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type AsCodeEvent struct {
	ID             int64           `json:"id" db:"id"`
	WorkflowID     int64           `json:"workflow_id" db:"workflow_id"`
	PullRequestID  int64           `json:"pullrequest_id" db:"pullrequest_id"`
	PullRequestURL string          `json:"pullrequest_url" db:"pullrequest_url"`
	Username       string          `json:"username" db:"username"`
	CreateDate     time.Time       `json:"creation_date" db:"creation_date"`
	FromRepo       string          `json:"from_repository" db:"from_repository"`
	Migrate        bool            `json:"migrate" db:"migrate"`
	Data           AsCodeEventData `json:"data" db:"data"`
}

type AsCodeEventData struct {
	Workflows    AsCodeEventDataValue `json:"workflows"`
	Pipelines    AsCodeEventDataValue `json:"pipelines"`
	Applications AsCodeEventDataValue `json:"applications"`
	Environments AsCodeEventDataValue `json:"environments"`
}

type AsCodeEventDataValue map[int64]string

// Scan consumer data.
func (d *AsCodeEventData) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(JSONUnmarshal(source, d), "cannot unmarshal AsCodeEventData")
}

// Value returns driver.Value from consumer data.
func (d AsCodeEventData) Value() (driver.Value, error) {
	j, err := json.Marshal(d)
	return j, WrapError(err, "cannot marshal AsCodeEventData")
}
