package sdk

import (
	"encoding/json"
	"fmt"
)

// SharedInfraGroupName is the name of the builtin group used to share infrastructure between projects
const SharedInfraGroupName = "shared.infra"

// Group represent a group of user.
type Group struct {
	ID     int64   `json:"id" yaml:"-"`
	Name   string  `json:"name" yaml:"name" cli:"name"`
	Admins []User  `json:"admins,omitempty" yaml:"admin,omitempty"`
	Users  []User  `json:"users,omitempty" yaml:"users,omitempty"`
	Tokens []Token `json:"tokens,omitempty" yaml:"tokens,omitempty"`
}

// GroupPermission represent a group and his role in the project
type GroupPermission struct {
	Group      Group `json:"group"`
	Permission int   `json:"permission"`
}

// EnvironmentGroup represent a link with a pipeline
type EnvironmentGroup struct {
	Environment Environment `json:"environment"`
	Permission  int         `json:"permission"`
}

// ApplicationGroup represent a link with a pipeline
type ApplicationGroup struct {
	Application Application `json:"application"`
	Permission  int         `json:"permission"`
}

// PipelineGroup represent a link with a pipeline
type PipelineGroup struct {
	Pipeline   Pipeline `json:"pipeline"`
	Permission int      `json:"permission"`
}

// ProjectGroup represent a link with a project
type ProjectGroup struct {
	Project    Project `json:"project"`
	Permission int     `json:"permission"`
}

// WorkflowGroup represents the permission to a workflow
type WorkflowGroup struct {
	Workflow   Workflow `json:"workflow"`
	Permission int      `json:"permission"`
}

// AddGroup creates a new group
func AddGroup(name string) error {
	a := Group{Name: name}
	data, err := json.MarshalIndent(a, " ", " ")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/group")
	data, _, err = Request("POST", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RenameGroup Rename a group
func RenameGroup(oldName, newName string) error {
	g, err := GetGroup(oldName)
	if err != nil {
		return err
	}

	g.Name = newName
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/group/%s", oldName)
	data, _, err = Request("PUT", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RemoveGroup remove group
func RemoveGroup(name string) error {
	url := fmt.Sprintf("/group/%s", name)
	data, _, err := Request("DELETE", url, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// ListGroups returns all available group to caller
func ListGroups() ([]Group, error) {
	data, _, err := Request("GET", "/group", nil)
	if err != nil {
		return nil, err
	}

	var groups []Group
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// AddUsersInGroup call API to add users in the group
func AddUsersInGroup(groupName string, users []string) error {
	data, err := json.MarshalIndent(users, " ", " ")
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/group/%s/user", groupName)
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// GetGroup call API to get Group information
func GetGroup(groupName string) (Group, error) {
	path := fmt.Sprintf("/group/%s", groupName)

	var group Group
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return group, err
	}

	if err := json.Unmarshal(data, &group); err != nil {
		return group, err
	}

	return group, nil
}

// RemoveUserFromGroup call API to remove a  user in the group
func RemoveUserFromGroup(groupName string, userName string) error {
	path := fmt.Sprintf("/group/%s/user/%s", groupName, userName)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// SetUserGroupAdmin grants to given user privileges on given group
func SetUserGroupAdmin(groupName string, userName string) error {
	uri := fmt.Sprintf("/group/%s/user/%s/admin", groupName, userName)
	_, _, err := Request("POST", uri, nil)
	return err
}

// UnsetUserGroupAdmin removes user priviles on group
func UnsetUserGroupAdmin(groupName string, userName string) error {
	uri := fmt.Sprintf("/group/%s/user/%s/admin", groupName, userName)
	_, _, err := Request("DELETE", uri, nil)
	return err
}
