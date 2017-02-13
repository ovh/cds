package exportentities

import "github.com/ovh/cds/sdk"

// Pipeline represents exported sdk.Pipeline
type Pipeline struct {
	Name        string                    `json:"name" yaml:"name"`
	Type        string                    `json:"type" yaml:"type"`
	Permissions map[string]int            `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Parameters  map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Stages      map[string]Stage          `json:"stages,omitempty" yaml:"stages,omitempty"`
	Jobs        map[string]Job            `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Step        []Step                    `json:"step,omitempty" yaml:"steps,omitempty"`
}

// Stage represents exported sdk.Stage
type Stage struct {
	name    string
	Enabled *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Order   int            `json:"order,omitempty" yaml:"order,omitempty"`
	Job     map[string]Job `json:"job,omitempty" yaml:"job,omitempty"`
}

// Job represents exported sdk.Job
type Job struct {
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled      *bool         `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Steps        []Step        `json:"step,omitempty" yaml:"steps,omitempty"`
	Requirements []Requirement `json:"requirements,omitempty" yaml:"requirements,omitempty"`
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
		switch len(pip.Stages[0].Jobs) {
		case 0:
			return
		case 1:
			p.Step = newSteps(pip.Stages[0].Jobs[0].Action)
			return
		default:
			p.Jobs = newJobs(pip.Stages[0].Jobs)
		}
	default:
		p.Stages = newStages(pip.Stages)
	}

	return
}

func newStages(stages []sdk.Stage) map[string]Stage {
	res := map[string]Stage{}
	var order int
	for _, s := range stages {
		if len(s.Jobs) == 0 {
			continue
		}
		order++
		st := Stage{
			Order: order,
		}
		if !s.Enabled {
			st.Enabled = &s.Enabled
		}
		st.Job = newJobs(s.Jobs)
		st.name = s.Name
		res[s.Name] = st
	}
	return res
}

func newJobs(jobs []sdk.Job) map[string]Job {
	res := map[string]Job{}
	for _, j := range jobs {
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
	for _, a := range a.Actions {
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
