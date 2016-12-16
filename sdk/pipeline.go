package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Pipeline represents the complete behavior of CDS for each projects
type Pipeline struct {
	ID                  int64             `json:"id" yaml:"-"`
	Name                string            `json:"name"`
	Type                PipelineType      `json:"type"`
	ProjectKey          string            `json:"projectKey"`
	ProjectID           int64             `json:"-"`
	LastPipelineBuild   *PipelineBuild    `json:"last_pipeline_build"`
	Stages              []Stage           `json:"stages"`
	GroupPermission     []GroupPermission `json:"groups,omitempty"`
	Parameter           []Parameter       `json:"parameters,omitempty"`
	AttachedApplication []Application     `json:"attached_application,omitempty"`
	Permission          int               `json:"permission"`
	LastModified        int64             `json:"last_modified"`
}

// PipelineBuild Struct for history table
type PipelineBuild struct {
	ID          int64       `json:"id"`
	BuildNumber int64       `json:"build_number"`
	Version     int64       `json:"version"`
	Parameters  []Parameter `json:"parameters"`
	Status      Status      `json:"status"`
	Start       time.Time   `json:"start,omitempty"`
	Done        time.Time   `json:"done,omitempty"`
	Stages      []Stage     `json:"stages"`

	Pipeline    Pipeline    `json:"pipeline"`
	Application Application `json:"application"`
	Environment Environment `json:"environment"`

	Artifacts             []Artifact           `json:"artifacts,omitempty"`
	Tests                 *Tests               `json:"tests,omitempty"`
	Commits               []VCSCommit          `json:"commits,omitempty"`
	Trigger               PipelineBuildTrigger `json:"trigger"`
	PreviousPipelineBuild *PipelineBuild       `json:"previous_pipeline_build"`
}

// PipelineBuildTrigger Struct for history table
type PipelineBuildTrigger struct {
	ManualTrigger       bool           `json:"manual_trigger"`
	TriggeredBy         *User          `json:"triggered_by"`
	ParentPipelineBuild *PipelineBuild `json:"parent_pipeline_build"`
	VCSChangesBranch    string         `json:"vcs_branch"`
	VCSChangesHash      string         `json:"vcs_hash"`
	VCSChangesAuthor    string         `json:"vcs_author"`
}

// PipelineType defines the purpose of a given pipeline
type PipelineType string

// Different types of Pipeline
const (
	BuildPipeline      PipelineType = "build"
	DeploymentPipeline PipelineType = "deployment"
	TestingPipeline    PipelineType = "testing"
)

// AvailablePipelineType List of all pipeline type
var AvailablePipelineType = []string{
	string(BuildPipeline),
	string(DeploymentPipeline),
	string(TestingPipeline),
}

// PipelineTypeFromString returns the proper PipelineType
func PipelineTypeFromString(in string) PipelineType {
	switch in {
	case string(BuildPipeline):
		return BuildPipeline
	case string(DeploymentPipeline):
		return DeploymentPipeline
	case string(TestingPipeline):
		return TestingPipeline
	default:
		return BuildPipeline
	}
}

// PipelineAction represents an action in a pipeline
type PipelineAction struct {
	ActionName      string      `json:"actionName"`
	Args            []Parameter `json:"args"`
	PipelineStageID int64       `json:"pipeline_stage_id"`
}

// CDPipeline  Represent a pipeline in the CDTree
type CDPipeline struct {
	Project      Project         `json:"project"`
	Application  Application     `json:"application"`
	Environment  Environment     `json:"environment"`
	Pipeline     Pipeline        `json:"pipeline"`
	SubPipelines []CDPipeline    `json:"subPipelines"`
	Trigger      PipelineTrigger `json:"trigger"`
}

// RunRequest  Request to run a pipeline
type RunRequest struct {
	Params              []Parameter `json:"parameters,omitempty"`
	Env                 Environment `json:"env,omitempty"`
	ParentBuildNumber   int64       `json:"parent_build_number,omitempty"`
	ParentPipelineID    int64       `json:"parent_pipeline_id,omitempty"`
	ParentEnvironmentID int64       `json:"parent_environment_id,omitempty"`
	ParentApplicationID int64       `json:"parent_application_id,omitempty"`
}

// ListPipelines retrieves all available pipelines to called
func ListPipelines(projectKey string) ([]Pipeline, error) {
	url := fmt.Sprintf("/project/%s/pipeline", projectKey)

	data, _, err := Request("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var pip []Pipeline
	err = json.Unmarshal(data, &pip)
	if err != nil {
		return nil, err
	}

	return pip, nil
}

// GetPipeline retrieves pipeline definition from CDS
func GetPipeline(key, name string) (*Pipeline, error) {

	path := fmt.Sprintf("/project/%s/pipeline/%s", key, name)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	p := &Pipeline{}
	err = json.Unmarshal(data, p)
	if err != nil {
		return nil, err
	}

	p.ProjectKey = key
	return p, err
}

// AddPipeline creates a new empty pipeline
func AddPipeline(name string, projectKey string, pipelineType PipelineType, params []Parameter) error {

	p := Pipeline{
		Name:       name,
		ProjectKey: projectKey,
		Type:       pipelineType,
		Parameter:  params,
	}

	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/project/%s/pipeline", projectKey)
	_, code, err := Request("POST", url, data)
	if err != nil {
		return err
	}
	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// DeletePipelineAction delete the given action from the given pipeline
func DeletePipelineAction(projectKey string, pipelineName string, actionPipelineID int64) error {
	path := fmt.Sprintf("/project/%s/pipeline/%s/action/%d", projectKey, pipelineName, actionPipelineID)
	data, code, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}
	if code != http.StatusCreated && code != http.StatusOK {
		return fmt.Errorf("Error [%d]: %s", code, data)
	}
	return nil
}

// MoveActionInPipeline Move an action in a pipeline
func MoveActionInPipeline(projectKey, pipelineName string, actionPipelineID int64, newOrder int) error {

	pipeline, err := GetPipeline(projectKey, pipelineName)
	if err != nil {
		return err
	}
	var stageID int64
	var action Action
	for _, stage := range pipeline.Stages {
		if stage.BuildOrder == newOrder {
			stageID = stage.ID
		}
		for _, actionInStage := range stage.Actions {
			if actionInStage.PipelineActionID == actionPipelineID {
				action = actionInStage
			}
		}
	}

	if stageID != 0 && action.ID != 0 {
		action.PipelineStageID = stageID

		data, err := json.Marshal(action)
		if err != nil {
			return err
		}

		path := fmt.Sprintf("/project/%s/pipeline/%s/action/%d", projectKey, pipelineName, actionPipelineID)

		_, _, err = Request("PUT", path, data)
		if err != nil {
			return err
		}
		e := DecodeError(data)
		if e != nil {
			return e
		}

		return nil
	}
	return fmt.Errorf("Action or stage not found")
}

// RestartPipeline will have two distinct behavior:
// - If the pipeline build result is failed, it will only restart failed actions
// - If the pipeline build result is success, it will restart all actions
func RestartPipeline(key, app, pip, env string, bn int) (chan Log, error) {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/restart", key, app, pip, bn)

	_, code, err := Request("POST", uri, nil)
	if err != nil {
		return nil, err
	}
	if code > 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	return StreamPipelineBuild(key, app, pip, env, bn, false)
}

//GetPipelineCommits returns list of commit between this build and the previous
//one the same branch. If previous build is not available, it returns only the
//last commit for the branch
func GetPipelineCommits(key, app, pip, env string, bn int) ([]VCSCommit, error) {
	commits := []VCSCommit{}
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/commits?envName=%s", key, app, pip, bn, env)
	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return commits, err
	}
	if code > 300 {
		return commits, fmt.Errorf("HTTP %d", code)
	}

	json.Unmarshal([]byte(data), &commits)
	return commits, nil
}

// RunPipeline trigger a CDS pipeline
func RunPipeline(key, appName, name, env string, stream bool, request RunRequest, followTriggers bool) (chan Log, error) {

	request.Env = Environment{Name: env}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/run", key, appName, name)
	_, code, err := Request("POST", path, data)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	if stream {
		return StreamPipelineBuild(key, appName, name, env, 0, followTriggers)
	}
	return nil, nil
}

// GetPipelineBuildHistory retrieves recent build history for given pipeline
func GetPipelineBuildHistory(key, appName, name, env string) ([]PipelineBuild, error) {
	var res []PipelineBuild

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/history", key, appName, name)
	if env != "" {
		path = fmt.Sprintf("%s?envName=%s", path, env)
	}
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal([]byte(data), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetBuildLogs retrieve all output from given build
func GetBuildLogs(key, pipelineName, env string, buildID int) ([]Log, error) {
	var logs []Log
	var path string

	if buildID == 0 {
		path = fmt.Sprintf("/project/%s/pipeline/%s/build/last/log", key, pipelineName)
	} else {
		path = fmt.Sprintf("/project/%s/pipeline/%s/build/%d/log", key, pipelineName, buildID)
	}

	if env != "" {
		path = fmt.Sprintf("%s?envName=%s", path, env)
	}

	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(data), &logs)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

// StreamPipelineBuild poll the api to fetch logs of building pipeline and push them in returned channel
func StreamPipelineBuild(key, appName, pipelineName, env string, buildID int, followTrigger bool) (chan Log, error) {
	ch := make(chan Log)
	var logs []Log
	const LogLimit = 10000

	var path string

	if buildID == 0 {
		path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/last/log", key, appName, pipelineName)
	} else {
		path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/log", key, appName, pipelineName, buildID)
	}

	if env != "" {
		path = fmt.Sprintf("%s?envName=%s", path, env)
	}

	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(data), &logs)
	if err != nil {
		return nil, err
	}

	var lastID int64
	go func() {
		for {

			if buildID == 0 {
				path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/last/log?offset=%d", key, appName, pipelineName, lastID)
			} else {
				path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/log?offset=%d", key, appName, pipelineName, buildID, lastID)
			}
			if env != "" {
				path = fmt.Sprintf("%s&envName=%s", path, env)
			}

			data, _, err := Request("GET", path, nil)
			if err != nil {
				close(ch)
				return
			}

			err = json.Unmarshal([]byte(data), &logs)
			if err != nil {
				close(ch)
				return
			}

			for i := range logs {
				if logs[i].ID > 0 {
					lastID = logs[i].ID
				}
				if logs[i].ID != 0 || len(logs) < LogLimit {
					ch <- logs[i]
				}

				if logs[i].ID == 0 && len(logs) < LogLimit {
					//Before closing the channel, check if we want to  follower triggers
					if followTrigger {
						wg := &sync.WaitGroup{}
						//Get child triggers
						triggers, err := GetTriggersAsSource(key, appName, pipelineName, env)
						if err == nil && len(triggers) > 0 {
							for _, t := range triggers {
								//If there is any trigger, stream each of them
								triggerCh, err := StreamPipelineBuild(t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name, 0, followTrigger)
								if err == nil {
									wg.Add(1)
									go func(mainCh, triggerCh chan Log) {
										//Get log from the trigger's channel and push it to the main channel
										for l := range triggerCh {
											ch <- l
										}
										wg.Done()
									}(ch, triggerCh)
								}
							}
						}
						//When all of the triggers are done, close the main channel
						wg.Wait()
					}
					close(ch)
					return
				}
			}

			if len(logs) < LogLimit {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return ch, nil
}

// DeletePipeline remove given pipeline from CDS
func DeletePipeline(key, name string) error {
	path := fmt.Sprintf("/project/%s/pipeline/%s", key, name)

	_, code, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// RemoveGroupFromPipeline  call api to remove a group from the given pipeline
func RemoveGroupFromPipeline(projectKey, pipelineName, groupName string) error {

	path := fmt.Sprintf("/project/%s/pipeline/%s/group/%s", projectKey, pipelineName, groupName)
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

// UpdateGroupInPipeline  call api to update group permission on pipeline
func UpdateGroupInPipeline(projectKey, pipelineName, groupName string, permission int) error {

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

	path := fmt.Sprintf("/project/%s/pipeline/%s/group/%s", projectKey, pipelineName, groupName)
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

// AddGroupInPipeline  add a group in a pipeline
func AddGroupInPipeline(projectKey, pipelineName, groupName string, permission int) error {

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

	path := fmt.Sprintf("/project/%s/pipeline/%s/group", projectKey, pipelineName)
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

// ShowParameterInPipeline  show parameters for a pipeline
func ShowParameterInPipeline(projectKey, pipelineName string) ([]Parameter, error) {

	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter", projectKey, pipelineName)
	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusCreated && code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var params []Parameter
	err = json.Unmarshal(data, &params)
	if err != nil {
		return nil, err
	}
	return params, nil
}

// AddParameterInPipeline  add a variable in a pipeline
func AddParameterInPipeline(projectKey, pipelineName, paramName, paramValue, paramType, paramDescription string) error {

	newParam := Parameter{
		Name:        paramName,
		Value:       paramValue,
		Type:        ParameterTypeFromString(paramType),
		Description: paramDescription,
	}

	data, err := json.Marshal(newParam)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter/%s", projectKey, pipelineName, paramName)
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

// UpdateParameterInPipeline update a variable in a pipeline
func UpdateParameterInPipeline(projectKey, pipelineName, paramName, paramValue, paramType, paramDescription string) error {

	newParam := Parameter{
		Name:        paramName,
		Value:       paramValue,
		Type:        ParameterTypeFromString(paramType),
		Description: paramDescription,
	}

	data, err := json.Marshal(newParam)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter/%s", projectKey, pipelineName, paramName)
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

// RemoveParameterFromPipeline  remove a parameter from a pipeline
func RemoveParameterFromPipeline(projectKey, pipelineName, paramName string) error {
	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter/%s", projectKey, pipelineName, paramName)
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

// GetPipelineBuildStatus retrieves current build information.
// With buildNumber at 0, fetch last build
func GetPipelineBuildStatus(proj, app, pip, env string, buildNumber int64) (PipelineBuild, error) {
	var pb PipelineBuild
	var uri string

	if buildNumber == 0 {
		uri = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/last?envName=%s",
			proj, app, pip, env)
	} else {
		uri = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d?envName=%s",
			proj, app, pip, buildNumber, env)
	}

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return pb, err
	}
	if code >= 300 {
		return pb, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &pb)
	if err != nil {
		return pb, err
	}

	return pb, nil
}

// GetBuildingPipelines retrieves all building pipelines
func GetBuildingPipelines() ([]PipelineBuild, error) {

	data, code, err := Request("GET", "/mon/building", nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var pbs []PipelineBuild
	err = json.Unmarshal(data, &pbs)
	if err != nil {
		return nil, err
	}

	return pbs, nil
}

// GetBuildingPipelineByHash retrieves pipeline building a specific commit hash
func GetBuildingPipelineByHash(hash string) ([]PipelineBuild, error) {
	var pbs []PipelineBuild

	data, code, err := Request("GET", "/mon/building/"+hash, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal(data, &pbs)
	if err != nil {
		return nil, err
	}

	return pbs, nil
}
