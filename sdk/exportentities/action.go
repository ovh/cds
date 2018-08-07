package exportentities

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/sdk"
)

// Action represents exported sdk.Action
type Action struct {
	Version      string                    `json:"version,omitempty" yaml:"version,omitempty"`
	Name         string                    `json:"name,omitempty" yaml:"name,omitempty"`
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
func NewAction(act sdk.Action) (a Action) {
	a.Name = act.Name
	a.Version = ActionVersion1
	a.Description = act.Description
	a.Parameters = make(map[string]ParameterValue, len(act.Parameters))
	for k, v := range act.Parameters {
		param := ParameterValue{
			Type:         string(v.Type),
			DefaultValue: v.Value,
			Description:  v.Description,
		}
		// no need to export it if "Advanced" is false
		if v.Advanced {
			param.Advanced = &act.Parameters[k].Advanced
		}
		a.Parameters[v.Name] = param
	}
	a.Steps = newSteps(act)
	a.Requirements = newRequirements(act.Requirements)
	// enabled is the default value
	// set enable attribute only if it's disabled
	// no need to export it if action is enabled
	if !act.Enabled {
		a.Enabled = &act.Enabled
	}
	return a
}

func newSteps(a sdk.Action) []Step {
	res := make([]Step, len(a.Actions))
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
				script := sdk.ParameterFind(&act.Parameters, "script")
				if script != nil {
					s["script"] = strings.SplitN(script.Value, "\n", -1)
				}
			case sdk.CoverageAction:
				coverageArgs := map[string]string{}
				path := sdk.ParameterFind(&act.Parameters, "path")
				if path != nil {
					coverageArgs["path"] = path.Value
				}
				format := sdk.ParameterFind(&act.Parameters, "format")
				if format != nil {
					coverageArgs["format"] = format.Value
				}
				s["coverage"] = coverageArgs
			case sdk.ArtifactDownload:
				artifactDownloadArgs := map[string]string{}
				path := sdk.ParameterFind(&act.Parameters, "path")
				if path != nil {
					artifactDownloadArgs["path"] = path.Value
				}
				tag := sdk.ParameterFind(&act.Parameters, "tag")
				if tag != nil {
					artifactDownloadArgs["tag"] = tag.Value
				}
				application := sdk.ParameterFind(&act.Parameters, "application")
				if application != nil {
					artifactDownloadArgs["application"] = application.Value
				}
				pipeline := sdk.ParameterFind(&act.Parameters, "pipeline")
				if pipeline != nil {
					artifactDownloadArgs["pipeline"] = pipeline.Value
				}
				s["artifactDownload"] = artifactDownloadArgs
			case sdk.ArtifactUpload:
				artifactUploadArgs := map[string]string{}
				path := sdk.ParameterFind(&act.Parameters, "path")
				if path != nil {
					artifactUploadArgs["path"] = path.Value
				}
				tag := sdk.ParameterFind(&act.Parameters, "tag")
				if tag != nil {
					artifactUploadArgs["tag"] = tag.Value
				}

				s["artifactUpload"] = artifactUploadArgs
			case sdk.GitCloneAction:
				gitCloneArgs := map[string]string{}
				branch := sdk.ParameterFind(&act.Parameters, "branch")
				if branch != nil {
					gitCloneArgs["branch"] = branch.Value
				}
				commit := sdk.ParameterFind(&act.Parameters, "commit")
				if commit != nil {
					gitCloneArgs["commit"] = commit.Value
				}
				directory := sdk.ParameterFind(&act.Parameters, "directory")
				if directory != nil {
					gitCloneArgs["directory"] = directory.Value
				}
				password := sdk.ParameterFind(&act.Parameters, "password")
				if password != nil {
					gitCloneArgs["password"] = password.Value
				}
				privateKey := sdk.ParameterFind(&act.Parameters, "privateKey")
				if privateKey != nil {
					gitCloneArgs["privateKey"] = privateKey.Value
				}
				url := sdk.ParameterFind(&act.Parameters, "url")
				if url != nil {
					gitCloneArgs["url"] = url.Value
				}
				user := sdk.ParameterFind(&act.Parameters, "user")
				if user != nil {
					gitCloneArgs["user"] = user.Value
				}

				s["gitClone"] = gitCloneArgs
			case sdk.JUnitAction:
				path := sdk.ParameterFind(&act.Parameters, "path")
				if path != nil {
					s["jUnitReport"] = path.Value
				}
			case sdk.CheckoutApplicationAction:
				directory := sdk.ParameterFind(&act.Parameters, "directory")
				if directory != nil {
					s["checkout"] = directory.Value
				}
			case sdk.DeployApplicationAction:
				s["deploy"] = "{{.cds.application}}"
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
		res[i] = s
	}

	return res
}

//AsScript returns the step a sdk.Action
func (s Step) AsScript() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("AsScript.Malformatted Step")
	}

	bI, ok := s["script"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)

	if !ok {
		var arScript []interface{}
		arScript, ok = bI.([]interface{})
		var asScriptString = make([]string, len(arScript))
		for i, s := range arScript {
			asScriptString[i], ok = s.(string)
			if !ok {
				break
			}
		}
		bS = strings.Join(asScriptString, "\n")
	}

	if !ok {
		return nil, true, fmt.Errorf("Malformatted Step : script must be a string or a string array")
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
		return nil, false, fmt.Errorf("AsAction.Malformatted Step")
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
		return nil, true, sdk.WrapError(err, "AsAction.decode.Malformatted Step")
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
		return nil, false, fmt.Errorf("AsJUnitReport.Malformatted Step")
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
		return nil, false, fmt.Errorf("AsGitClone.Malformatted Step")
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
		return nil, true, sdk.WrapError(err, "AsGitClone.Malformatted Step")
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
		return nil, false, fmt.Errorf("AsArtifactUpload.Malformatted Step")
	}

	bI, ok := s["artifactUpload"]
	if !ok {
		return nil, false, nil
	}

	if reflect.ValueOf(bI).Kind() != reflect.Map && reflect.ValueOf(bI).Kind() != reflect.String {
		return nil, false, nil
	}

	var a sdk.Action
	if s, ok := bI.(string); ok {
		a = sdk.NewStepArtifactUpload(s)
	} else if m, ok := bI.(map[interface{}]interface{}); ok {
		argss := map[string]string{}
		if err := mapstructure.Decode(m, &argss); err != nil {
			return nil, true, sdk.WrapError(err, "AsArtifactUpload.decode>Malformatted Step")
		}
		a = sdk.NewStepArtifactUpload(argss)
	} else {
		return nil, false, fmt.Errorf("AsArtifactUpload> Unknown type")
	}

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
		return nil, false, fmt.Errorf("AsArtifactDownload.Malformatted Step")
	}

	bI, ok := s["artifactDownload"]
	if !ok {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WrapError(err, "AsArtifactDownload.decode> Malformatted Step")
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

//AsCheckoutApplication returns the step as a sdk.Action
func (s Step) AsCheckoutApplication() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("AsCheckoutApplication> Malformatted Step")
	}
	bI, ok := s["checkout"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)
	if !ok {
		return nil, true, fmt.Errorf("Malformatted Step : checkout must be a string")
	}
	a := sdk.NewCheckoutApplication(bS)

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

//AsCoverageAction returns the step as a sdk.Action
func (s Step) AsCoverageAction() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("AsCoverageAction> Malformatted Step")
	}
	bI, ok := s["coverage"]
	if !ok {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WrapError(err, "AsCoverageAction.decode> Malformatted Step")
	}
	a := sdk.NewCoverage(argss)

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

//AsDeployApplication returns the step as a sdk.Action
func (s Step) AsDeployApplication() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, fmt.Errorf("AsDeployApplication> Malformatted Step")
	}
	bI, ok := s["deploy"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)
	if !ok {
		return nil, true, fmt.Errorf("Malformatted Step : deploy must be a string")
	}
	a := sdk.NewDeployApplication(bS)

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

// Action returns an sdk.Action
func (act *Action) Action() (*sdk.Action, error) {
	a := new(sdk.Action)
	a.Name = act.Name
	a.Type = sdk.DefaultAction
	a.Description = act.Description

	//Compute parameters
	a.Parameters = make([]sdk.Parameter, len(act.Parameters))
	var i int
	for p, v := range act.Parameters {
		param := sdk.Parameter{
			Name:        p,
			Type:        v.Type,
			Value:       v.DefaultValue,
			Description: v.Description,
		}
		if param.Type == "" {
			param.Type = sdk.StringParameter
		}
		if v.Advanced != nil && *v.Advanced {
			param.Advanced = true
		}
		a.Parameters[i] = param
		i++
	}

	a.Requirements = computeJobRequirements(act.Requirements)

	//Compute steps for the jobs
	children, err := computeSteps(act.Steps)
	if err != nil {
		return nil, err
	}

	a.Actions = children

	// if no flag enabled: the default value is true
	if act.Enabled == nil {
		a.Enabled = true
	} else {
		a.Enabled = *act.Enabled
	}
	return a, nil
}
