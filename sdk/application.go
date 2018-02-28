package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Repository structs contains all needed information about a single repository
type Repository struct {
	URL  string
	Hook bool
}

// Application represent an application in a project
type Application struct {
	ID                 int64                 `json:"id" db:"id"`
	Name               string                `json:"name" db:"name" cli:"name,key"`
	Description        string                `json:"description"  db:"description"`
	ProjectID          int64                 `json:"-" db:"project_id"`
	ProjectKey         string                `json:"project_key" db:"-"`
	ApplicationGroups  []GroupPermission     `json:"groups,omitempty" db:"-"`
	Variable           []Variable            `json:"variables,omitempty" db:"-"`
	Pipelines          []ApplicationPipeline `json:"pipelines,omitempty" db:"-"`
	PipelinesBuild     []PipelineBuild       `json:"pipelines_build,omitempty" db:"-"`
	Permission         int                   `json:"permission" db:"-"`
	Notifications      []UserNotification    `json:"notifications,omitempty" db:"-"`
	LastModified       time.Time             `json:"last_modified" db:"last_modified"`
	VCSServer          string                `json:"vcs_server,omitempty" db:"vcs_server"`
	RepositoryFullname string                `json:"repository_fullname,omitempty" db:"repo_fullname"`
	RepositoryPollers  []RepositoryPoller    `json:"pollers,omitempty" db:"-"`
	RepositoryStrategy RepositoryStrategy    `json:"vcs_strategy,omitempty" db:"-"`
	Hooks              []Hook                `json:"hooks,omitempty" db:"-"`
	Workflows          []CDPipeline          `json:"workflows,omitempty" db:"-"`
	Schedulers         []PipelineScheduler   `json:"schedulers,omitempty" db:"-"`
	Metadata           Metadata              `json:"metadata" yaml:"metadata" db:"-"`
	WorkflowMigration  string                `json:"workflow_migration" yaml:"workflow_migration" db:"workflow_migration"`
	Keys               []ApplicationKey      `json:"keys" yaml:"keys" db:"-"`
	Usage              *Usage                `json:"usage,omitempty" db:"-" cli:"-"`
}

// SSHKeys returns the slide of ssh key for an application
func (a Application) SSHKeys() []ApplicationKey {
	keys := []ApplicationKey{}
	for _, k := range a.Keys {
		if k.Type == KeyTypeSSH {
			keys = append(keys, k)
		}
	}
	return keys
}

// PGPKeys returns the slide of pgp key for an application
func (a Application) PGPKeys() []ApplicationKey {
	keys := []ApplicationKey{}
	for _, k := range a.Keys {
		if k.Type == KeyTypePGP {
			keys = append(keys, k)
		}
	}
	return keys
}

// RepositoryStrategy represent the way to use the repository
type RepositoryStrategy struct {
	ConnectionType string `json:"connection_type"`
	SSHKey         string `json:"ssh_key"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Branch         string `json:"branch"`
	DefaultBranch  string `json:"default_branch"`
	PGPKey         string `json:"pgp_key"`
}

// ApplicationVariableAudit represents an audit on an application variable
type ApplicationVariableAudit struct {
	ID             int64     `json:"id" yaml:"-" db:"id"`
	ApplicationID  int64     `json:"application_id" yaml:"-" db:"application_id"`
	VariableID     int64     `json:"variable_id" yaml:"-" db:"variable_id"`
	Type           string    `json:"type" yaml:"-" db:"type"`
	VariableBefore *Variable `json:"variable_before,omitempty" yaml:"-" db:"-"`
	VariableAfter  *Variable `json:"variable_after,omitempty" yaml:"-" db:"-"`
	Versionned     time.Time `json:"versionned" yaml:"-" db:"versionned"`
	Author         string    `json:"author" yaml:"-" db:"author"`
}

// ApplicationPipeline Represent the link between an application and a pipeline
type ApplicationPipeline struct {
	ID           int64             `json:"id"`
	Pipeline     Pipeline          `json:"pipeline"`
	Parameters   []Parameter       `json:"parameters"`
	LastModified int64             `json:"last_modified"`
	Triggers     []PipelineTrigger `json:"triggers,omitempty"`
}

// NewApplication instanciate a new NewApplication
func NewApplication(name string) *Application {
	a := &Application{
		Name: name,
	}
	return a
}

// AddApplication create an application in the given project
func AddApplication(key, appName string) error {

	a := NewApplication(appName)
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/applications", key)
	data, _, err = Request("POST", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// ListApplications returns all available application for the given project
func ListApplications(key string) ([]Application, error) {
	url := fmt.Sprintf("/project/%s/applications", key)
	data, _, err := Request("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var applications []Application
	if err := json.Unmarshal(data, &applications); err != nil {
		return nil, err
	}

	return applications, nil
}

// GetApplicationOptions are options for GetApplication
var GetApplicationOptions = struct {
	WithPollers    RequestModifier
	WithHooks      RequestModifier
	WithNotifs     RequestModifier
	WithWorkflow   RequestModifier
	WithTriggers   RequestModifier
	WithSchedulers RequestModifier
}{
	WithPollers: func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withPollers", "true")
		r.URL.RawQuery = q.Encode()
	},
	WithHooks: func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withHooks", "true")
		r.URL.RawQuery = q.Encode()
	},
	WithNotifs: func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withNotifs", "true")
		r.URL.RawQuery = q.Encode()
	},
	WithWorkflow: func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withWorkflow", "true")
		r.URL.RawQuery = q.Encode()
	},
	WithTriggers: func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withTriggers", "true")
		r.URL.RawQuery = q.Encode()
	},
	WithSchedulers: func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withSchedulers", "true")
		r.URL.RawQuery = q.Encode()
	},
}

// GetApplication retrieve the given application from CDS
func GetApplication(pk, name string, opts ...RequestModifier) (*Application, error) {
	var a Application

	path := fmt.Sprintf("/project/%s/application/%s", pk, name)
	data, _, err := Request("GET", path, nil, opts...)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

// UpdateApplication update an application in CDS
func UpdateApplication(app *Application) error {
	data, err := json.Marshal(app)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/application/%s", app.ProjectKey, app.Name)
	data, _, err = Request("PUT", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RenameApplication renames an application from CDS
func RenameApplication(pk, name, newName string) error {
	app := NewApplication(newName)

	data, err := json.Marshal(app)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/application/%s", pk, name)
	data, _, err = Request("PUT", url, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// DeleteApplication delete an application from CDS
func DeleteApplication(pk, name string) error {

	path := fmt.Sprintf("/project/%s/application/%s", pk, name)
	_, _, err := Request("DELETE", path, nil)
	return err
}

// ShowApplicationVariable  show variables for an application
func ShowApplicationVariable(projectKey, appName string) ([]Variable, error) {

	path := fmt.Sprintf("/project/%s/application/%s/variable", projectKey, appName)
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

// AddApplicationVariable  add a variable in an application
func AddApplicationVariable(projectKey, appName, varName, varValue string, varType string) error {
	newVar := Variable{
		Name:  varName,
		Value: varValue,
		Type:  varType,
	}

	data, err := json.Marshal(newVar)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/application/%s/variable/%s", projectKey, appName, varName)
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// GetVariableInApplication Get a variable in the given application
func GetVariableInApplication(projectKey, appName, name string) (*Variable, error) {
	var v Variable

	path := fmt.Sprintf("/project/%s/application/%s/variable/%s", projectKey, appName, name)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if e := DecodeError(data); e != nil {
		return nil, e
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	return &v, nil
}

// UpdateApplicationVariable update a variable in an application
func UpdateApplicationVariable(projectKey, appName, oldName, varName, varValue, varType string) error {
	oldVar, err := GetVariableInApplication(projectKey, appName, oldName)
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

	path := fmt.Sprintf("/project/%s/application/%s/variable/%s", projectKey, appName, varName)
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// RemoveApplicationVariable  remove a variable from an application
func RemoveApplicationVariable(projectKey, appName, varName string) error {
	path := fmt.Sprintf("/project/%s/application/%s/variable/%s", projectKey, appName, varName)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// RemoveGroupFromApplication  call api to remove a group from the given application
func RemoveGroupFromApplication(projectKey, appName, groupName string) error {
	path := fmt.Sprintf("/project/%s/application/%s/group/%s", projectKey, appName, groupName)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// UpdateGroupInApplication  call api to update group permission for the given application
func UpdateGroupInApplication(projectKey, appName, groupName string, permission int) error {
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

	path := fmt.Sprintf("/project/%s/application/%s/group/%s", projectKey, appName, groupName)
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// AddGroupInApplication  add a group in an application
func AddGroupInApplication(projectKey, appName, groupName string, permission int) error {
	if permission < 4 || permission > 7 {
		return fmt.Errorf("Permission should be between 4-7 ")
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

	path := fmt.Sprintf("/project/%s/application/%s/group", projectKey, appName)
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// ListApplicationPipeline  list all pipelines attached to the application
func ListApplicationPipeline(projectKey, appName string) ([]Pipeline, error) {
	var pipelines []Pipeline
	path := fmt.Sprintf("/project/%s/application/%s/pipeline", projectKey, appName)
	data, _, errReq := Request("GET", path, nil)
	if errReq != nil {
		return nil, errReq
	}
	if e := DecodeError(data); e != nil {
		return nil, e
	}

	if err := json.Unmarshal(data, &pipelines); err != nil {
		return nil, err
	}

	for i, pip := range pipelines {
		pip2, err := GetPipeline(projectKey, pip.Name)
		if err != nil {
			return nil, err
		}
		pipelines[i] = *pip2
	}

	return pipelines, nil
}

// AttachPipeline allows pipeline to be used in application context
func AttachPipeline(key, app, pip string) error {
	return AddApplicationPipeline(key, app, pip)
}

// AddApplicationPipeline  add a pipeline in an application
func AddApplicationPipeline(projectKey, appName, pipelineName string) error {
	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s", projectKey, appName, pipelineName)
	data, _, err := Request("POST", path, nil)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// UpdateApplicationPipeline  add a pipeline in an application
func UpdateApplicationPipeline(projectKey, appName, pipelineName string, params []Parameter) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s", projectKey, appName, pipelineName)
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

// RemoveApplicationPipeline  remove a pipeline from an application
func RemoveApplicationPipeline(projectKey, appName, pipelineName string) error {
	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s", projectKey, appName, pipelineName)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}
	return DecodeError(data)
}

//GetPipelineScheduler returns all pipeline scheduler
func GetPipelineScheduler(projectKey, appName, pipelineName string) ([]PipelineScheduler, error) {
	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/scheduler", projectKey, appName, pipelineName)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if err := DecodeError(data); err != nil {
		return nil, err
	}

	ps := []PipelineScheduler{}
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, err
	}

	return ps, nil
}

//AddPipelineScheduler add a pipeline scheduler
func AddPipelineScheduler(projectKey, appName, pipelineName, cronExpr, envName string, params []Parameter) (*PipelineScheduler, error) {
	s := PipelineScheduler{
		Crontab: cronExpr,
		Args:    params,
	}

	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/scheduler", projectKey, appName, pipelineName)
	if envName != "" {
		path = path + "?envName=" + url.QueryEscape(envName)
	}
	data, _, err := Request("POST", path, b)
	if err != nil {
		return nil, err
	}

	if err := DecodeError(data); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

//UpdatePipelineScheduler update a pipeline scheduler
func UpdatePipelineScheduler(projectKey, appName, pipelineName string, s *PipelineScheduler) (*PipelineScheduler, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/scheduler", projectKey, appName, pipelineName)
	data, _, err := Request("PUT", path, b)
	if err != nil {
		return nil, err
	}

	if err := DecodeError(data); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}

	return s, nil
}

//DeletePipelineScheduler update a pipeline scheduler
func DeletePipelineScheduler(projectKey, appName, pipelineName string, s *PipelineScheduler) error {
	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/scheduler/%d", projectKey, appName, pipelineName, s.ID)
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
}
