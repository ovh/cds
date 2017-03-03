package exportentities

import (
	"fmt"
	"strconv"

	"sort"

	"strings"

	"github.com/ovh/cds/sdk"
)

// Pipeline represents exported sdk.Pipeline
type Pipeline struct {
	Name        string                    `json:"name" yaml:"name"`
	Type        string                    `json:"type" yaml:"type"`
	Permissions map[string]int            `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Parameters  map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Stages      map[string]Stage          `json:"stages,omitempty" yaml:"stages,omitempty"`
	Jobs        map[string]Job            `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Steps       []Step                    `json:"steps,omitempty" yaml:"steps,omitempty" hcl:"step,omitempty"`
}

// Stage represents exported sdk.Stage
type Stage struct {
	Enabled    *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Jobs       map[string]Job    `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Conditions map[string]string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Job represents exported sdk.Job
type Job struct {
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled      *bool         `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Steps        []Step        `json:"steps,omitempty" yaml:"steps,omitempty" hcl:"step,omitempty"`
	Requirements []Requirement `json:"requirements,omitempty" yaml:"requirements,omitempty" hcl:"requirement,omitempty"`
}

// Step represents exported step used in a job
type Step struct {
	Enabled          *bool                        `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Final            *bool                        `json:"final,omitempty" yaml:"final,omitempty"`
	ArtifactUpload   map[string]string            `json:"artifactUpload,omitempty" yaml:"artifactUpload,omitempty"`
	ArtifactDownload map[string]string            `json:"artifactDownload,omitempty" yaml:"artifactDownload,omitempty"`
	Script           string                       `json:"script,omitempty" yaml:"script,omitempty"`
	JUnitReport      string                       `json:"jUnitReport,omitempty" yaml:"jUnitReport,omitempty"`
	Plugin           map[string]map[string]string `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Action           map[string]map[string]string `json:"action,omitempty" yaml:"action,omitempty"`
}

// Requirement represents an exported sdk.Requirement
type Requirement struct {
	Binary   string             `json:"binary,omitempty" yaml:"binary,omitempty"`
	Network  string             `json:"network,omitempty" yaml:"network,omitempty"`
	Model    string             `json:"model,omitempty" yaml:"model,omitempty"`
	Hostname string             `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Plugin   string             `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Service  ServiceRequirement `json:"service,omitempty" yaml:"service,omitempty"`
	Memory   string             `json:"memory,omitempty" yaml:"memory,omitempty"`
}

// ServiceRequirement represents an exported sdk.Requirement of type ServiceRequirement
type ServiceRequirement struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

//NewPipeline creates an exportable pipeline from a sdk.Pipeline
func NewPipeline(pip *sdk.Pipeline) (p *Pipeline) {
	p = &Pipeline{}
	p.Name = pip.Name
	p.Type = string(pip.Type)
	p.Permissions = make(map[string]int, len(pip.GroupPermission))
	for _, perm := range pip.GroupPermission {
		p.Permissions[perm.Group.Name] = perm.Permission
	}
	p.Parameters = make(map[string]ParameterValue, len(pip.Parameter))
	for _, v := range pip.Parameter {
		p.Parameters[v.Name] = ParameterValue{
			Type:         string(v.Type),
			DefaultValue: v.Value,
		}
	}

	switch len(pip.Stages) {
	case 0:
		return
	case 1:
		if len(pip.Stages[0].Prerequisites) == 0 {
			switch len(pip.Stages[0].Jobs) {
			case 0:
				return
			case 1:
				p.Steps = newSteps(pip.Stages[0].Jobs[0].Action)
				return
			default:
				p.Jobs = newJobs(pip.Stages[0].Jobs)
			}
			return
		}
		p.Stages = newStages(pip.Stages)
	default:
		p.Stages = newStages(pip.Stages)
	}

	return
}

func newStages(stages []sdk.Stage) map[string]Stage {
	res := map[string]Stage{}
	var order int
	for i := range stages {
		s := &stages[i]
		if len(s.Jobs) == 0 {
			continue
		}
		order++
		st := Stage{}
		if !s.Enabled {
			st.Enabled = &s.Enabled
		}
		if len(s.Prerequisites) > 0 {
			st.Conditions = make(map[string]string)
		}
		for _, r := range s.Prerequisites {
			st.Conditions[r.Parameter] = r.ExpectedValue
		}
		st.Jobs = newJobs(s.Jobs)
		res[fmt.Sprintf("%d|%s", order, s.Name)] = st
	}
	return res
}

func newJobs(jobs []sdk.Job) map[string]Job {
	res := map[string]Job{}
	for i := range jobs {
		j := &jobs[i]
		if len(j.Action.Actions) == 0 {
			continue
		}
		jo := Job{}
		if !j.Enabled {
			jo.Enabled = &j.Enabled
		}
		jo.Steps = newSteps(j.Action)
		jo.Description = j.Action.Description
		for _, r := range j.Action.Requirements {
			switch r.Type {
			case sdk.BinaryRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Binary: r.Value})
			case sdk.NetworkAccessRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Network: r.Value})
			case sdk.ModelRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Model: r.Value})
			case sdk.HostnameRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Hostname: r.Value})
			case sdk.PluginRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Plugin: r.Value})
			case sdk.ServiceRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Service: ServiceRequirement{Name: r.Name, Value: r.Value}})
			case sdk.MemoryRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Memory: r.Value})
			}
		}

		res[j.Action.Name] = jo

	}
	return res
}

func newSteps(a sdk.Action) []Step {
	res := []Step{}
	for i := range a.Actions {
		a := &a.Actions[i]
		s := Step{}
		if !a.Enabled {
			s.Enabled = &a.Enabled
		}
		if a.Final {
			s.Final = &a.Final
		}

		switch a.Type {
		case sdk.BuiltinAction:
			switch a.Name {
			case sdk.ScriptAction:
				script := sdk.ParameterFind(a.Parameters, "script")
				if script != nil {
					s.Script = script.Value
				}
			case sdk.ArtifactDownload:
				s.ArtifactDownload = map[string]string{}
				path := sdk.ParameterFind(a.Parameters, "path")
				if path != nil {
					s.ArtifactDownload["path"] = path.Value
				}
				tag := sdk.ParameterFind(a.Parameters, "tag")
				if tag != nil {
					s.ArtifactDownload["tag"] = tag.Value
				}
			case sdk.ArtifactUpload:
				s.ArtifactUpload = map[string]string{}
				path := sdk.ParameterFind(a.Parameters, "path")
				if path != nil {
					s.ArtifactUpload["path"] = path.Value
				}
				tag := sdk.ParameterFind(a.Parameters, "tag")
				if tag != nil {
					s.ArtifactUpload["tag"] = tag.Value
				}
			case sdk.JUnitAction:
				path := sdk.ParameterFind(a.Parameters, "path")
				if path != nil {
					s.JUnitReport = path.Value
				}
			}
		case sdk.PluginAction:
			s.Plugin = map[string]map[string]string{}
			s.Plugin[a.Name] = map[string]string{}
			for _, p := range a.Parameters {
				if p.Value != "" {
					s.Plugin[a.Name][p.Name] = p.Value
				}
			}
		default:
			s.Action = map[string]map[string]string{}
			s.Action[a.Name] = map[string]string{}
			for _, p := range a.Parameters {
				if p.Value != "" {
					s.Action[a.Name][p.Name] = p.Value
				}
			}
		}
		res = append(res, s)
	}

	return res
}

//Pipeline returns a sdk.Pipeline entity
func (p *Pipeline) Pipeline() (*sdk.Pipeline, error) {
	pip := new(sdk.Pipeline)

	pip.Name = p.Name
	pip.Type = p.Type

	//Compute permissions
	for g, p := range p.Permissions {
		perm := sdk.GroupPermission{
			Group:      sdk.Group{Name: g},
			Permission: p,
		}
		pip.GroupPermission = append(pip.GroupPermission, perm)
	}

	//Compute parameters
	for p, v := range p.Parameters {
		param := sdk.Parameter{
			Name:  p,
			Type:  sdk.ParameterTypeFromString(v.Type),
			Value: v.DefaultValue,
		}
		pip.Parameter = append(pip.Parameter, param)
	}

	if p.Steps != nil {
		//There one stage, with one job
		actions, err := computeSteps(p.Steps)
		if err != nil {
			return nil, err
		}
		pip.Stages = []sdk.Stage{
			sdk.Stage{
				Name:       p.Name,
				BuildOrder: 1,
				Enabled:    true,
				Jobs: []sdk.Job{
					sdk.Job{
						Enabled: true,
						Action: sdk.Action{
							Name:    p.Name,
							Actions: actions,
						},
					},
				},
			},
		}

	} else if p.Jobs != nil {
		//There is one stage with several jobs
		stage := sdk.Stage{
			Name:       p.Name,
			BuildOrder: 1,
			Enabled:    true,
		}
		for s, j := range p.Jobs {
			job, err := computeJob(s, j)
			if err != nil {
				return nil, err
			}
			stage.Jobs = append(stage.Jobs, *job)
		}
		pip.Stages = []sdk.Stage{stage}
	} else {
		//There is more than one stage
		stageKeys := []string{}
		for k := range p.Stages {
			stageKeys = append(stageKeys, k)
		}
		sort.Strings(stageKeys)

		//Compute stages
		for i, stageName := range stageKeys {
			buildOrder := i
			name := stageName
			//Try to find buildOrder and name
			if strings.Contains(stageName, "|") {
				t := strings.SplitN(stageName, "|", 2)
				var err error
				buildOrder, err = strconv.Atoi(t[0])
				if err != nil {
					return nil, fmt.Errorf("malformatted stage name : %s", stageName)
				}
				name = t[1]
			}

			s := sdk.Stage{
				BuildOrder: buildOrder,
				Name:       name,
			}

			if p.Stages[stageName].Enabled != nil {
				s.Enabled = *p.Stages[stageName].Enabled
			} else {
				s.Enabled = true
			}

			//Compute stage Prerequisites
			for n, c := range p.Stages[stageName].Conditions {
				s.Prerequisites = append(s.Prerequisites, sdk.Prerequisite{
					Parameter:     n,
					ExpectedValue: c,
				})
			}

			//Compute jobs
			for n, j := range p.Stages[stageName].Jobs {
				job, err := computeJob(n, j)
				if err != nil {
					return nil, err
				}
				s.Jobs = append(s.Jobs, *job)
			}

			pip.Stages = append(pip.Stages, s)
		}
	}

	return pip, nil
}

func computeSteps(steps []Step) ([]sdk.Action, error) {
	res := []sdk.Action{}
	for _, s := range steps {
		a, err := computeStep(s)
		if err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, nil
}

func computeStep(s Step) (a sdk.Action, e error) {
	//Compute artifact upload
	if s.Script != "" {
		a = sdk.NewStepScript(s.Script)
	} else if s.JUnitReport != "" {
		a = sdk.NewStepJUnitReport(s.JUnitReport)
	} else if s.ArtifactUpload != nil {
		a = sdk.NewStepArtifactUpload(s.ArtifactUpload)
	} else if s.ArtifactDownload != nil {
		a = sdk.NewStepArtifactDownload(s.ArtifactDownload)
	} else if s.Plugin != nil {
		act, err := sdk.NewStepPlugin(s.Plugin)
		if err != nil {
			e = err
			return
		}
		a = *act
	} else if s.Action != nil {
		act, err := sdk.NewStepDefault(s.Action)
		if err != nil {
			e = err
			return
		}
		a = *act
	} else {
		e = fmt.Errorf("Malformatted step")
		return
	}
	//Compute enable flag
	if s.Enabled != nil {
		a.Enabled = *s.Enabled
	} else {
		a.Enabled = true
	}
	//Compute final flag
	if s.Final != nil {
		a.Final = *s.Final
	} else {
		a.Final = false
	}

	return
}

func computeJob(name string, j Job) (*sdk.Job, error) {
	job := sdk.Job{
		Action: sdk.Action{
			Name:        name,
			Description: j.Description,
		},
	}
	if j.Enabled != nil {
		job.Enabled = *j.Enabled
	} else {
		job.Enabled = true
	}
	for _, r := range j.Requirements {
		var name, tpe, val string
		if r.Binary != "" {
			name = r.Binary
			val = r.Binary
			tpe = sdk.BinaryRequirement
		} else if r.Hostname != "" {
			name = "hostname"
			val = r.Hostname
			tpe = sdk.HostnameRequirement
		} else if r.Memory != "" {
			name = "memory"
			val = r.Memory
			tpe = sdk.MemoryRequirement
		} else if r.Model != "" {
			name = "model"
			val = r.Model
			tpe = sdk.ModelRequirement
		} else if r.Network != "" {
			name = "network"
			val = r.Network
			tpe = sdk.NetworkAccessRequirement
		} else if r.Plugin != "" {
			name = r.Plugin
			val = r.Plugin
			tpe = sdk.PluginRequirement
		} else if r.Service.Name != "" {
			name = r.Service.Name
			val = r.Service.Value
			tpe = sdk.ServiceRequirement
		}
		job.Action.Requirement(name, tpe, val)
	}

	//Compute steps for the jobs
	children, err := computeSteps(j.Steps)
	if err != nil {
		return nil, err
	}
	job.Action.Actions = children

	return &job, nil
}
