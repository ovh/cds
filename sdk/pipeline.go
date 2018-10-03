package sdk

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/ovh/venom"
)

// Pipeline represents the complete behavior of CDS for each projects
type Pipeline struct {
	ID                int64             `json:"id" yaml:"-"`
	Name              string            `json:"name" cli:"name"`
	Description       string            `json:"description" cli:"description"`
	Type              string            `json:"type"`
	ProjectKey        string            `json:"projectKey"`
	ProjectID         int64             `json:"-"`
	LastPipelineBuild *PipelineBuild    `json:"last_pipeline_build"`
	Stages            []Stage           `json:"stages"`
	GroupPermission   []GroupPermission `json:"groups,omitempty"`
	Parameter         []Parameter       `json:"parameters,omitempty"`
	Usage             *Usage            `json:"usage,omitempty"`
	Permission        int               `json:"permission"`
	LastModified      int64             `json:"last_modified" cli:"modified"`
}

// PipelineAudit represents pipeline audit
type PipelineAudit struct {
	ID         int64     `json:"id" db:"id"`
	PipelineID int64     `json:"pipeline_id" db:"pipeline_id"`
	UserName   string    `json:"username" db:"username"`
	Versionned time.Time `json:"versionned" db:"versionned"`
	Pipeline   *Pipeline `json:"pipeline" db:"-"`
	Action     string    `json:"action" db:"action"`
}

// PipelineBuild Struct for history table
type PipelineBuild struct {
	ID          int64                  `json:"id"`
	BuildNumber int64                  `json:"build_number"`
	Version     int64                  `json:"version"`
	Parameters  []Parameter            `json:"parameters"`
	Status      Status                 `json:"status"`
	Warnings    []PipelineBuildWarning `json:"warnings"`
	Start       time.Time              `json:"start,omitempty"`
	Done        time.Time              `json:"done,omitempty"`
	Stages      []Stage                `json:"stages"`

	Pipeline    Pipeline    `json:"pipeline"`
	Application Application `json:"application"`
	Environment Environment `json:"environment"`

	Artifacts             []Artifact           `json:"artifacts,omitempty"`
	Tests                 *venom.Tests         `json:"tests,omitempty"`
	Commits               []VCSCommit          `json:"commits,omitempty"`
	Trigger               PipelineBuildTrigger `json:"trigger"`
	PreviousPipelineBuild *PipelineBuild       `json:"previous_pipeline_build"`
}

// PipelineBuildTrigger Struct for history table
type PipelineBuildTrigger struct {
	ScheduledTrigger    bool           `json:"scheduled_trigger"`
	ManualTrigger       bool           `json:"manual_trigger"`
	TriggeredBy         *User          `json:"triggered_by"`
	ParentPipelineBuild *PipelineBuild `json:"parent_pipeline_build"`
	VCSChangesBranch    string         `json:"vcs_branch"`
	VCSChangesHash      string         `json:"vcs_hash"`
	VCSChangesAuthor    string         `json:"vcs_author"`
	VCSRemote           string         `json:"vcs_remote,omitempty"`
	VCSRemoteURL        string         `json:"vcs_remote_url,omitempty"`
}

// PipelineBuildWarning Struct for display warnings about build
type PipelineBuildWarning struct {
	Type   string `json:"type"`
	Action Action `json:"action"`
}

// This constant deals with pipelines
const (
	// Different types of Pipeline
	BuildPipeline      = "build"      // DEPRECATED with workflows
	DeploymentPipeline = "deployment" // DEPRECATED with workflows
	TestingPipeline    = "testing"    // DEPRECATED with workflows
	// Different types of warning for PipelineBuild
	OptionalStepFailed = "optional_step_failed"
)

// AvailablePipelineType List of all pipeline type
var AvailablePipelineType = []string{
	BuildPipeline,
	DeploymentPipeline,
	TestingPipeline,
}

// PipelineAction represents an action in a pipeline
type PipelineAction struct {
	ActionName      string      `json:"actionName"`
	Args            []Parameter `json:"args"`
	PipelineStageID int64       `json:"pipeline_stage_id"`
}

// CDPipeline  Represent a pipeline in the CDTree
type CDPipeline struct {
	Project      Project             `json:"project"`
	Application  Application         `json:"application"`
	Environment  Environment         `json:"environment"`
	Pipeline     Pipeline            `json:"pipeline"`
	SubPipelines []CDPipeline        `json:"subPipelines"`
	Trigger      PipelineTrigger     `json:"trigger"`
	Schedulers   []PipelineScheduler `json:"schedulers"`
	Hooks        []Hook              `json:"hooks"`
}

// RunRequest  Request to run a pipeline
type RunRequest struct {
	Params              []Parameter `json:"parameters,omitempty"`
	Env                 Environment `json:"env,omitempty"`
	ParentBuildNumber   int64       `json:"parent_build_number,omitempty"`
	ParentVersion       int64       `json:"parent_version,omitempty"`
	ParentPipelineID    int64       `json:"parent_pipeline_id,omitempty"`
	ParentEnvironmentID int64       `json:"parent_environment_id,omitempty"`
	ParentApplicationID int64       `json:"parent_application_id,omitempty"`
}

// GetPipeline retrieves pipeline definition from CDS
func GetPipeline(key, name string) (*Pipeline, error) {
	path := fmt.Sprintf("/project/%s/pipeline/%s", key, name)
	data, _, errr := Request("GET", path, nil)
	if errr != nil {
		return nil, errr
	}

	p := &Pipeline{}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}

	p.ProjectKey = key
	return p, nil
}

// AddPipeline creates a new empty pipeline
func AddPipeline(name string, projectKey string, pipelineType string, params []Parameter) error {
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
	_, _, err = Request("POST", url, data)
	return err
}

//GetPipelineCommits returns list of commit between this build and the previous
//one the same branch. If previous build is not available, it returns only the
//last commit for the branch
func GetPipelineCommits(key, app, pip, env string, bn int) ([]VCSCommit, error) {
	commits := []VCSCommit{}
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/commits?envName=%s", key, app, pip, bn, url.QueryEscape(env))
	data, _, err := Request("GET", uri, nil)
	if err != nil {
		return commits, err
	}
	json.Unmarshal([]byte(data), &commits)
	return commits, nil
}

// GetPipelineBuildHistory retrieves recent build history for given pipeline
func GetPipelineBuildHistory(key, appName, name, env, buildNumber string) ([]PipelineBuild, error) {
	var res []PipelineBuild

	path := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/history", key, appName, name)
	sep := "?"
	if env != "" {
		path = fmt.Sprintf("%s%senvName=%s", path, sep, url.QueryEscape(env))
		sep = "&"
	}
	if buildNumber != "" {
		path = fmt.Sprintf("%s%sbuildNumber=%s", path, sep, url.QueryEscape(buildNumber))
	}
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// StreamPipelineBuild poll the api to fetch logs of building pipeline and push them in returned channel
func StreamPipelineBuild(key, appName, pipelineName, env string, buildID int, followTrigger bool) (chan Log, error) {
	ch := make(chan Log)
	go func() {
		var path string
		var logs []Log
		currentStep := 0
		currentStepPosition := 0
		for {

			if buildID == 0 {
				path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/last/log", key, appName, pipelineName)
			} else {
				path = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/log", key, appName, pipelineName, buildID)
			}
			if env != "" {
				path = fmt.Sprintf("%s?envName=%s", path, url.QueryEscape(env))
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

			totalStepsReturn := len(logs)
			if totalStepsReturn > 0 {
				// remove old step
				logs = logs[currentStep:]

				// remove line already displayed on current step
				if currentStepPosition <= len(logs[0].Val) {
					logs[0].Val = logs[0].Val[currentStepPosition:]
				}

				// Update data

				// If stay on same stage
				if currentStep == totalStepsReturn-1 {
					currentStepPosition += len(logs[len(logs)-1].Val)
				} else {
					currentStepPosition = len(logs[len(logs)-1].Val)
				}
				currentStep = totalStepsReturn - 1

				for i := range logs {
					ch <- logs[i]
					if logs[i].Id != 0 {
						continue
					}

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
			time.Sleep(1 * time.Second)
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
	data, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// UpdateGroupInPipeline  call api to update group permission on pipeline
func UpdateGroupInPipeline(projectKey, pipelineName, groupName string, permission int) error {
	if permission < 4 || permission > 7 {
		return fmt.Errorf("permission should be between 4-7")
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
	data, _, err = Request("PUT", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// AddGroupInPipeline  add a group in a pipeline
func AddGroupInPipeline(projectKey, pipelineName, groupName string, permission int) error {
	if permission < 4 || permission > 7 {
		return fmt.Errorf("permission should be between 4-7")
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
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// ShowParameterInPipeline  show parameters for a pipeline
func ShowParameterInPipeline(projectKey, pipelineName string) ([]Parameter, error) {
	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter", projectKey, pipelineName)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var params []Parameter
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}
	return params, nil
}

// AddParameterInPipeline  add a variable in a pipeline
func AddParameterInPipeline(projectKey, pipelineName, paramName, paramValue, paramType, paramDescription string) error {
	newParam := Parameter{
		Name:        paramName,
		Value:       paramValue,
		Type:        paramType,
		Description: paramDescription,
	}

	data, err := json.Marshal(newParam)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter/%s", projectKey, pipelineName, paramName)
	data, _, err = Request("POST", path, data)
	if err != nil {
		return err
	}

	return DecodeError(data)
}

// UpdateParameterInPipeline update a variable in a pipeline
func UpdateParameterInPipeline(projectKey, pipelineName, paramName, newParamName, paramValue, paramType, paramDescription string) error {
	newParam := Parameter{
		Name:        newParamName,
		Value:       paramValue,
		Type:        paramType,
		Description: paramDescription,
	}

	data, errm := json.Marshal(newParam)
	if errm != nil {
		return errm
	}

	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter/%s", projectKey, pipelineName, paramName)
	data, _, errr := Request("PUT", path, data)
	if errr != nil {
		return errr
	}
	return DecodeError(data)
}

// RemoveParameterFromPipeline  remove a parameter from a pipeline
func RemoveParameterFromPipeline(projectKey, pipelineName, paramName string) error {
	path := fmt.Sprintf("/project/%s/pipeline/%s/parameter/%s", projectKey, pipelineName, paramName)
	data, _, errr := Request("DELETE", path, nil)
	if errr != nil {
		return errr
	}
	return DecodeError(data)
}

// GetPipelineBuildStatus retrieves current build information.
// With buildNumber at 0, fetch last build
func GetPipelineBuildStatus(proj, app, pip, env string, buildNumber int64) (PipelineBuild, error) {
	var pb PipelineBuild
	var uri string

	if buildNumber == 0 {
		uri = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/last?envName=%s",
			proj, app, pip, url.QueryEscape(env))
	} else {
		uri = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d?envName=%s",
			proj, app, pip, buildNumber, url.QueryEscape(env))
	}

	data, _, errr := Request("GET", uri, nil)
	if errr != nil {
		return pb, errr
	}
	if err := json.Unmarshal(data, &pb); err != nil {
		return pb, err
	}

	return pb, nil
}

// GetBuildingPipelines retrieves all building pipelines
func GetBuildingPipelines() ([]PipelineBuild, error) {
	data, _, err := Request("GET", "/mon/building", nil)
	if err != nil {
		return nil, err
	}
	var pbs []PipelineBuild
	if err := json.Unmarshal(data, &pbs); err != nil {
		return nil, err
	}

	return pbs, nil
}

// GetBuildingPipelineByHash retrieves pipeline building a specific commit hash
func GetBuildingPipelineByHash(hash string) ([]PipelineBuild, error) {
	var pbs []PipelineBuild

	data, _, errr := Request("GET", "/mon/building/"+hash, nil)
	if errr != nil {
		return nil, errr
	}
	if err := json.Unmarshal(data, &pbs); err != nil {
		return nil, err
	}

	return pbs, nil
}

// AddSpawnInfosPipelineBuildJob books a job for a Hatchery
func AddSpawnInfosPipelineBuildJob(pipelineBuildJobID int64, infos []SpawnInfo) error {
	data, errm := json.Marshal(infos)
	if errm != nil {
		return errm
	}

	path := fmt.Sprintf("/queue/%d/spawn/infos", pipelineBuildJobID)
	_, _, err := Request("POST", path, data)
	return err
}

// Translate translates messages in pipelineBuild
func (p *PipelineBuild) Translate(lang string) {
	for ks := range p.Stages {
		for kj := range p.Stages[ks].PipelineBuildJobs {
			p.Stages[ks].PipelineBuildJobs[kj].Translate(lang)
		}
	}
}
