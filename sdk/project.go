package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Project represent a team with group of users and pipelines
type Project struct {
	ID            int64                 `json:"-" yaml:"-" db:"id" cli:"-"`
	Key           string                `json:"key" yaml:"key" db:"projectkey" cli:"key,key"`
	Name          string                `json:"name" yaml:"name" db:"name" cli:"name"`
	Pipelines     []Pipeline            `json:"pipelines,omitempty" yaml:"pipelines,omitempty" db:"-"  cli:"-"`
	Applications  []Application         `json:"applications,omitempty" yaml:"applications,omitempty" db:"-"  cli:"-"`
	ProjectGroups []GroupPermission     `json:"groups,omitempty" yaml:"permissions,omitempty" db:"-"  cli:"-"`
	Variable      []Variable            `json:"variables,omitempty" yaml:"variables,omitempty" db:"-"  cli:"-"`
	Environments  []Environment         `json:"environments,omitempty"  yaml:"environments,omitempty" db:"-"  cli:"-"`
	Permission    int                   `json:"permission"  yaml:"-" db:"-"  cli:"-"`
	Created       time.Time             `json:"created"  yaml:"created" db:"created" `
	LastModified  time.Time             `json:"last_modified"  yaml:"last_modified" db:"last_modified"`
	ReposManager  []RepositoriesManager `json:"repositories_manager"  yaml:"-" db:"-" cli:"-"`
	Metadata      Metadata              `json:"metadata" yaml:"metadata" db:"-" cli:"-"`
}

// ProjectVariableAudit represents an audit on a project variable
type ProjectVariableAudit struct {
	ID             int64     `json:"id" yaml:"-" db:"id"`
	ProjectID      int64     `json:"project_id" yaml:"-" db:"project_id"`
	VariableID     int64     `json:"variable_id" yaml:"-" db:"variable_id"`
	Type           string    `json:"type" yaml:"-" db:"type"`
	VariableBefore *Variable `json:"variable_before,omitempty" yaml:"-" db:"-"`
	VariableAfter  *Variable `json:"variable_after,omitempty" yaml:"-" db:"-"`
	Versionned     time.Time `json:"versionned" yaml:"-" db:"versionned"`
	Author         string    `json:"author" yaml:"-" db:"author"`
}

// Metadata represents metadata
type Metadata map[string]string

//LastModification is stored in cache and used for ProjectLastUpdates computing
type LastModification struct {
	Name         string `json:"name"`
	Username     string `json:"username"`
	LastModified int64  `json:"last_modified"`
}

//ProjectLastUpdates update times of project, application and pipelines
type ProjectLastUpdates struct {
	LastModification
	Applications []LastModification `json:"applications"`
	Pipelines    []LastModification `json:"pipelines"`
	Environments []LastModification `json:"environments"`
	Workflows    []LastModification `json:"workflows"`
}

// ProjectKeyPattern  pattern for project key
const ProjectKeyPattern = "^[A-Z0-9]{1,}$"

// NewProject instanciate a new NewProject
func NewProject(key string) *Project {
	p := &Project{
		Key: key,
	}
	return p
}

// RemoveProject call api to delete a project
func RemoveProject(key string) error {

	url := fmt.Sprintf("/project/%s", key)
	data, code, err := Request("DELETE", url, nil)
	if err != nil {
		return err
	}

	if code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return e
	}

	return nil
}

// UpdateProject call API to update project
func UpdateProject(proj *Project) error {
	data, err := json.Marshal(proj)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s", proj.Key)
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

// RenameProject call API to update project
func RenameProject(key, newName string) error {

	p := NewProject(key)
	p.Name = newName

	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s", key)
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

// AddProject creates a new project available only to creator by default
func AddProject(name, key, groupName string) error {

	regexp := regexp.MustCompile(ProjectKeyPattern)
	if !regexp.MatchString(key) {
		return fmt.Errorf("project key '%s' must contain only upper-case alphanumerical characters", key)
	}

	p := NewProject(key)
	p.Name = name

	group := Group{Name: groupName}
	p.ProjectGroups = append(p.ProjectGroups, GroupPermission{Group: group, Permission: 7})

	data, err := json.MarshalIndent(p, " ", " ")
	if err != nil {
		return err
	}

	data, code, err := Request("POST", "/project", data)
	if err != nil {
		return err
	}

	if code == 409 {
		return ErrConflict
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

// RemoveGroupFromProject  call api to remove a group from the project
func RemoveGroupFromProject(projectKey, groupname string) error {

	path := fmt.Sprintf("/project/%s/group/%s", projectKey, groupname)
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

// UpdateGroupInProject  call api to update group permission on project
func UpdateGroupInProject(projectKey, groupname string, permission int) error {

	if permission < 4 || permission > 7 {
		return fmt.Errorf("Permission should be between 4-7 \n")
	}

	groupProject := GroupPermission{
		Group: Group{
			Name: groupname,
		},
		Permission: permission,
	}

	data, err := json.MarshalIndent(groupProject, " ", " ")
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/group/%s", projectKey, groupname)
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

// AddGroupInProject  add a group in a project
func AddGroupInProject(projectKey, groupname string, permission int) error {

	if permission < 4 || permission > 7 {
		return fmt.Errorf("Permission should be between 4-7 \n")
	}

	groupProject := GroupPermission{
		Group: Group{
			Name: groupname,
		},
		Permission: permission,
	}

	data, err := json.MarshalIndent(groupProject, " ", " ")
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/group", projectKey)
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

// ShowVariableInProject  show variables for a project
func ShowVariableInProject(projectKey string) ([]Variable, error) {

	path := fmt.Sprintf("/project/%s/variable", projectKey)
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

// AddProjectVariable adds a project wide variable
func AddProjectVariable(key, name, val string, t string) error {
	return AddVariableInProject(key, name, val, string(t))
}

// AddVariableInProject  add a variable in a project
func AddVariableInProject(projectKey, varName, varValue, varType string) error {

	newVar := Variable{
		Name:  varName,
		Value: varValue,
		Type:  varType,
	}

	data, err := json.Marshal(newVar)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/variable/%s", projectKey, varName)
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

// GetVariableInProject Get a variable by her name in the given project
func GetVariableInProject(projectKey, name string) (*Variable, error) {
	var v Variable

	path := fmt.Sprintf("/project/%s/variable/%s", projectKey, name)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}
	e := DecodeError(data)
	if e != nil {
		return nil, e
	}

	err = json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// UpdateVariableInProject  update a variable in a project
func UpdateVariableInProject(projectKey, oldName, varName, varValue, varType string) error {
	oldVar, err := GetVariableInProject(projectKey, oldName)
	if err != nil {
		return err
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

	path := fmt.Sprintf("/project/%s/variable/%s", projectKey, varName)
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

// RemoveVariableFromProject  remove a variable from a project
func RemoveVariableFromProject(projectKey, varName string) error {
	path := fmt.Sprintf("/project/%s/variable/%s", projectKey, varName)
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

// Mod is the functional parameter type of sdk function to alter their behavior
type Mod func(s string) string

// WithApplicationStatus is a func parameter of ListProject
func WithApplicationStatus() Mod {
	f := func(s string) string {
		return s + "&applicationStatus=true"
	}

	return f
}

// WithEnvironments is a func parameter of ListProject
func WithEnvironments() Mod {
	f := func(s string) string {
		return s + "&environment=true"
	}

	return f
}

// WithPipelines is a func parameter of ListProject
func WithPipelines() Mod {
	f := func(s string) string {
		return s + "&pipeline=true"
	}

	return f
}

// WithEverything is a func parameter of ListProject
func WithEverything() Mod {
	f := func(s string) string {
		return s + "&pipeline=true&environment=true&applicationStatus=true"
	}

	return f
}

// ListProject returns all available project to caller
func ListProject(mods ...Mod) ([]Project, error) {
	uri := "/project?application=true"
	for _, m := range mods {
		uri = m(uri)
	}

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var projects []Project
	err = json.Unmarshal(data, &projects)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

// WithApplicationHistory is a functional parameter of GetProject
func WithApplicationHistory(length int) Mod {
	f := func(s string) string {
		if strings.HasSuffix(s, "?") {
			return fmt.Sprintf("%sapplicationHistory=%d", s, length)
		}
		return fmt.Sprintf("%s&applicationHistory=%d", s, length)
	}

	return f
}

// GetProject retrieves project informations from CDS
func GetProject(key string, mods ...RequestModifier) (Project, error) {
	var p Project
	path := fmt.Sprintf("/project/%s", key)

	if len(mods) == 0 {
		mods = append(mods, func(r *http.Request) {
			q := r.URL.Query()
			q.Set("withApplications", "true")
			q.Set("withPipelines", "true")
			q.Set("withEnvironments", "true")
			q.Set("withGroups", "true")
			r.URL.RawQuery = q.Encode()
		})
	}

	data, _, err := Request("GET", path, nil, mods...)
	if err != nil {
		return p, err
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return p, err
	}

	return p, nil
}

// DeleteProject removes a project and all its pipeline from CDS
func DeleteProject(key string) error {

	path := fmt.Sprintf("/project/%s", key)
	_, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return nil
}
