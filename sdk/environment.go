package sdk

import (
	"time"
)

// Environment represent a deployment environment
type Environment struct {
	ID           int64            `json:"id" yaml:"-"`
	Name         string           `json:"name" yaml:"name" cli:"name,key"`
	Variable     []Variable       `json:"variables,omitempty" yaml:"variables"`
	ProjectID    int64            `json:"-" yaml:"-"`
	ProjectKey   string           `json:"project_key" yaml:"-"`
	Permission   int              `json:"permission"`
	LastModified int64            `json:"last_modified"`
	Keys         []EnvironmentKey `json:"keys"`
	Usage        *Usage           `json:"usage,omitempty"`
}

// EnvironmentVariableAudit represents an audit on an environment variable
type EnvironmentVariableAudit struct {
	ID             int64     `json:"id" yaml:"-" db:"id"`
	EnvironmentID  int64     `json:"environment_id" yaml:"-" db:"environment_id"`
	VariableID     int64     `json:"variable_id" yaml:"-" db:"variable_id"`
	Type           string    `json:"type" yaml:"-" db:"type"`
	VariableBefore *Variable `json:"variable_before,omitempty" yaml:"-" db:"-"`
	VariableAfter  *Variable `json:"variable_after,omitempty" yaml:"-" db:"-"`
	Versionned     time.Time `json:"versionned" yaml:"-" db:"versionned"`
	Author         string    `json:"author" yaml:"-" db:"author"`
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

// NewEnvironment instanciate a new Environment
func NewEnvironment(name string) *Environment {
	e := &Environment{
		Name: name,
	}
	return e
}

// DefaultEnv Default environment for pipeline build
var DefaultEnv = Environment{
	ID:   1,
	Name: "NoEnv",
}
