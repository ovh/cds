package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// Action is the base element of CDS pipeline
type Action struct {
	ID             int64         `json:"id" yaml:"-"`
	Name           string        `json:"name" cli:"name"`
	Type           string        `json:"type" yaml:"-" cli:"type"`
	Description    string        `json:"description" yaml:"desc,omitempty"`
	Requirements   []Requirement `json:"requirements"`
	Parameters     []Parameter   `json:"parameters"`
	Actions        []Action      `json:"actions" yaml:"actions,omitempty"`
	Enabled        bool          `json:"enabled" yaml:"-"`
	Deprecated     bool          `json:"deprecated" yaml:"-"`
	Optional       bool          `json:"optional" yaml:"-"`
	AlwaysExecuted bool          `json:"always_executed" yaml:"-"`
	LastModified   int64         `json:"last_modified" cli:"modified"`
}

// ActionSummary is the light representation of an action for CDS event
type ActionSummary struct {
	Name string `json:"name"`
}

func (a Action) ToSummary() ActionSummary {
	return ActionSummary{
		Name: a.Name,
	}
}

// ActionAudit Audit on action
type ActionAudit struct {
	ActionID   int64     `json:"action_id"`
	User       User      `json:"user"`
	Change     string    `json:"change"`
	Versionned time.Time `json:"versionned"`
	Action     Action    `json:"action"`
}

// ActionPlugin  is the Action Plugin representation from Engine side
type ActionPlugin struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Filename    string `json:"filename"`
	Path        string `json:"path"`

	Size       int64  `json:"size,omitempty"`
	Perm       uint32 `json:"perm,omitempty"`
	MD5sum     string `json:"md5sum,omitempty"`
	SHA512sum  string `json:"sha512sum,omitempty"`
	ObjectPath string `json:"object_path,omitempty"`
}

//GetName returns the name the action plugin
func (a *ActionPlugin) GetName() string {
	return a.Name
}

//GetPath returns the storage path of the action plugin
func (a *ActionPlugin) GetPath() string {
	return fmt.Sprintf("plugins")
}

// Action type
const (
	DefaultAction = "Default"
	BuiltinAction = "Builtin"
	PluginAction  = "Plugin"
	JoinedAction  = "Joined"
)

// Builtin Action
const (
	ScriptAction              = "Script"
	JUnitAction               = "JUnit"
	GitCloneAction            = "GitClone"
	GitTagAction              = "GitTag"
	ReleaseAction             = "Release"
	CheckoutApplicationAction = "CheckoutApplication"
	DeployApplicationAction   = "DeployApplication"
)

// NewAction instanciate a new Action
func NewAction(name string) *Action {
	a := &Action{
		Name:    name,
		Enabled: true,
	}
	return a
}

// Parameter add given parameter to Action
func (a *Action) Parameter(p Parameter) *Action {
	a.Parameters = append(a.Parameters, p)
	return a
}

// Add takes an action that will be executed when current action is executed
func (a *Action) Add(child Action) *Action {
	a.Actions = append(a.Actions, child)
	return a
}

// GetAction retrieve action definition
func GetAction(name string) (Action, error) {
	var a Action

	path := fmt.Sprintf("/action/%s", name)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return a, err
	}

	if err := json.Unmarshal(data, &a); err != nil {
		return a, err
	}

	return a, nil
}

// NewScriptAction setup a new Action object with all attribute ok for script action
func NewScriptAction(content string) Action {
	var a Action

	a.Name = ScriptAction
	a.Type = BuiltinAction
	a.Enabled = true
	a.Parameters = append(a.Parameters, Parameter{Name: "script", Value: content})
	return a
}

// AddJob creates a joined action in given pipeline
func AddJob(projectKey, pipelineName string, j *Job) error {
	uri := fmt.Sprintf("/project/%s/pipeline/%s/stage/%d/job", projectKey, pipelineName, j.PipelineStageID)

	data, err := json.Marshal(j)
	if err != nil {
		return err
	}

	_, code, err := Request("POST", uri, data)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// UpdateJoinedAction update given joined action in given pipeline stage
func UpdateJoinedAction(projectKey, pipelineName string, stage int64, j *Job) error {
	uri := fmt.Sprintf("/project/%s/pipeline/%s/stage/%d/job/%d", projectKey, pipelineName, stage, j.PipelineActionID)

	data, err := json.Marshal(j)
	if err != nil {
		return err
	}

	_, code, err := Request("PUT", uri, data)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}
