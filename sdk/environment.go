package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Environment represent a deployment environment
type Environment struct {
	ID                int64             `json:"id" yaml:"-"`
	Name              string            `json:"name" yaml:"name" cli:"name"`
	EnvironmentGroups []GroupPermission `json:"groups,omitempty" yaml:"groups"`
	Variable          []Variable        `json:"variables,omitempty" yaml:"variables"`
	ProjectID         int64             `json:"-" yaml:"-"`
	ProjectKey        string            `json:"project_key" yaml:"-"`
	Permission        int               `json:"permission"`
	LastModified      int64             `json:"last_modified"`
	Keys              []EnvironmentKey  `json:"keys"`
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
	data, code, err := Request("POST", url, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}

// UpdateEnvironment create an environment
func UpdateEnvironment(key, oldName, newName string) error {

	env := NewEnvironment(newName)
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/environment/%s", key, url.QueryEscape(oldName))
	data, code, err := Request("PUT", url, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}

// ListEnvironments returns all available environments for the given project
func ListEnvironments(key string) ([]Environment, error) {

	url := fmt.Sprintf("/project/%s/environment", key)
	data, code, err := Request("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var envs []Environment
	err = json.Unmarshal(data, &envs)
	if err != nil {
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

	err = json.Unmarshal(data, &e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// DeleteEnvironment delete an environment from CDS
func DeleteEnvironment(pk, name string) error {

	path := fmt.Sprintf("/project/%s/environment/%s", pk, url.QueryEscape(name))
	_, _, err := Request("DELETE", path, nil)
	if err != nil {
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
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var variables []Variable
	err = json.Unmarshal(data, &variables)
	if err != nil {
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
	data, code, err := Request("POST", path, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}

	return nil
}

// GetEnvironmentVariable Get a specific variable from the given environment
func GetEnvironmentVariable(projectKey, envName, varName string) (*Variable, error) {
	path := fmt.Sprintf("/project/%s/environment/%s/variable/%s", projectKey, url.QueryEscape(envName), varName)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	myVar := &Variable{}

	err = json.Unmarshal(data, &myVar)
	if err != nil {
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
	data, code, err := Request("PUT", path, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}

	return nil
}

// RemoveEnvironmentVariable  remove a variable from an environment
func RemoveEnvironmentVariable(projectKey, envName, varName string) error {
	path := fmt.Sprintf("/project/%s/environment/%s/variable/%s", projectKey, url.QueryEscape(envName), varName)
	data, code, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}

	return nil
}

// RemoveGroupFromEnvironment  call api to remove a group from the given environment
func RemoveGroupFromEnvironment(projectKey, envName, groupName string) error {
	path := fmt.Sprintf("/project/%s/environment/%s/group/%s", projectKey, url.QueryEscape(envName), groupName)
	data, code, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
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
	data, code, err := Request("PUT", path, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}

// AddGroupInEnvironment  add a group in an environment
func AddGroupInEnvironment(projectKey, envName, groupName string, permission int) error {

	if permission < 4 || permission > 7 {
		return fmt.Errorf("Permission should be between 4-7 \n")
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
	data, code, err := Request("POST", path, data)
	if err != nil {
		return err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}
	return nil
}
