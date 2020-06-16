package exportentities

import (
	"sort"

	"github.com/ovh/cds/sdk"
)

// Pipeliner interface, depending on the version we will use different struct.
type Pipeliner interface {
	Pipeline() (*sdk.Pipeline, error)
}

// PipelineV1 represents exported sdk.Pipeline
type PipelineV1 struct {
	Version      string                    `json:"version,omitempty" yaml:"version,omitempty" jsonschema_description:"The version for the current pipeline file (v1.0)."`
	Name         string                    `json:"name,omitempty" yaml:"name,omitempty" jsonschema_description:"The name of the pipeline."`
	Description  string                    `json:"description,omitempty" yaml:"description,omitempty" jsonschema_description:"The description of the pipeline."`
	Parameters   map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty" jsonschema_description:"The list of parameters of the pipeline."`
	Stages       []string                  `json:"stages,omitempty" yaml:"stages,omitempty" jsonschema_description:"The list of stage's names for the pipeline."`
	StageOptions map[string]Stage          `json:"options,omitempty" yaml:"options,omitempty" jsonschema_description:"The options for stages of the pipeline."` //Here Stage.Jobs will NEVER be set
	Jobs         []Job                     `json:"jobs,omitempty" yaml:"jobs,omitempty" jsonschema_description:"The list of jobs for the pipeline."`
}

// PipelineVersion is a version
type PipelineVersion string

// There are the supported versions
const (
	PipelineVersion1 = "v1.0"
)

// Stage represents exported sdk.Stage
type Stage struct {
	Enabled    *bool                       `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Jobs       map[string]Job              `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Conditions *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Job represents exported sdk.Job
type Job struct {
	Name           string        `json:"job,omitempty" yaml:"job,omitempty" jsonschema_description:"The name of the job."`
	Stage          string        `json:"stage,omitempty" yaml:"stage,omitempty" jsonschema_description:"The name of the stage for the job."`
	Description    string        `json:"description,omitempty" yaml:"description,omitempty" jsonschema_description:"The description of the job."`
	Enabled        *bool         `json:"enabled,omitempty" yaml:"enabled,omitempty" jsonschema_description:"Job is enabled by default, you can set this option to disable a job."`
	Steps          []Step        `json:"steps,omitempty" yaml:"steps,omitempty" jsonschema_description:"The list of steps for the job."`
	Requirements   []Requirement `json:"requirements,omitempty" yaml:"requirements,omitempty" jsonschema_description:"The list of requirements for the jobs."`
	Optional       *bool         `json:"optional,omitempty" yaml:"optional,omitempty" jsonschema_description:"Set this option to ignore job's errors."`
	AlwaysExecuted *bool         `json:"always_executed,omitempty" yaml:"always_executed,omitempty" jsonschema_description:"Set this option to execute the job even if a previous step failed."`
}

// Requirement represents an exported sdk.Requirement
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

// ServiceRequirement represents an exported sdk.Requirement of type ServiceRequirement
type ServiceRequirement struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

//NewPipelineV1 creates an exportable pipeline from a sdk.Pipeline
func NewPipelineV1(pip sdk.Pipeline) (p PipelineV1) {
	p.Name = pip.Name
	p.Description = pip.Description
	p.Version = PipelineVersion1

	p.Parameters = make(map[string]ParameterValue, len(pip.Parameter))
	for _, v := range pip.Parameter {
		p.Parameters[v.Name] = ParameterValue{
			Type:         string(v.Type),
			DefaultValue: v.Value,
			Description:  v.Description,
		}
	}

	p.Stages, p.StageOptions = newStagesForPipelineV1(pip.Stages)

	//If there is one stages and no options
	if len(p.Stages) == 1 && len(p.StageOptions) == 0 {
		p.Stages = nil
	}

	for _, s := range pip.Stages {
		for _, j := range s.Jobs {
			jo := newJob(j)
			if len(pip.Stages) > 1 {
				jo.Stage = s.Name
			}
			jo.Name = j.Action.Name
			p.Jobs = append(p.Jobs, jo)
		}
	}

	return
}

func newStagesForPipelineV1(stages []sdk.Stage) ([]string, map[string]Stage) {
	res := make([]string, len(stages))
	opts := make(map[string]Stage, len(stages))
	for i := range stages {
		s := &stages[i]
		res[i] = s.Name

		var hasOptions bool

		st := Stage{}
		if !s.Enabled {
			st.Enabled = &s.Enabled
			hasOptions = true
		}
		if len(s.Conditions.PlainConditions) > 0 || s.Conditions.LuaScript != "" {
			st.Conditions = &s.Conditions
			hasOptions = true
		}

		if hasOptions == true {
			opts[s.Name] = st
		}
	}
	return res, opts
}

func newRequirements(req []sdk.Requirement) []Requirement {
	if req == nil {
		return nil
	}
	res := make([]Requirement, 0, len(req))
	for _, r := range req {
		switch r.Type {
		case sdk.BinaryRequirement:
			res = append(res, Requirement{Binary: r.Value})
		case sdk.ModelRequirement:
			res = append(res, Requirement{Model: r.Value})
		case sdk.HostnameRequirement:
			res = append(res, Requirement{Hostname: r.Value})
		case sdk.PluginRequirement:
			res = append(res, Requirement{Plugin: r.Value})
		case sdk.ServiceRequirement:
			res = append(res, Requirement{Service: ServiceRequirement{Name: r.Name, Value: r.Value}})
		case sdk.OSArchRequirement:
			res = append(res, Requirement{OSArchRequirement: r.Value})
		case sdk.RegionRequirement:
			res = append(res, Requirement{RegionRequirement: r.Value})
		case sdk.MemoryRequirement:
			res = append(res, Requirement{Memory: r.Value})
		}
	}
	return res
}

func newJob(j sdk.Job) Job {
	jo := Job{}
	if !j.Enabled {
		jo.Enabled = &j.Enabled
	}
	jo.Steps = newSteps(j.Action)
	jo.Description = j.Action.Description
	jo.Requirements = newRequirements(j.Action.Requirements)
	return jo
}

func newJobs(jobs []sdk.Job) map[string]Job {
	res := map[string]Job{}
	for i := range jobs {
		j := jobs[i]
		if len(j.Action.Actions) == 0 {
			continue
		}
		jo := newJob(j)
		res[j.Action.Name] = jo
	}
	return res
}

func computeSteps(steps []Step) ([]sdk.Action, error) {
	res := make([]sdk.Action, len(steps))
	for i, s := range steps {
		a, err := s.toAction()
		if err != nil {
			return nil, err
		}
		res[i] = *a
	}
	return res, nil
}

func computeJobRequirements(req []Requirement) []sdk.Requirement {
	res := make([]sdk.Requirement, len(req))
	for i, r := range req {
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
		} else if r.OSArchRequirement != "" {
			name = r.OSArchRequirement
			val = r.OSArchRequirement
			tpe = sdk.OSArchRequirement
		} else if r.RegionRequirement != "" {
			name = "region"
			val = r.RegionRequirement
			tpe = sdk.RegionRequirement
		} else if r.Plugin != "" {
			name = r.Plugin
			val = r.Plugin
			tpe = sdk.PluginRequirement
		} else if r.Service.Name != "" {
			name = r.Service.Name
			val = r.Service.Value
			tpe = sdk.ServiceRequirement
		}
		res[i] = sdk.Requirement{
			Name:  name,
			Type:  tpe,
			Value: val,
		}
	}
	return res
}

func computeJob(name string, j Job) (*sdk.Job, error) {
	job := sdk.Job{
		Action: sdk.Action{
			Name:        name,
			Description: j.Description,
			Type:        sdk.JoinedAction,
		},
	}
	if j.Enabled != nil {
		job.Enabled = *j.Enabled
	} else {
		job.Enabled = true
	}
	job.Action.Enabled = job.Enabled
	job.Action.Requirements = computeJobRequirements(j.Requirements)

	//Compute steps for the jobs
	children, err := computeSteps(j.Steps)
	if err != nil {
		return nil, err
	}
	job.Action.Actions = children

	return &job, nil
}

//Pipeline returns a sdk.Pipeline entity
func (p PipelineV1) Pipeline() (pip *sdk.Pipeline, err error) {
	pip = new(sdk.Pipeline)
	pip.Name = p.Name
	pip.Description = p.Description

	pip.Parameter = make([]sdk.Parameter, 0, len(p.Parameters))
	//Compute parameters
	for p, v := range p.Parameters {
		param := sdk.Parameter{
			Name:        p,
			Type:        v.Type,
			Value:       v.DefaultValue,
			Description: v.Description,
		}
		if param.Type == "" {
			param.Type = sdk.StringParameter
		}
		pip.Parameter = append(pip.Parameter, param)
	}

	//Compute stage
	mapStages := map[string]*sdk.Stage{}
	for i, s := range p.Stages {
		mapStages[s] = &sdk.Stage{
			Name:       s,
			BuildOrder: i + 1, //Yes, buildOrder start at 1
			Enabled:    true,
		}
	}

	for s, opt := range p.StageOptions {
		if mapStages[s] == nil {
			return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid stage option, stage %s not found", s)
		}
		if opt.Enabled != nil {
			mapStages[s].Enabled = *opt.Enabled
		} else {
			mapStages[s].Enabled = true
		}

		if opt.Conditions != nil {
			mapStages[s].Conditions = *opt.Conditions
		}
	}

	//Compute Jobs
	for _, j := range p.Jobs {
		s := mapStages[j.Stage]
		if s == nil { //If the stage is not found
			if len(mapStages) != 1 || j.Stage != "" { // if there isn't only one stage
				mapStages[j.Stage] = &sdk.Stage{
					Name:       j.Stage,
					BuildOrder: len(mapStages) + 1,
					Enabled:    true,
				}
				s = mapStages[j.Stage]
			} else {
				// choose the only one stage
				for _, stage := range mapStages {
					s = stage
					break
				}
			}
		}

		job, err := computeJob(j.Name, j)
		if err != nil {
			return pip, err
		}
		s.Jobs = append(s.Jobs, *job)
	}

	pip.Stages = make([]sdk.Stage, len(mapStages))
	iS := 0
	for _, s := range mapStages {
		pip.Stages[iS] = *s
		iS++
	}

	sort.Slice(pip.Stages, func(i, j int) bool {
		return pip.Stages[i].BuildOrder < pip.Stages[j].BuildOrder
	})

	return pip, nil
}

// ParsePipeline returns a pipeliner from given data.
func ParsePipeline(format Format, data []byte) (Pipeliner, error) {
	rawPayload := make(map[string]interface{})
	if err := Unmarshal(data, format, &rawPayload); err != nil {
		return nil, err
	}

	version := PipelineVersion1
	if v, ok := rawPayload["version"]; ok {
		switch v.(string) {
		case PipelineVersion1:
			version = PipelineVersion1
		default:
			return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid pipeline version")
		}
	}

	var payload Pipeliner
	switch version {
	case PipelineVersion1:
		payload = &PipelineV1{}
	}
	if err := Unmarshal(data, format, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
