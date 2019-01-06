package sdk

import (
	"time"
)

// Action is the base element of CDS pipeline
type Action struct {
	ID             int64         `json:"id" yaml:"-"`
	Name           string        `json:"name" cli:"name,key"`
	StepName       string        `json:"step_name,omitempty" yaml:"step_name,omitempty" cli:"step_name"`
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
	Name     string `json:"name"`
	StepName string `json:"step_name"`
}

// ToSummary returns an ActionSummary from an Action
func (a Action) ToSummary() ActionSummary {
	return ActionSummary{
		Name:     a.Name,
		StepName: a.StepName,
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
	CoverageAction            = "Coverage"
	GitCloneAction            = "GitClone"
	GitTagAction              = "GitTag"
	ReleaseAction             = "Release"
	CheckoutApplicationAction = "CheckoutApplication"
	DeployApplicationAction   = "DeployApplication"

	DefaultGitCloneParameterTagValue = "{{.git.tag}}"
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

// NewScriptAction setup a new Action object with all attribute ok for script action
func NewScriptAction(content string) Action {
	var a Action

	a.Name = ScriptAction
	a.Type = BuiltinAction
	a.Enabled = true
	a.Parameters = append(a.Parameters, Parameter{Name: "script", Value: content})
	return a
}
