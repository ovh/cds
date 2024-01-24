package sdk

import "time"

const (
	ProjectVariableTypeSecret = "secret"
	ProjectVariableTypeString = "string"
)

type ProjectVariableSet struct {
	ID         string                   `json:"id" db:"id"`
	ProjectKey string                   `json:"project_key" db:"project_key"`
	Name       string                   `json:"name" db:"name"`
	Created    time.Time                `json:"created" db:"created"`
	Items      []ProjectVariableSetItem `json:"items" db:"-"`
}

type ProjectVariableSetItem struct {
	ID                   string    `json:"id" db:"id"`
	ProjectVariableSetID string    `json:"project_variable_set_id"`
	LastModified         time.Time `json:"last_modified"`
	Name                 string    `json:"name"`
	Type                 string    `json:"type"`
	Value                string    `json:"value"`
}
