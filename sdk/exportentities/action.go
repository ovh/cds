package exportentities

import (
	"github.com/ovh/cds/sdk"
)

// Action represents exported sdk.Action
type Action struct {
	Version      string                    `json:"version,omitempty" yaml:"version,omitempty"`
	Name         string                    `json:"name,omitempty" yaml:"name,omitempty"`
	Group        string                    `json:"group,omitempty" yaml:"group,omitempty"`
	Description  string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled      *bool                     `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Parameters   map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Requirements []Requirement             `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Steps        []Step                    `json:"steps,omitempty" yaml:"steps,omitempty"`
}

// ActionVersion is a version
type ActionVersion string

// There are the supported versions
const (
	ActionVersion1 = "v1.0"
)

// NewAction returns a ready to export action
func NewAction(a sdk.Action) Action {
	var ea Action

	ea.Name = a.Name

	if a.Group != nil {
		ea.Group = a.Group.Name
	}

	ea.Version = ActionVersion1
	ea.Description = a.Description
	ea.Parameters = make(map[string]ParameterValue, len(a.Parameters))
	for k, v := range a.Parameters {
		param := ParameterValue{
			Type:         string(v.Type),
			DefaultValue: v.Value,
			Description:  v.Description,
		}
		// no need to export it if "Advanced" is false
		if v.Advanced {
			param.Advanced = &a.Parameters[k].Advanced
		}
		ea.Parameters[v.Name] = param
	}
	ea.Steps = newSteps(a)
	ea.Requirements = NewRequirements(a.Requirements)
	// enabled is the default value
	// set enable attribute only if it's disabled
	// no need to export it if action is enabled
	if !a.Enabled {
		ea.Enabled = &a.Enabled
	}

	return ea
}

func newSteps(a sdk.Action) []Step {
	res := make([]Step, len(a.Actions))
	for i := range a.Actions {
		res[i] = NewStep(a.Actions[i])
	}

	return res
}

// GetAction returns an sdk.Action
func (ea *Action) GetAction() (sdk.Action, error) {
	a := sdk.Action{
		Name:        ea.Name,
		Group:       &sdk.Group{Name: ea.Group},
		Type:        sdk.DefaultAction,
		Enabled:     true,
		Description: ea.Description,
		Parameters:  make([]sdk.Parameter, len(ea.Parameters)),
	}
	if ea.Group == "" {
		a.Group.Name = sdk.SharedInfraGroupName
	}

	var i int
	for p, v := range ea.Parameters {
		a.Parameters[i] = sdk.Parameter{
			Name:        p,
			Type:        v.Type,
			Value:       v.DefaultValue,
			Description: v.Description,
			Advanced:    v.Advanced != nil && *v.Advanced,
		}
		if v.Type == "" {
			a.Parameters[i].Type = sdk.StringParameter
		}
		i++
	}

	a.Requirements = computeJobRequirements(ea.Requirements)

	children, err := computeSteps(ea.Steps)
	if err != nil {
		return a, err
	}
	a.Actions = children

	return a, nil
}
