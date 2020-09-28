package sdk

import (
	json "encoding/json"
	"time"
)

// Environment represent a deployment environment
type Environment struct {
	ID                   int64                 `json:"id" yaml:"-"`
	Name                 string                `json:"name" yaml:"name" cli:"name,key"`
	Variables            []EnvironmentVariable `json:"variables,omitempty" yaml:"variables"`
	ProjectID            int64                 `json:"-" yaml:"-"`
	ProjectKey           string                `json:"project_key" yaml:"-"`
	Created              time.Time             `json:"created"`
	LastModified         time.Time             `json:"last_modified"`
	Keys                 []EnvironmentKey      `json:"keys"`
	Usage                *Usage                `json:"usage,omitempty"`
	FromRepository       string                `json:"from_repository,omitempty"`
	WorkflowAscodeHolder *Workflow             `json:"workflow_ascode_holder,omitempty" cli:"-"`
}

// UnmarshalJSON custom for last modified.
func (e *Environment) UnmarshalJSON(data []byte) error {
	var tmp struct {
		ID             int64                 `json:"id"`
		Name           string                `json:"name"`
		Variables      []EnvironmentVariable `json:"variables"`
		ProjectKey     string                `json:"project_key"`
		Created        time.Time             `json:"created"`
		Keys           []EnvironmentKey      `json:"keys"`
		Usage          *Usage                `json:"usage"`
		FromRepository string                `json:"from_repository"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	e.ID = tmp.ID
	e.Name = tmp.Name
	e.Variables = tmp.Variables
	e.ProjectKey = tmp.ProjectKey
	e.Created = tmp.Created
	e.Keys = tmp.Keys
	e.Usage = tmp.Usage
	e.FromRepository = tmp.FromRepository

	var v map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if lastModifiedNumber, ok := v["last_modified"].(float64); ok {
		e.LastModified = time.Unix(int64(lastModifiedNumber), 0)
	}
	if lastModifiedString, ok := v["last_modified"].(string); ok {
		date, _ := time.Parse(time.RFC3339, lastModifiedString)
		e.LastModified = date
	}

	return nil
}

// EnvironmentVariableAudit represents an audit on an environment variable
type EnvironmentVariableAudit struct {
	ID             int64                `json:"id" yaml:"-" db:"id"`
	EnvironmentID  int64                `json:"environment_id" yaml:"-" db:"environment_id"`
	VariableID     int64                `json:"variable_id" yaml:"-" db:"variable_id"`
	Type           string               `json:"type" yaml:"-" db:"type"`
	VariableBefore *EnvironmentVariable `json:"variable_before,omitempty" yaml:"-" db:"-"`
	VariableAfter  EnvironmentVariable  `json:"variable_after,omitempty" yaml:"-" db:"-"`
	Versionned     time.Time            `json:"versionned" yaml:"-" db:"versionned"`
	Author         string               `json:"author" yaml:"-" db:"author"`
}

// GetKey return a key by name
func (e Environment) GetKey(kname string) *EnvironmentKey {
	for i := range e.Keys {
		if e.Keys[i].Name == kname {
			return &e.Keys[i]
		}
	}
	return nil
}

// GetSSHKey return a key by name
func (e Environment) GetSSHKey(kname string) *EnvironmentKey {
	for i := range e.Keys {
		if e.Keys[i].Type == KeyTypeSSH && e.Keys[i].Name == kname {
			return &e.Keys[i]
		}
	}
	return nil
}

// SSHKeys returns the slice of ssh key for an environment
func (e Environment) SSHKeys() []EnvironmentKey {
	keys := []EnvironmentKey{}
	for _, k := range e.Keys {
		if k.Type == KeyTypeSSH {
			keys = append(keys, k)
		}
	}
	return keys
}

// PGPKeys returns the slice of pgp key for an environment
func (e Environment) PGPKeys() []EnvironmentKey {
	keys := []EnvironmentKey{}
	for _, k := range e.Keys {
		if k.Type == KeyTypePGP {
			keys = append(keys, k)
		}
	}
	return keys
}

// NewEnvironment instantiate a new Environment
func NewEnvironment(name string) *Environment {
	e := &Environment{
		Name: name,
	}
	return e
}
