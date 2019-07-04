package sdk

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"
)

// Project represent a team with group of users and pipelines
type Project struct {
	ID               int64                `json:"-" yaml:"-" db:"id" cli:"-"`
	Key              string               `json:"key" yaml:"key" db:"projectkey" cli:"key,key"`
	Name             string               `json:"name" yaml:"name" db:"name" cli:"name"`
	Description      string               `json:"description" yaml:"description" db:"description" cli:"description"`
	Icon             string               `json:"icon" yaml:"icon" db:"icon" cli:"-"`
	Workflows        []Workflow           `json:"workflows,omitempty" yaml:"workflows,omitempty" db:"-" cli:"-"`
	WorkflowNames    IDNames              `json:"workflow_names,omitempty" yaml:"workflow_names,omitempty" db:"-" cli:"-"`
	Pipelines        []Pipeline           `json:"pipelines,omitempty" yaml:"pipelines,omitempty" db:"-"  cli:"-"`
	PipelineNames    IDNames              `json:"pipeline_names,omitempty" yaml:"pipeline_names,omitempty" db:"-"  cli:"-"`
	Applications     []Application        `json:"applications,omitempty" yaml:"applications,omitempty" db:"-"  cli:"-"`
	ApplicationNames IDNames              `json:"application_names,omitempty" yaml:"application_names,omitempty" db:"-"  cli:"-"`
	ProjectGroups    []GroupPermission    `json:"groups,omitempty" yaml:"permissions,omitempty" db:"-"  cli:"-"`
	Variable         []Variable           `json:"variables,omitempty" yaml:"variables,omitempty" db:"-"  cli:"-"`
	Environments     []Environment        `json:"environments,omitempty"  yaml:"environments,omitempty" db:"-"  cli:"-"`
	EnvironmentNames IDNames              `json:"environment_names,omitempty"  yaml:"environment_names,omitempty" db:"-"  cli:"-"`
	Labels           []Label              `json:"labels,omitempty"  yaml:"labels,omitempty" db:"-"  cli:"-"`
	Permission       int                  `json:"permission"  yaml:"-" db:"-"  cli:"-"`
	Created          time.Time            `json:"created"  yaml:"created" db:"created" `
	LastModified     time.Time            `json:"last_modified"  yaml:"last_modified" db:"last_modified"`
	Metadata         Metadata             `json:"metadata" yaml:"metadata" db:"-" cli:"-"`
	Keys             []ProjectKey         `json:"keys" yaml:"keys" db:"-" cli:"-"`
	VCSServers       []ProjectVCSServer   `json:"vcs_servers" yaml:"vcs_servers" db:"-" cli:"-"`
	Integrations     []ProjectIntegration `json:"integrations" yaml:"integrations" db:"-" cli:"-"`
	Features         map[string]bool      `json:"features" yaml:"features" db:"-" cli:"-"`
	Favorite         bool                 `json:"favorite" yaml:"favorite" db:"-" cli:"favorite"`
	URLs             URL                  `json:"urls" yaml:"-" db:"-" cli:"-"`
}

type URL struct {
	APIURL string `json:"api_url"`
	UIURL  string `json:"ui_url"`
}

// SetApplication data on project
func (proj *Project) SetApplication(app Application) {
	found := false
	for i, a := range proj.Applications {
		if a.Name == app.Name {
			proj.Applications[i] = app
			found = true
			break
		}
	}
	if !found {
		proj.Applications = append(proj.Applications, app)
	}
}

// SetEnvironment data on project
func (proj *Project) SetEnvironment(env Environment) {
	found := false
	for i, e := range proj.Environments {
		if e.Name == env.Name {
			proj.Environments[i] = env
			found = true
			break
		}
	}
	if !found {
		proj.Environments = append(proj.Environments, env)
	}
}

// SetPipeline data on project
func (proj *Project) SetPipeline(pip Pipeline) {
	found := false
	for i, p := range proj.Pipelines {
		if p.Name == pip.Name {
			proj.Pipelines[i] = pip
			found = true
			break
		}
	}
	if !found {
		proj.Pipelines = append(proj.Pipelines, pip)
	}
}

// IsValid returns error if the project is not valid.
func (proj Project) IsValid() error {
	if !NamePatternRegex.MatchString(proj.Key) {
		return NewError(ErrInvalidName, fmt.Errorf("Invalid project key. It should match %s", NamePattern))
	}

	if proj.Icon != "" {
		if !strings.HasPrefix(proj.Icon, IconFormat) {
			return ErrIconBadFormat
		}
		if len(proj.Icon) > MaxIconSize {
			return ErrIconBadSize
		}
	}

	return nil
}

// GetSSHKey returns a ssh key given his name
func (proj Project) GetSSHKey(name string) *ProjectKey {
	for _, k := range proj.Keys {
		if k.Type == KeyTypeSSH && k.Name == name {
			return &k
		}
	}
	return nil
}

// SSHKeys returns the slice of ssh key for an application
func (proj Project) SSHKeys() []ProjectKey {
	keys := []ProjectKey{}
	for _, k := range proj.Keys {
		if k.Type == KeyTypeSSH {
			keys = append(keys, k)
		}
	}
	return keys
}

// PGPKeys returns the slice of pgp key for a project
func (proj Project) PGPKeys() []ProjectKey {
	keys := []ProjectKey{}
	for _, k := range proj.Keys {
		if k.Type == KeyTypePGP {
			keys = append(keys, k)
		}
	}
	return keys
}

// GetIntegration returns the ProjectIntegration given a name
func (proj Project) GetIntegration(pfName string) (ProjectIntegration, bool) {
	for i := range proj.Integrations {
		if proj.Integrations[i].Name == pfName {
			return proj.Integrations[i], true
		}
	}
	return ProjectIntegration{}, false
}

// GetIntegrationByID returns the ProjectIntegration given a name
func (proj Project) GetIntegrationByID(id int64) *ProjectIntegration {
	for i := range proj.Integrations {
		if proj.Integrations[i].ID == id {
			return &proj.Integrations[i]
		}
	}
	return nil
}

// ProjectVCSServer represents associations between a project and a vcs server
type ProjectVCSServer struct {
	Name     string            `json:"name" yaml:"name" db:"-" cli:"name"`
	Username string            `json:"username" yaml:"username" db:"-" cli:"username"`
	Data     map[string]string `json:"-" yaml:"data" db:"-" cli:"-"`
}

// Hash creating a unique hash value
func (vcs ProjectVCSServer) Hash() uint64 {
	hash, _ := hashstructure.Hash(vcs, nil)
	return hash
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
	// ProjectIntegrationsLastModificationType represent key for last update event about project.integrations (add, update, delete a integration)
	ProjectIntegrationsLastModificationType = "project.integrations"
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

// ProjectsToIDs returns ids of given projects.
func ProjectsToIDs(ps []Project) []int64 {
	ids := make([]int64, len(ps))
	for i := range ps {
		ids[i] = ps[i].ID
	}
	return ids
}
