package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Action is the base element of CDS pipeline
type Action struct {
	ID               int64         `json:"id" yaml:"-"`
	Name             string        `json:"name"`
	Type             string        `json:"type" yaml:"-"` // Source - Build - Packaging - Deployment
	Description      string        `json:"description" yaml:"desc,omitempty"`
	Requirements     []Requirement `json:"requirements"`
	Parameters       []Parameter   `json:"parameters"`
	Actions          []Action      `json:"actions" yaml:"actions,omitempty"`
	Enabled          bool          `json:"enabled" yaml:"-"`
	PipelineStageID  int64         `json:"pipeline_stage_id" yaml:"-"`
	PipelineActionID int64         `json:"pipeline_action_id" yaml:"-"`
	Final            bool          `json:"final" yaml:"-"`
	LastModified     int64         `json:"last_modified"`
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
	ScriptAction = "Script"
	NotifAction  = "Notif"
	JUnitAction  = "JUnit"
)

const (
	//BinaryRequirement refers to the need to a specific binary on host running the action
	BinaryRequirement = "binary"
	// NetworkAccessRequirement refers to the need of an opened network acces to given endpoint.
	NetworkAccessRequirement = "network"
	// ModelRequirement refers to the need fo a specific model
	ModelRequirement = "model"
	// HostnameRequirement checks the hostname of the worker
	HostnameRequirement = "hostname"
	//PluginRequirement installs & checks plugins of the worker
	PluginRequirement = "plugin"
	//ServiceRequirement links a service to a worker
	ServiceRequirement = "service"
	//MemoryRequirement set memory limit on a container
	MemoryRequirement = "memory"
)

var (
	// AvailableRequirementsType List of all requirements
	AvailableRequirementsType = []string{
		BinaryRequirement,
		NetworkAccessRequirement,
		ModelRequirement,
		HostnameRequirement,
		PluginRequirement,
		ServiceRequirement,
		MemoryRequirement,
	}
)

// Requirement can be :
// - a binary "which /usr/bin/docker"
// - a network access "telnet google.com 443"
type Requirement struct {
	Name  string `json:"name"`
	Type  string `json:"type" yaml:"-"`
	Value string `json:"value" yaml:"-"`
}

// NewAction instanciate a new Action
func NewAction(name string) *Action {
	a := &Action{
		Name:    name,
		Enabled: true,
	}
	return a
}

// NewJoinedAction is a helper to build an action object acting as an joined action
func NewJoinedAction(actionName string, parameters []Parameter) (*Action, error) {

	// Now retrieves the action to add into joined action
	a, err := GetAction(actionName)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve action %s (%s)", actionName, err)
	}
	// Set parameters value
	for i := range a.Parameters {
		var paramSet bool
		for _, up := range parameters {
			if a.Parameters[i].Name == up.Name {
				a.Parameters[i].Value = up.Value
				paramSet = true
				break
			}
		}
		if !paramSet {
			return nil, fmt.Errorf("parameter '%s' of action '%s' not provided", a.Parameters[i].Name, actionName)
		}
	}

	// Create joined action
	return NewAction(actionName).Add(a), nil
}

// Requirement add given requirement to Action
func (a *Action) Requirement(name string, t string, value string) *Action {
	r := Requirement{
		Name:  name,
		Type:  t,
		Value: value,
	}

	a.Requirements = append(a.Requirements, r)
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

// JSON return the marshalled string of Action object
func (a *Action) JSON() string {

	data, err := json.Marshal(a)
	if err != nil {
		fmt.Printf("Action.JSON: cannot marshal: %s\n", err)
		return ""
	}

	return string(data)
}

// FromJSON unmarshal given json data into Action object
func (a *Action) FromJSON(data []byte) (*Action, error) {
	return a, json.Unmarshal(data, &a)
}

// AddAction creates a new action available only to creator by default
// params are stringParameter only (for now), with no description
func AddAction(name string, params []Parameter, requirements []Requirement) error {

	a := NewAction(name)
	a.Parameters = params
	a.Requirements = requirements
	a.Enabled = true

	data, err := json.MarshalIndent(a, " ", " ")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/action/%s", name)
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

// AddActionStep add a new step of type Action to given action
func AddActionStep(actionName string, child Action) error {

	a, err := GetAction(actionName)
	if err != nil {
		return err
	}

	a.Actions = append(a.Actions, child)

	return UpdateAction(a)
}

// UpdateAction update given action
func UpdateAction(a Action) error {
	uri := fmt.Sprintf("/action/%s", a.Name)

	data, err := json.Marshal(a)
	if err != nil {
		return err
	}

	_, code, err := Request("PUT", uri, data)
	if err != nil {
		return err
	}

	if code > 300 {
		return fmt.Errorf("HTTP %d", code)
	}

	return nil
}

// ListActions returns all available actions to caller
func ListActions() ([]Action, error) {

	data, code, err := Request("GET", "/action", nil)
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	var acts []Action
	err = json.Unmarshal(data, &acts)
	if err != nil {
		return nil, err
	}

	return acts, nil
}

// GetAction retrieve action definition
func GetAction(name string) (Action, error) {
	var a Action

	path := fmt.Sprintf("/action/%s", name)
	data, _, err := Request("GET", path, nil)
	if err != nil {
		return a, err
	}

	err = json.Unmarshal(data, &a)
	if err != nil {
		return a, err
	}

	return a, nil
}

// DeleteAction remove given action from CDS
// if action is not used in any pipeline
func DeleteAction(name string) error {
	path := fmt.Sprintf("/action/%s", name)

	_, _, err := Request("DELETE", path, nil)
	if err != nil {
		return err
	}

	return nil
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

// AddJoinedAction creates a joined action in given pipeline
func AddJoinedAction(projectKey, pipelineName string, stageID int64, a *Action) error {
	uri := fmt.Sprintf("/project/%s/pipeline/%s/stage/%d/joined", projectKey, pipelineName, stageID)

	data, err := json.Marshal(a)
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
func UpdateJoinedAction(projectKey, pipelineName string, stage int, a Action) error {
	uri := fmt.Sprintf("/project/%s/pipeline/%s/stage/%d/joined/%d", projectKey, pipelineName, stage, a.ID)

	data, err := json.Marshal(a)
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

//ImportAction imports an action on CDS
func ImportAction(action *Action) (*Action, error) {
	path := "/action/import"

	btes, errMarshall := json.Marshal(action)
	if errMarshall != nil {
		return nil, errMarshall
	}
	data, code, errRequest := Request("POST", path, btes)
	if errRequest != nil {
		return nil, errRequest
	}

	if code >= 300 {
		return nil, fmt.Errorf("Error [%d]: %s", code, data)
	}

	if err := DecodeError(data); err != nil {
		return nil, err
	}

	var act Action
	if err := json.Unmarshal(data, &act); err != nil {
		return nil, err
	}
	return &act, nil
}

//GetRequirements returns the list of all used requirements
func GetRequirements() ([]Requirement, error) {
	path := "/action/requirement"

	data, code, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var req []Requirement
	err = json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}
	return req, nil
}
