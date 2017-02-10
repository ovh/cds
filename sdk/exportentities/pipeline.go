package exportentities

import "github.com/ovh/cds/sdk"

// Pipeline represents exported sdk.Pipeline
type Pipeline struct {
	Name        string                    `json:"name" yaml:"name"`
	Type        string                    `json:"string" yaml:"string"`
	Permissions map[string]int            `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Parameters  map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Stages      map[string]Stage          `json:"stages,omitempty" yaml:"stages,omitempty"`
	Jobs        map[string]Job            `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Step        []Step                    `json:"step,omitempty" yaml:"step,omitempty"`
}

// Stage represents exported sdk.Stage
type Stage struct {
	Enabled *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Order   int            `json:"order,omitempty" yaml:"order,omitempty"`
	Job     map[string]Job `json:"job,omitempty" yaml:"job,omitempty"`
}

// Job represents exported sdk.Job
type Job struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Step        []Step `json:"job,omitempty" yaml:"job,omitempty"`
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

//NewPipeline creates an exportable pipeline from a sdk.Pipeline
func NewPipeline(pip sdk.Pipeline) (p *Pipeline) {
	p.Name = pip.Name
	p.Type = string(p.Type)
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
		jo.Step = newSteps(j.Action)
		jo.Description = j.Action.Description
		res[j.Action.Name] = jo
	}
	return res
}

func newSteps(a sdk.Action) []Step {
	res := make([]Step, len(a.Actions))
	var i int
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

			case sdk.ArtifactDownload:

			case sdk.ArtifactUpload:
				s.ArtifactUpload = map[string]string{
					"path": "",
				}
			case sdk.JUnitAction:

			}
		case sdk.PluginAction:

		default:
			s.Action = map[string]map[string]string{}
			s.Action[a.Name] = map[string]string{}
			for _, p := range a.Parameters {
				s.Action[a.Name][p.Name] = p.Value
			}
		}

		res[i] = s
	}

	return res
}
