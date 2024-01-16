package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type ProjectNotification struct {
	ID           string                     `json:"id" db:"id"`
	ProjectKey   string                     `json:"project_key" db:"project_key"`
	Name         string                     `json:"name" db:"name" cli:"name"`
	LastModified time.Time                  `json:"last_modified" db:"last_modified" cli:"last_modified"`
	WebHookURL   string                     `json:"webhook_url" db:"webhook_url" cli:"webhook_url"`
	Filters      ProjectNotificationFilters `json:"filters" db:"filters" cli:"filters"`
	Auth         ProjectNotificationAuth    `json:"auth" db:"auth" gorpmapping:"encrypted,ID,ProjectKey"`
}

type ProjectNotificationAuth struct {
	Headers map[string]string `json:"headers"`
}

type ProjectNotificationFilters map[string]ProjectNotificationFilter

type ProjectNotificationFilter struct {
	Events []string `json:"event"`
}

func (f ProjectNotificationFilters) Value() (driver.Value, error) {
	m, err := json.Marshal(f)
	return m, WrapError(err, "cannot marshal ProjectNotificationFilters")
}

func (s *ProjectNotificationFilters) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	if err := JSONUnmarshal(source, s); err != nil {
		return WrapError(err, "cannot unmarshal ProjectNotificationFilters")
	}
	return nil
}
