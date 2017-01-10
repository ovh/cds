package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Repository structs contains all needed information about a single repository
type Repository struct {
	URL  string
	Hook bool
}

// Application represent an application in a project
type Application struct {
	ID                  int64                 `json:"id"`
	Name                string                `json:"name"`
	ProjectKey          string                `json:"project_key"`
	ApplicationGroups   []GroupPermission     `json:"groups,omitempty"`
	Variable            []Variable            `json:"variables,omitempty"`
	Pipelines           []ApplicationPipeline `json:"pipelines,omitempty"`
	PipelinesBuild      []PipelineBuild       `json:"pipelines_build,omitempty"`
	Permission          int                   `json:"permission"`
	Notifications       []UserNotification    `json:"notifications,omitempty"`
	LastModified        int64                 `json:"last_modified"`
	RepositoriesManager *RepositoriesManager  `json:"repositories_manager,omitempty"`
	RepositoryFullname  string                `json:"repository_fullname,omitempty"`
	RepositoryPollers   []RepositoryPoller    `json:"pollers,omitempty"`
	Hooks               []Hook                `json:"hooks,omitempty"`
	Workflows           []CDPipeline          `json:"workflows,omitempty"`
}

// ApplicationPipeline Represent the link between an application and a pipeline
type ApplicationPipeline struct {
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

// ListApplications returns all available application for the given project
func ListApplications(key string) ([]Application, error) {

	url := fmt.Sprintf("/project/%s/applications", key)
	data, code, err := Request("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var applications []Application
	err = json.Unmarshal(data, &applications)
	if err != nil {
		return nil, err
	}

	return applications, nil
}

// GetApplication retrieve the given application from CDS
func GetApplication(pk, name string) (*Application, error) {
	var a Application

	path := fmt.Sprintf("/project/%s/application/%s", pk, name)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &a)
	if err != nil {
		return nil, err
	}

	return &a, nil
}

// RenameApplication renames an application from CDS
func RenameApplication(pk, name, newName string) error {
	app := NewApplication(newName)

	data, err := json.Marshal(app)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/application/%s", pk, name)
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

// DeleteApplication delete an application from CDS
func DeleteApplication(pk, name string) error {

	path := fmt.Sprintf("/project/%s/application/%s", pk, name)
	_, code, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// ShowApplicationVariable  show variables for an application
func ShowApplicationVariable(projectKey, appName string) ([]Variable, error) {

	path := fmt.Sprintf("/project/%s/application/%s/variable", projectKey, appName)
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

// AddApplicationVariable  add a variable in an application
func AddApplicationVariable(projectKey, appName, varName, varValue string, varType VariableType) error {

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

// GetVariableInApplication Get a variable in the given application
func GetVariableInApplication(projectKey, appName, name string) (*Variable, error) {
	var v Variable

	path := fmt.Sprintf("/project/%s/application/%s/variable/%s", projectKey, appName, name)
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
		Type:  VariableTypeFromString(varType),
	}

	data, err := json.Marshal(newVar)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/application/%s/variable/%s", projectKey, appName, varName)
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

// RemoveApplicationVariable  remove a variable from an application
func RemoveApplicationVariable(projectKey, appName, varName string) error {
	path := fmt.Sprintf("/project/%s/application/%s/variable/%s", projectKey, appName, varName)
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

// RemoveGroupFromApplication  call api to remove a group from the given application
func RemoveGroupFromApplication(projectKey, appName, groupName string) error {

	path := fmt.Sprintf("/project/%s/application/%s/group/%s", projectKey, appName, groupName)
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

// ListApplicationPipeline  list all pipelines attached to the application
func ListApplicationPipeline(projectKey, appName string) ([]Pipeline, error) {
	var pipelines []Pipeline
	path := fmt.Sprintf("/project/%s/application/%s/pipeline", projectKey, appName)
	data, code, errReq := Request("GET", path, nil)
	if errReq != nil {
		return nil, errReq
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
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
	data, code, err := Request("POST", path, nil)
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

// UpdateApplicationPipeline  add a pipeline in an application
func UpdateApplicationPipeline(projectKey, appName, pipelineName string, params []Parameter) error {

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s", projectKey, appName, pipelineName)
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

// RemoveApplicationPipeline  remove a pipeline from an application
func RemoveApplicationPipeline(projectKey, appName, pipelineName string) error {
	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s", projectKey, appName, pipelineName)
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

//GetPipelineScheduler returns all pipeline scheduler
func GetPipelineScheduler(projectKey, appName, pipelineName string) ([]PipelineScheduler, error) {
	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/scheduler", projectKey, appName, pipelineName)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
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
		path = path + url.QueryEscape("?envName="+envName)
	}
	data, code, err := Request("POST", path, b)
	if err != nil {
		return nil, err
	}

	if err := DecodeError(data); err != nil {
		return nil, err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
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
	data, code, err := Request("PUT", path, b)
	if err != nil {
		return nil, err
	}

	if err := DecodeError(data); err != nil {
		return nil, err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}

	return s, nil
}

//DeletePipelineScheduler update a pipeline scheduler
func DeletePipelineScheduler(projectKey, appName, pipelineName string, s *PipelineScheduler) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/scheduler", projectKey, appName, pipelineName)
	data, code, err := Request("DELETE", path, b)
	if err != nil {
		return err
	}

	if err := DecodeError(data); err != nil {
		return err
	}

	if code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}

	return nil
}
