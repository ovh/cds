package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Project represent a team with group of users and pipelines
type Project struct {
	ID                int64              `json:"-" yaml:"-" db:"id" cli:"-"`
	Key               string             `json:"key" yaml:"key" db:"projectkey" cli:"key,key"`
	Name              string             `json:"name" yaml:"name" db:"name" cli:"name"`
	Workflows         []Workflow         `json:"workflows,omitempty" yaml:"workflows,omitempty" db:"-" cli:"-"`
	WorkflowNames     []string           `json:"workflow_names,omitempty" yaml:"workflow_names,omitempty" db:"-" cli:"-"`
	Pipelines         []Pipeline         `json:"pipelines,omitempty" yaml:"pipelines,omitempty" db:"-"  cli:"-"`
	PipelineNames     []IDName           `json:"pipeline_names,omitempty" yaml:"pipeline_names,omitempty" db:"-"  cli:"-"`
	Applications      []Application      `json:"applications,omitempty" yaml:"applications,omitempty" db:"-"  cli:"-"`
	ApplicationNames  []IDName           `json:"application_names,omitempty" yaml:"application_names,omitempty" db:"-"  cli:"-"`
	ProjectGroups     []GroupPermission  `json:"groups,omitempty" yaml:"permissions,omitempty" db:"-"  cli:"-"`
	Variable          []Variable         `json:"variables,omitempty" yaml:"variables,omitempty" db:"-"  cli:"-"`
	Environments      []Environment      `json:"environments,omitempty"  yaml:"environments,omitempty" db:"-"  cli:"-"`
	Permission        int                `json:"permission"  yaml:"-" db:"-"  cli:"-"`
	Created           time.Time          `json:"created"  yaml:"created" db:"created" `
	LastModified      time.Time          `json:"last_modified"  yaml:"last_modified" db:"last_modified"`
	Metadata          Metadata           `json:"metadata" yaml:"metadata" db:"-" cli:"-"`
	WorkflowMigration string             `json:"workflow_migration" yaml:"workflow_migration" db:"workflow_migration"`
	Keys              []ProjectKey       `json:"keys" yaml:"keys" db:"-" cli:"-"`
	VCSServers        []ProjectVCSServer `json:"vcs_servers" yaml:"vcs_servers" db:"-" cli:"-"`
	Platforms         []ProjectPlatform  `json:"platforms" yaml:"platforms" db:"-" cli:"-"`
	Features          map[string]bool    `json:"features" yaml:"features" db:"-" cli:"-"`
	Favorite          bool               `json:"favorite" yaml:"favorite" db:"-" cli:"favorite"`
}

// SSHKeys returns the slice of ssh key for an application
func (a Project) SSHKeys() []ProjectKey {
	keys := []ProjectKey{}
	for _, k := range a.Keys {
		if k.Type == KeyTypeSSH {
			keys = append(keys, k)
		}
	}
	return keys
}

// PGPKeys returns the slice of pgp key for an application
func (a Project) PGPKeys() []ProjectKey {
	keys := []ProjectKey{}
	for _, k := range a.Keys {
		if k.Type == KeyTypePGP {
			keys = append(keys, k)
		}
	}
	return keys
}

// GetPlatform returns the ProjectPlatform given a name
func (a Project) GetPlatform(pfName string) *ProjectPlatform {
	for i := range a.Platforms {
		if a.Platforms[i].Name == pfName {
			return &a.Platforms[i]
		}
	}
	return nil
}

// ProjectVCSServer represents associations between a project and a vcs server
type ProjectVCSServer struct {
	Name string            `json:"name" yaml:"name" db:"-" cli:"-"`
	Data map[string]string `json:"-" yaml:"data" db:"-" cli:"-"`
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
	Key          string `json:"key,omitempty"`
	Name         string `json:"name"`
	Username     string `json:"username"`
	LastModified int64  `json:"last_modified"`
	Type         string `json:"type,omitempty"`
}

const (
	// ApplicationLastModificationType represent key for last update event about application
	ApplicationLastModificationType = "application"
	// PipelineLastModificationType represent key for last update event about pipeline
	PipelineLastModificationType = "pipeline"
	// WorkflowLastModificationType represent key for last update event about workflow
	WorkflowLastModificationType = "workflow"
	// ProjectLastModificationType represent key for last update event about project
	ProjectLastModificationType = "project"
	// ProjectPipelineLastModificationType represent key for last update event about project.pipeline (rename, delete or add a pipeline)
	ProjectPipelineLastModificationType = "project.pipeline"
	// ProjectApplicationLastModificationType represent key for last update event about project.application (rename, delete or add an application)
	ProjectApplicationLastModificationType = "project.application"
	// ProjectEnvironmentLastModificationType represent key for last update event about project.environment (rename, delete or add an environment)
	ProjectEnvironmentLastModificationType = "project.environment"
	// ProjectWorkflowLastModificationType represent key for last update event about project.workflow (rename, delete or add a workflow)
	ProjectWorkflowLastModificationType = "project.workflow"
	// ProjectVariableLastModificationType represent key for last update event about project.variable (rename, delete or add a variable)
	ProjectVariableLastModificationType = "project.variable"
	// ProjectKeysLastModificationType represent key for last update event about project.keys (add, delete a key)
	ProjectKeysLastModificationType = "project.keys"
	// ProjectPlatformsLastModificationType represent key for last update event about project.platforms (add, update, delete a platform)
	ProjectPlatformsLastModificationType = "project.platforms"
)

//ProjectLastUpdates update times of project, application and pipelines
// Deprecated
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

// UpdateProject call API to update project
func UpdateProject(proj *Project) error {
	data, err := json.Marshal(proj)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s", proj.Key)
	data, _, err = Request("PUT", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
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
	data, _, err = Request("PUT", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RemoveGroupFromProject  call api to remove a group from the project
func RemoveGroupFromProject(projectKey, groupname string) error {
	path := fmt.Sprintf("/project/%s/group/%s", projectKey, groupname)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// UpdateGroupInProject  call api to update group permission on project
func UpdateGroupInProject(projectKey, groupname string, permission int) error {
	if permission < 4 || permission > 7 {
		return fmt.Errorf("permission should be between 4-7")
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
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// AddGroupInProject  add a group in a project
func AddGroupInProject(projectKey, groupname string, permission int) error {
	if permission < 4 || permission > 7 {
		return fmt.Errorf("permission should be between 4-7")
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
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// GetVariableInProject Get a variable by her name in the given project
func GetVariableInProject(projectKey, name string) (*Variable, error) {
	var v Variable

	path := fmt.Sprintf("/project/%s/variable/%s", projectKey, name)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	e := DecodeError(data)
	if e != nil {
		return nil, e
	}

	if err := json.Unmarshal(data, &v); err != nil {
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
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RemoveVariableFromProject  remove a variable from a project
func RemoveVariableFromProject(projectKey, varName string) error {
	path := fmt.Sprintf("/project/%s/variable/%s", projectKey, varName)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// Mod is the functional parameter type of sdk function to alter their behavior
type Mod func(s string) string

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
