package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
)

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
	InstallKeyAction          = "InstallKey"

	DefaultGitCloneParameterTagValue = "{{.git.tag}}"
)

// NewAction instantiate a new Action
func NewAction(name string) *Action {
	return &Action{
		Name:    name,
		Enabled: true,
	}
}

// Action is the base element of CDS pipeline
type Action struct {
	ID          int64  `json:"id" yaml:"-" db:"id"`
	GroupID     *int64 `json:"group_id,omitempty" yaml:"-" db:"group_id"`
	Name        string `json:"name" db:"name"`
	Type        string `json:"type" yaml:"-" db:"type"`
	Description string `json:"description" yaml:"desc,omitempty" db:"description"`
	Enabled     bool   `json:"enabled" yaml:"-" db:"enabled"`
	Deprecated  bool   `json:"deprecated" yaml:"-" db:"deprecated"`
	// aggregates from action_edge
	StepName       string `json:"step_name,omitempty" yaml:"step_name,omitempty" db:"-"`
	Optional       bool   `json:"optional" yaml:"-" db:"-"`
	AlwaysExecuted bool   `json:"always_executed" yaml:"-" db:"-"`
	// aggregates
	Requirements RequirementList `json:"requirements" db:"-"`
	Parameters   []Parameter     `json:"parameters" db:"-"`
	Actions      []Action        `json:"actions,omitempty" yaml:"actions,omitempty" db:"-"`
	Group        *Group          `json:"group,omitempty" db:"-"`
	FirstAudit   *AuditAction    `json:"first_audit,omitempty" db:"-"`
	LastAudit    *AuditAction    `json:"last_audit,omitempty" db:"-"`
	Editable     bool            `json:"editable,omitempty" db:"-"`
}

// UsageAction represent a action using an action.
type UsageAction struct {
	GroupID          int64  `json:"group_id"`
	GroupName        string `json:"group_name"`
	ParentActionID   int64  `json:"parent_action_id"`
	ParentActionName string `json:"parent_action_name"`
	ActionID         int64  `json:"action_id"`
	ActionName       string `json:"action_name"`
	Warning          bool   `json:"warning"`
}

// ActionUsages for action.
type ActionUsages struct {
	Pipelines []UsagePipeline `json:"pipelines"`
	Actions   []UsageAction   `json:"actions"`
}

// UsagePipeline represent a pipeline using an action.
type UsagePipeline struct {
	ProjectID    int64  `json:"project_id"`
	ProjectKey   string `json:"project_key"`
	ProjectName  string `json:"project_name"`
	PipelineID   int64  `json:"pipeline_id"`
	PipelineName string `json:"pipeline_name"`
	StageID      int64  `json:"stage_id"`
	StageName    string `json:"stage_name"`
	JobID        int64  `json:"job_id"`
	JobName      string `json:"job_name"`
	ActionID     int64  `json:"action_id"`
	ActionName   string `json:"action_name"`
	Warning      bool   `json:"warning"`
}

// Value returns driver.Value from action.
func (a Action) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, WrapError(err, "cannot marshal Action")
}

// Scan action.
func (a *Action) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, a), "cannot unmarshal Action")
}

// IsValidDefault returns default action validity.
func (a Action) IsValidDefault() error {
	if a.GroupID == nil || *a.GroupID == 0 {
		return NewErrorFrom(ErrWrongRequest, "invalid group id for action")
	}

	return a.IsValid()
}

// IsValid returns error if the action is not valid.
func (a Action) IsValid() error {
	if a.Name == "" {
		return NewErrorFrom(ErrWrongRequest, "invalid name for action")
	}

	for i := range a.Parameters {
		if err := a.Parameters[i].IsValid(); err != nil {
			return err
		}
	}

	if err := a.Requirements.IsValid(); err != nil {
		return err
	}

	for i := range a.Actions {
		if a.Actions[i].ID == 0 {
			return NewErrorFrom(ErrWrongRequest, "invalid action id for child")
		}
		for j := range a.Actions[i].Parameters {
			if err := a.Actions[i].Parameters[j].IsValid(); err != nil {
				return err
			}
		}
	}

	return nil
}

// FlattenRequirements returns all requirements for an action and its children.
func (a *Action) FlattenRequirements() RequirementList {
	rs := a.Requirements

	// copy requirements from childs
	for i := range a.Actions {
		if !a.Actions[i].Enabled {
			continue
		}

		rsChild := a.Actions[i].Requirements

		// now filter child requirements, not already in parent
		// do not add a model or hostname requirement if parent already contains one
		filtered := make([]Requirement, 0, len(rsChild))
		for j := range rsChild {
			var found bool
			for k := range rs {
				if rs[k].Type == rsChild[j].Type &&
					(rs[k].Type == ModelRequirement ||
						rs[k].Type == HostnameRequirement ||
						rs[k].Value == rsChild[j].Value) {
					found = true
					break
				}
			}
			if !found {
				filtered = append(filtered, rsChild[j])
			}
		}

		rs = append(rs, filtered...)
	}

	return rs
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

// ToUniqueChildrenIDs returns distinct children ids for given action.
func (a Action) ToUniqueChildrenIDs() []int64 {
	mChildrenIDs := make(map[int64]struct{}, len(a.Actions))
	for i := range a.Actions {
		mChildrenIDs[a.Actions[i].ID] = struct{}{}
	}
	childrenIDs := make([]int64, len(mChildrenIDs))
	i := 0
	for id := range mChildrenIDs {
		childrenIDs[i] = id
		i++
	}
	return childrenIDs
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

// ActionsToIDs returns ids for given actions list.
func ActionsToIDs(as []*Action) []int64 {
	ids := make([]int64, len(as))
	for i := range as {
		ids[i] = as[i].ID
	}
	return ids
}

// ActionsToGroupIDs returns group ids for given actions list.
func ActionsToGroupIDs(as []*Action) []int64 {
	ids := make([]int64, 0, len(as))
	for i := range as {
		if as[i].GroupID != nil {
			ids = append(ids, *as[i].GroupID)
		}
	}
	return ids
}

// ActionsFilterNotTypes returns a list of actions filtered by types.
func ActionsFilterNotTypes(as []*Action, ts ...string) []*Action {
	f := make([]*Action, 0, len(as))
	for i := range as {
		for j := range ts {
			if as[i].Type != ts[j] {
				f = append(f, as[i])
			}
		}
	}
	return f
}
