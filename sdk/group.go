package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Group represent a group of user.
type Group struct {
	ID                int64              `json:"id"`
	Name              string             `json:"name"`
	Admins            []User             `json:"admins,omitempty"`
	Users             []User             `json:"users,omitempty"`
	ProjectGroups     []ProjectGroup     `json:"projects,omitempty"`
	PipelineGroups    []PipelineGroup    `json:"pipelines,omitempty"`
	ApplicationGroups []ApplicationGroup `json:"applications,omitempty"`
	EnvironmentGroups []EnvironmentGroup `json:"environments,omitempty"`
}

// GroupPermission represent a group and his role in the project
type GroupPermission struct {
	Group      Group `json:"group"`
	Permission int   `json:"permission"`
	Recursive  bool  `json:"recursive,omitempty"`
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

// NewGroup instanciate a new Group
func NewGroup(name string) *Group {
	g := &Group{
		Name: name,
	}
	return g
}

// JSON return the marshalled string of Group object
func (p *Group) JSON() string {

	data, err := json.Marshal(p)
	if err != nil {
		fmt.Printf("Group.JSON: cannot marshal: %s\n", err)
		return ""
	}

	return string(data)
}

// FromJSON unmarshal given json data into Group object
func (p *Group) FromJSON(data []byte) (*Group, error) {
	return p, json.Unmarshal(data, &p)
}

// AddGroup creates a new group
func AddGroup(name string) error {

	a := NewGroup(name)

	data, err := json.MarshalIndent(a, " ", " ")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/group")
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

// RemoveGroup remove group
func RemoveGroup(name string) error {
	url := fmt.Sprintf("/group/%s", name)
	data, code, err := Request("DELETE", url, nil)
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

// ListGroups returns all available group to caller
func ListGroups() ([]Group, error) {

	data, code, err := Request("GET", "/group", nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var groups []Group
	err = json.Unmarshal(data, &groups)
	if err != nil {
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

// GetGroup call API to get Group information
func GetGroup(groupName string) (Group, error) {
	path := fmt.Sprintf("/group/%s", groupName)

	var group Group
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return group, err
	}

	if code != http.StatusOK {
		return group, fmt.Errorf("Error [%d]: %s", code, data)
	}

	err = json.Unmarshal(data, &group)
	if err != nil {
		return group, err
	}

	return group, nil
}

// RemoveUserFromGroup call API to remove a  user in the group
func RemoveUserFromGroup(groupName string, userName string) error {
	path := fmt.Sprintf("/group/%s/user/%s", groupName, userName)
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

// SetUserGroupAdmin grants to given user privileges on given group
func SetUserGroupAdmin(groupName string, userName string) error {
	uri := fmt.Sprintf("/group/%s/user/%s/admin", groupName, userName)

	_, code, err := Request("POST", uri, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d\n", code)
	}

	return nil
}

// UnsetUserGroupAdmin removes user priviles on group
func UnsetUserGroupAdmin(groupName string, userName string) error {
	uri := fmt.Sprintf("/group/%s/user/%s/admin", groupName, userName)

	_, code, err := Request("DELETE", uri, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d\n", code)
	}

	return nil
}
