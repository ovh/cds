package workflowv3

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

type Jobs map[string]Job

func (j Jobs) FilterByStage(stageName string) Jobs {
	filtered := make(Jobs, len(j))
	for jName, job := range j {
		if job.Stage == stageName {
			filtered[jName] = job
		}
	}
	return filtered
}

func (j Jobs) ExistJob(jobName string) bool {
	_, ok := j[jobName]
	return ok
}

func (j Jobs) ToGraphs() []Graph {
	stageGraphs := make(map[string]Graph)
	for jName, job := range j {
		stageGraphs[job.Stage] = append(stageGraphs[job.Stage], Node{
			Name:      jName,
			DependsOn: job.DependsOn,
		})
	}
	var res []Graph
	for _, g := range stageGraphs {
		res = append(res, g)
	}
	return res
}

type Job struct {
	ID           string                       `json:"-" yaml:"-"`
	Enabled      *bool                        `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Description  string                       `json:"description,omitempty" yaml:"description,omitempty"`
	Conditions   *Condition                   `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Context      []ContextRaw                 `json:"context,omitempty" yaml:"context,omitempty"` // can be @
	Stage        string                       `json:"stage,omitempty" yaml:"stage,omitempty"`
	Steps        []Step                       `json:"steps,omitempty" yaml:"steps,omitempty"`
	Requirements []exportentities.Requirement `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	DependsOn    []string                     `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}

func (j Job) Validate(w Workflow) (ExternalDependencies, error) {
	var extDep ExternalDependencies

	// Graph validation
	useStages := len(w.Stages) > 0
	if useStages && j.Stage == "" {
		return extDep, fmt.Errorf("stage name is required")
	}
	if j.Stage != "" {
		if _, ok := w.Stages[j.Stage]; !ok {
			return extDep, fmt.Errorf("invalid workflow stage %q", j.Stage)
		}
	}
	if len(j.DependsOn) > 0 {
		for _, d := range j.DependsOn {
			if j.Stage != "" {
				if !w.Jobs.FilterByStage(j.Stage).ExistJob(d) {
					return extDep, fmt.Errorf("depends on unknown job %q for stage %q", d, j.Stage)
				}
			} else if !w.Jobs.ExistJob(d) {
				return extDep, fmt.Errorf("depends on unknown job %q", d)
			}
		}
	}

	// Context validation
	for _, c := range j.Context {
		isExternal, cType, cName, err := c.Parse()
		if err != nil {
			return extDep, err
		}
		if isExternal {
			switch cType {
			case ContextTypeRepository:
				extDep.Repositories = append(extDep.Repositories, cName)
			case ContextTypeVar:
				extDep.Variables = append(extDep.Variables, cName)
			case ContextTypeSecret:
				extDep.Secrets = append(extDep.Secrets, cName)
			}
		} else {
			switch cType {
			case ContextTypeRepository:
				if !w.Repositories.ExistRepo(cName) {
					return extDep, fmt.Errorf("requires unknown %q context %q", cType, cName)
				}
			case ContextTypeVar:
				if !w.Variables.ExistVariable(cName) {
					return extDep, fmt.Errorf("requires unknown %q context %q", cType, cName)
				}
			case ContextTypeSecret:
				if !w.Secrets.ExistSecret(cName) {
					return extDep, fmt.Errorf("requires unknown %q context %q", cType, cName)
				}
			default:
				return extDep, fmt.Errorf("invalid context type %q", cType)
			}
		}
	}

	// Steps validation
	for i, s := range j.Steps {
		dep, err := s.Validate(w)
		if err != nil {
			return extDep, errors.WithMessagef(err, "step %d", i)
		}
		extDep.Add(dep)
	}

	return extDep, nil
}

type ContextType string

const (
	ContextTypeRepository ContextType = "repository"
	ContextTypeVar        ContextType = "var"
	ContextTypeSecret     ContextType = "secret"
)

func (c ContextType) Validate() error {
	switch c {
	case ContextTypeRepository, ContextTypeVar, ContextTypeSecret:
		return nil
	default:
		return fmt.Errorf("invalid given context type %q", c)
	}
}

type ContextRaw string

func (c ContextRaw) Parse() (bool, ContextType, string, error) {
	splitted := strings.SplitN(string(c), ".", 2)
	if len(splitted) < 2 {
		return false, "", "", fmt.Errorf("invalid given context ref %q, should be formatted like: \"context-type.name\"", c)
	}
	contextType := ContextType(strings.TrimPrefix(splitted[0], "@"))
	isExternal := string(contextType) != splitted[0]
	return isExternal, contextType, splitted[1], contextType.Validate()
}

func ConvertJob(j sdk.Job, isFullExport bool) Job {
	jo := Job{}
	if !j.Enabled {
		jo.Enabled = &j.Enabled
	}

	if !isFullExport {
		return jo
	}

	jo.Steps = make([]Step, len(j.Action.Actions))
	for i := range j.Action.Actions {
		s := exportentities.NewStep(j.Action.Actions[i])
		stepCustom := make(exportentities.StepCustom, len(s.StepCustom))
		for k, v := range s.StepCustom {
			stepCustom["@"+k] = v
		}
		jo.Steps[i] = Step{
			StepCustom:       stepCustom,
			Coverage:         s.Coverage,
			ArtifactDownload: s.ArtifactDownload,
			ArtifactUpload:   s.ArtifactUpload,
			ServeStaticFiles: s.ServeStaticFiles,
			GitClone:         s.GitClone,
			GitTag:           s.GitTag,
			ReleaseVCS:       s.ReleaseVCS,
			JUnitReport:      s.JUnitReport,
			Checkout:         s.Checkout,
			InstallKey:       s.InstallKey,
			Deploy:           s.Deploy,
			Release:          s.Release,
			PushBuildInfo:    s.PushBuildInfo,
			Promote:          s.Promote,
		}
		if s.Script != nil {
			var script StepScript
			if lines, ok := s.Script.([]string); ok {
				script = StepScript(strings.Join(lines, "\n"))
				// Display script as strings slice if it can't be marshaled to multiline yaml string
				bf, err := yaml.Marshal(script)
				if err == nil && !strings.HasPrefix(string(bf), "|-") {
					script = StepScript(lines)
				}
			} else {
				script = StepScript(s.Script.(string))
			}
			jo.Steps[i].Script = &script
		}
	}

	jo.Description = j.Action.Description
	jo.Requirements = exportentities.NewRequirements(j.Action.Requirements)
	return jo
}
