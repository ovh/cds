package sdk

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Environment represent a deployment environment
type Environment struct {
	ID                int64             `json:"id" yaml:"-"`
	Name              string            `json:"name" yaml:"name" cli:"name,key"`
	EnvironmentGroups []GroupPermission `json:"groups,omitempty" yaml:"groups"`
	Variable          []Variable        `json:"variables,omitempty" yaml:"variables"`
	ProjectID         int64             `json:"-" yaml:"-"`
	ProjectKey        string            `json:"project_key" yaml:"-"`
	Permission        int               `json:"permission"`
	LastModified      int64             `json:"last_modified"`
	Keys              []EnvironmentKey  `json:"keys"`
	Usage             *Usage            `json:"usage,omitempty"`
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

// AddEnvironment create an environment
func AddEnvironment(key, envName string) error {
	env := NewEnvironment(envName)
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/environment", key)
	data, _, err = Request("POST", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// UpdateEnvironment create an environment
func UpdateEnvironment(key, oldName, newName string) error {
	env := NewEnvironment(newName)
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/environment/%s", key, url.QueryEscape(oldName))
	data, _, err = Request("PUT", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// ListEnvironments returns all available environments for the given project
func ListEnvironments(key string) ([]Environment, error) {
	url := fmt.Sprintf("/project/%s/environment", key)
	data, _, err := Request("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var envs []Environment
	if err := json.Unmarshal(data, &envs); err != nil {
		return nil, err
	}

	return envs, nil
}

// GetEnvironment retrieve the given environment from CDS
func GetEnvironment(pk, name string) (*Environment, error) {
	var e Environment

	path := fmt.Sprintf("/project/%s/environment/%s", pk, url.QueryEscape(name))
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

// DeleteEnvironment delete an environment from CDS
func DeleteEnvironment(pk, name string) error {
	path := fmt.Sprintf("/project/%s/environment/%s", pk, url.QueryEscape(name))
	if _, _, err := Request("DELETE", path, nil); err != nil {
		return err
	}
	return nil
}

// CloneEnvironment clone the given environment in CDS
func CloneEnvironment(pk, name, new string) (*Environment, error) {
	path := fmt.Sprintf("/project/%s/environment/%s/clone/%s", pk, name, url.QueryEscape(new))
	if _, _, err := Request("POST", path, nil); err != nil {
		return nil, err
	}
	return GetEnvironment(pk, new)
}

// ShowEnvironmentVariable  show variables for an environment
func ShowEnvironmentVariable(projectKey, envName string) ([]Variable, error) {
	path := fmt.Sprintf("/project/%s/environment/%s/variable", projectKey, url.QueryEscape(envName))
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var variables []Variable
	if err := json.Unmarshal(data, &variables); err != nil {
		return nil, err
	}
	return variables, nil
}

// AddEnvironmentVariable  add a variable in an environment
func AddEnvironmentVariable(projectKey, envName, varName, varValue string, varType string) error {
	newVar := Variable{
		Name:  varName,
		Value: varValue,
		Type:  varType,
	}

	data, err := json.Marshal(newVar)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/environment/%s/variable/%s", projectKey, url.QueryEscape(envName), varName)
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// GetEnvironmentVariable Get a specific variable from the given environment
func GetEnvironmentVariable(projectKey, envName, varName string) (*Variable, error) {
	path := fmt.Sprintf("/project/%s/environment/%s/variable/%s", projectKey, url.QueryEscape(envName), varName)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	myVar := &Variable{}
	if err := json.Unmarshal(data, &myVar); err != nil {
		return nil, err
	}
	return myVar, nil
}

// UpdateEnvironmentVariable update a variable in an environment
func UpdateEnvironmentVariable(projectKey, envName, oldVarName, varName, varValue, varType string) error {
	oldVar, errGetVar := GetEnvironmentVariable(projectKey, envName, oldVarName)
	if errGetVar != nil {
		return errGetVar
	}

	newVar := Variable{
		ID:    oldVar.ID,
		Name:  varName,
		Value: varValue,
		Type:  varType,
	}

	data, err := json.Marshal(newVar)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/environment/%s/variable/%s", projectKey, url.QueryEscape(envName), varName)
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RemoveEnvironmentVariable  remove a variable from an environment
func RemoveEnvironmentVariable(projectKey, envName, varName string) error {
	path := fmt.Sprintf("/project/%s/environment/%s/variable/%s", projectKey, url.QueryEscape(envName), varName)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RemoveGroupFromEnvironment  call api to remove a group from the given environment
func RemoveGroupFromEnvironment(projectKey, envName, groupName string) error {
	path := fmt.Sprintf("/project/%s/environment/%s/group/%s", projectKey, url.QueryEscape(envName), groupName)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// UpdateGroupInEnvironment  call api to update group permission for the given environment
func UpdateGroupInEnvironment(projectKey, envName, groupName string, permission int) error {

	if permission < 4 || permission > 7 {
		return fmt.Errorf("Permission should be between 4-7")
	}

	groupApplication := GroupPermission{
		Group: Group{
			Name: groupName,
		},
		Permission: permission,
	}

	data, err := json.Marshal(groupApplication)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/environment/%s/group/%s", projectKey, url.QueryEscape(envName), groupName)
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// AddGroupInEnvironment  add a group in an environment
func AddGroupInEnvironment(projectKey, envName, groupName string, permission int) error {
	if permission < 4 || permission > 7 {
		return fmt.Errorf("Permission should be between 4-7")
	}

	groupPipeline := GroupPermission{
		Group: Group{
			Name: groupName,
		},
		Permission: permission,
	}

	data, err := json.Marshal(groupPipeline)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/environment/%s/group", projectKey, url.QueryEscape(envName))
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}
