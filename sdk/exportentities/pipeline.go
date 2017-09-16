package exportentities

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/sdk"
)

// Pipeline represents exported sdk.Pipeline
type Pipeline struct {
	Name         string                    `json:"name,omitempty" yaml:"name,omitempty"`
	Type         string                    `json:"type,omitempty" yaml:"type,omitempty"`
	Permissions  map[string]int            `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Parameters   map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Stages       map[string]Stage          `json:"stages,omitempty" yaml:"stages,omitempty"`
	Jobs         map[string]Job            `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Requirements []Requirement             `json:"requirements,omitempty" yaml:"requirements,omitempty" hcl:"requirement,omitempty"`
	Steps        []Step                    `json:"steps,omitempty" yaml:"steps,omitempty" hcl:"step,omitempty"`
}

// Stage represents exported sdk.Stage
type Stage struct {
	Enabled    *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Jobs       map[string]Job    `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Conditions map[string]string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Job represents exported sdk.Job
type Job struct {
	Description    string        `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled        *bool         `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Steps          []Step        `json:"steps,omitempty" yaml:"steps,omitempty" hcl:"step,omitempty"`
	Requirements   []Requirement `json:"requirements,omitempty" yaml:"requirements,omitempty" hcl:"requirement,omitempty"`
	Optional       *bool         `json:"optional,omitempty" yaml:"optional,omitempty" hcl:"optional,omitempty"`
	AlwaysExecuted *bool         `json:"always_executed,omitempty" yaml:"always_executed,omitempty" hcl:"always_executed,omitempty"`
}

// Step represents exported step used in a job
type Step map[string]interface{}

// IsValid returns true is the step is valid
func (s Step) IsValid() bool {
	keys := []string{}
	for k := range s {
		if k != "enabled" && k != "optional" && k != "always_executed" {
			keys = append(keys, k)
		}
	}
	return len(keys) == 1
}

func (s Step) key() string {
	keys := []string{}
	for k := range s {
		if k != "enabled" && k != "optional" && k != "always_executed" {
			keys = append(keys, k)
		}
	}
	return keys[0]
}

//AsScript returns the step a sdk.Action
func (s Step) AsScript() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("Malformatted Step")
	}

	bI, ok := s["script"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)
	if !ok {
		return nil, true, fmt.Errorf("Malformatted Step : script must be a string")
	}

	a := sdk.NewStepScript(bS)

	var err error
	a.Enabled, err = s.IsFlagged("enabled")
	if err != nil {
		return nil, true, err
	}
	a.Optional, err = s.IsFlagged("optional")
	if err != nil {
		return nil, true, err
	}
	a.AlwaysExecuted, err = s.IsFlagged("always_executed")
	if err != nil {
		return nil, true, err
	}

	return &a, true, nil
}

//AsAction returns the step a sdk.Action
func (s Step) AsAction() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("Malformatted Step")
	}

	actionName := s.key()

	bI, ok := s[actionName]
	if !ok {
		return nil, false, nil
	}

	if reflect.ValueOf(bI).Kind() != reflect.Map {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WrapError(err, "Malformatted Step")
	}

	a, err := sdk.NewStepDefault(actionName, argss)
	if err != nil {
		return nil, true, err
	}

	a.Enabled, err = s.IsFlagged("enabled")
	if err != nil {
		return nil, true, err
	}
	a.Optional, err = s.IsFlagged("optional")
	if err != nil {
		return nil, true, err
	}
	a.AlwaysExecuted, err = s.IsFlagged("always_executed")
	if err != nil {
		return nil, true, err
	}
	return a, true, nil
}

//AsJUnitReport returns the step a sdk.Action
func (s Step) AsJUnitReport() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("Malformatted Step")
	}

	bI, ok := s["jUnitReport"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)
	if !ok {
		return nil, true, fmt.Errorf("Malformatted Step : jUnitReport must be a string")
	}

	a := sdk.NewStepJUnitReport(bS)

	var err error
	a.Enabled, err = s.IsFlagged("enabled")
	if err != nil {
		return nil, true, err
	}
	a.AlwaysExecuted, err = s.IsFlagged("always_executed")
	if err != nil {
		return nil, true, err
	}
	a.Optional, err = s.IsFlagged("optional")
	if err != nil {
		return nil, true, err
	}

	return &a, true, nil
}

//AsGitClone returns the step a sdk.Action
func (s Step) AsGitClone() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("Malformatted Step")
	}

	bI, ok := s["gitClone"]
	if !ok {
		return nil, false, nil
	}

	if reflect.ValueOf(bI).Kind() != reflect.Map {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WrapError(err, "Malformatted Step")
	}

	a := sdk.NewStepGitClone(argss)

	var err error
	a.Enabled, err = s.IsFlagged("enabled")
	if err != nil {
		return nil, true, err
	}
	a.Optional, err = s.IsFlagged("optional")
	if err != nil {
		return nil, true, err
	}
	a.AlwaysExecuted, err = s.IsFlagged("always_executed")
	if err != nil {
		return nil, true, err
	}

	return &a, true, nil
}

//AsArtifactUpload returns the step a sdk.Action
func (s Step) AsArtifactUpload() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("Malformatted Step")
	}

	bI, ok := s["artifactUpload"]
	if !ok {
		return nil, false, nil
	}

	if reflect.ValueOf(bI).Kind() != reflect.Map {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WrapError(err, "Malformatted Step")
	}

	a := sdk.NewStepArtifactUpload(argss)

	var err error
	a.Enabled, err = s.IsFlagged("enabled")
	if err != nil {
		return nil, true, err
	}
	a.Optional, err = s.IsFlagged("optional")
	if err != nil {
		return nil, true, err
	}
	a.AlwaysExecuted, err = s.IsFlagged("always_executed")
	if err != nil {
		return nil, true, err
	}

	return &a, true, nil
}

//AsArtifactDownload returns the step a sdk.Action
func (s Step) AsArtifactDownload() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("Malformatted Step")
	}

	bI, ok := s["artifactDownload"]
	if !ok {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WrapError(err, "Malformatted Step")
	}
	a := sdk.NewStepArtifactDownload(argss)

	var err error
	a.Enabled, err = s.IsFlagged("enabled")
	if err != nil {
		return nil, true, err
	}
	a.Optional, err = s.IsFlagged("optional")
	if err != nil {
		return nil, true, err
	}
	a.AlwaysExecuted, err = s.IsFlagged("always_executed")
	if err != nil {
		return nil, true, err
	}

	return &a, true, nil
}

// IsFlagged returns true the step has the flag set
func (s Step) IsFlagged(flag string) (bool, error) {
	bI, ok := s[flag]
	if !ok {
		// enabled is true by default
		return flag == "enabled", nil
	}
	bS, ok := bI.(bool)
	if !ok {
		return false, fmt.Errorf("Malformatted Step : %s attribute must be true|false", flag)
	}
	return bS, nil
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

	// Default name is like the type
	if strings.ToLower(pip.Name) != pip.Type {
		p.Name = pip.Name
	}

	// We consider build pipeline are default
	if pip.Type != sdk.BuildPipeline {
		p.Type = pip.Type
	}

	if len(pip.GroupPermission) > 0 {
		p.Permissions = make(map[string]int, len(pip.GroupPermission))
		for _, perm := range pip.GroupPermission {
			p.Permissions[perm.Group.Name] = perm.Permission
		}
	}

	if len(pip.Parameter) > 0 {
		p.Parameters = make(map[string]ParameterValue, len(pip.Parameter))
		for _, v := range pip.Parameter {
			p.Parameters[v.Name] = ParameterValue{
				Type:         string(v.Type),
				DefaultValue: v.Value,
			}
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
				p.Requirements = newRequirements(pip.Stages[0].Jobs[0].Action.Requirements)
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

func newRequirements(req []sdk.Requirement) []Requirement {
	if req == nil {
		return nil
	}
	res := []Requirement{}
	for _, r := range req {
		switch r.Type {
		case sdk.BinaryRequirement:
			res = append(res, Requirement{Binary: r.Value})
		case sdk.NetworkAccessRequirement:
			res = append(res, Requirement{Network: r.Value})
		case sdk.ModelRequirement:
			res = append(res, Requirement{Model: r.Value})
		case sdk.HostnameRequirement:
			res = append(res, Requirement{Hostname: r.Value})
		case sdk.PluginRequirement:
			res = append(res, Requirement{Plugin: r.Value})
		case sdk.ServiceRequirement:
			res = append(res, Requirement{Service: ServiceRequirement{Name: r.Name, Value: r.Value}})
		case sdk.MemoryRequirement:
			res = append(res, Requirement{Memory: r.Value})
		}
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
		jo.Requirements = newRequirements(j.Action.Requirements)
		res[j.Action.Name] = jo
	}
	return res
}

func newSteps(a sdk.Action) []Step {
	res := []Step{}
	for i := range a.Actions {
		act := &a.Actions[i]
		s := Step{}
		if !act.Enabled {
			s["enabled"] = act.Enabled
		}
		if act.Optional {
			s["optional"] = act.Optional
		}
		if act.AlwaysExecuted {
			s["always_executed"] = act.AlwaysExecuted
		}

		switch act.Type {
		case sdk.BuiltinAction:
			switch act.Name {
			case sdk.ScriptAction:
				script := sdk.ParameterFind(act.Parameters, "script")
				if script != nil {
					s["script"] = script.Value
				}
			case sdk.ArtifactDownload:
				artifactDownloadArgs := map[string]string{}
				path := sdk.ParameterFind(act.Parameters, "path")
				if path != nil {
					artifactDownloadArgs["path"] = path.Value
				}
				tag := sdk.ParameterFind(act.Parameters, "tag")
				if tag != nil {
					artifactDownloadArgs["tag"] = tag.Value
				}
				application := sdk.ParameterFind(act.Parameters, "application")
				if application != nil {
					artifactDownloadArgs["application"] = application.Value
				}
				pipeline := sdk.ParameterFind(act.Parameters, "pipeline")
				if pipeline != nil {
					artifactDownloadArgs["pipeline"] = pipeline.Value
				}
				s["artifactDownload"] = artifactDownloadArgs
			case sdk.ArtifactUpload:
				artifactUploadArgs := map[string]string{}
				path := sdk.ParameterFind(act.Parameters, "path")
				if path != nil {
					artifactUploadArgs["path"] = path.Value
				}
				tag := sdk.ParameterFind(act.Parameters, "tag")
				if tag != nil {
					artifactUploadArgs["tag"] = tag.Value
				}
				s["artifactUpload"] = artifactUploadArgs
			case sdk.GitCloneAction:
				gitCloneArgs := map[string]string{}
				branch := sdk.ParameterFind(act.Parameters, "branch")
				if branch != nil {
					gitCloneArgs["branch"] = branch.Value
				}
				commit := sdk.ParameterFind(act.Parameters, "commit")
				if commit != nil {
					gitCloneArgs["commit"] = commit.Value
				}
				directory := sdk.ParameterFind(act.Parameters, "directory")
				if directory != nil {
					gitCloneArgs["directory"] = directory.Value
				}
				password := sdk.ParameterFind(act.Parameters, "password")
				if password != nil {
					gitCloneArgs["password"] = password.Value
				}
				privateKey := sdk.ParameterFind(act.Parameters, "privateKey")
				if privateKey != nil {
					gitCloneArgs["privateKey"] = privateKey.Value
				}
				url := sdk.ParameterFind(act.Parameters, "url")
				if url != nil {
					gitCloneArgs["url"] = url.Value
				}
				user := sdk.ParameterFind(act.Parameters, "user")
				if user != nil {
					gitCloneArgs["user"] = user.Value
				}

				s["gitClone"] = gitCloneArgs
			case sdk.JUnitAction:
				path := sdk.ParameterFind(act.Parameters, "path")
				if path != nil {
					s["jUnitReport"] = path.Value
				}
			}
		default:
			args := map[string]string{}
			for _, p := range act.Parameters {
				if p.Value != "" {
					args[p.Name] = p.Value
				}
			}
			s[act.Name] = args
		}
		res = append(res, s)
	}

	return res
}

//Pipeline returns a sdk.Pipeline entity
func (p *Pipeline) Pipeline() (*sdk.Pipeline, error) {
	pip := new(sdk.Pipeline)

	if p.Type == "" {
		p.Type = sdk.BuildPipeline
	}

	if p.Name == "" {
		p.Name = strings.Title(p.Type)
	}

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
			Type:  v.Type,
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
							Enabled:      true,
							Name:         p.Name,
							Actions:      actions,
							Type:         sdk.JoinedAction,
							Requirements: computeJobRequirements(p.Requirements),
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
		res = append(res, *a)
	}
	return res, nil
}

func computeStep(s Step) (a *sdk.Action, e error) {
	if !s.IsValid() {
		e = fmt.Errorf("Malformatted step")
		return
	}

	var ok bool
	a, ok, e = s.AsArtifactDownload()
	if ok {
		return
	}

	a, ok, e = s.AsArtifactUpload()
	if ok {
		return
	}

	a, ok, e = s.AsJUnitReport()
	if ok {
		return
	}

	a, ok, e = s.AsGitClone()
	if ok {
		return
	}

	a, ok, e = s.AsScript()
	if ok {
		return
	}

	a, ok, e = s.AsAction()
	if ok {
		return
	}

	return
}

func computeJobRequirements(req []Requirement) []sdk.Requirement {
	res := []sdk.Requirement{}
	for _, r := range req {
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
		res = append(res, sdk.Requirement{
			Name:  name,
			Type:  tpe,
			Value: val,
		})
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
