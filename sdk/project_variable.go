package sdk

import "time"

const (
	ProjectVariableTypeSecret = "secret"
	ProjectVariableTypeString = "string"

	ProjectVariableSetItemNamePattern = "^[a-zA-Z0-9_-]{1,}$"
)

type ProjectVariableSet struct {
	ID         string                   `json:"id" db:"id"`
	ProjectKey string                   `json:"project_key" db:"project_key"`
	Name       string                   `json:"name" db:"name" cli:"name" action_metadata:"variable-set-name"`
	Created    time.Time                `json:"created" db:"created" cli:"created"`
	Items      []ProjectVariableSetItem `json:"items" db:"-"`
}

type ProjectVariableSetItem struct {
	ID                   string    `json:"id" db:"id"`
	ProjectVariableSetID string    `json:"project_variable_set_id"`
	LastModified         time.Time `json:"last_modified" cli:"last_modified"`
	Name                 string    `json:"name" cli:"name"`
	Type                 string    `json:"type" cli:"type"`
	Value                string    `json:"value" cli:"value"`
}

type CopyProjectVariableToVariableSet struct {
	VariableName    string `json:"variable_name"`
	VariableSetName string `json:"variable_set_name"`
	NewName         string `json:"new_name,omitempty"`
}

type CopyApplicationVariableToVariableSet struct {
	ApplicationName string `json:"application_name"`
	VariableSetName string `json:"variable_set_name"`
}

type CopyEnvironmentVariableToVariableSet struct {
	EnvironmentName string `json:"environment_name"`
	VariableSetName string `json:"variable_set_name"`
}
