package workflowv3

import "strings"

type Actions map[string]Action

func (a Actions) ExistAction(actionName string) bool {
	_, ok := a[actionName]
	return ok
}

func (a Actions) ToGraph() Graph {
	var res Graph
	for aName, action := range a {
		res = append(res, Node{
			Name:      aName,
			DependsOn: action.GetLocalDependencies(),
		})
	}
	return res
}

type Action struct {
	Parameters   map[string]ActionParameter `json:"description,omitempty" yaml:"description,omitempty"`
	Requirements []Requirement              `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Steps        []Step                     `json:"steps,omitempty" yaml:"steps,omitempty"`
}

func (a Action) Validate(w Workflow) (ExternalDependencies, error) {
	var extDep ExternalDependencies

	for _, s := range a.Steps {
		dep, err := s.Validate(w)
		if err != nil {
			return extDep, err
		}
		extDep.Add(dep)
	}

	return extDep, nil
}

func (a Action) GetLocalDependencies() []string {
	var res []string
	for _, s := range a.Steps {
		for sName := range s.StepCustom {
			if !strings.HasPrefix(sName, "@") {
				res = append(res, sName)
			}
		}
	}
	return res
}

type ActionParameter struct {
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Requirement struct {
	Binary            string             `json:"binary,omitempty" yaml:"binary,omitempty"`
	Model             string             `json:"model,omitempty" yaml:"model,omitempty"`
	Hostname          string             `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Plugin            string             `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Service           ServiceRequirement `json:"service,omitempty" yaml:"service,omitempty"`
	Memory            string             `json:"memory,omitempty" yaml:"memory,omitempty"`
	OSArchRequirement string             `json:"os-architecture,omitempty" yaml:"os-architecture,omitempty"`
	RegionRequirement string             `json:"region,omitempty" yaml:"region,omitempty"`
}

type ServiceRequirement struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}
