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
	ea.Requirements = newRequirements(a.Requirements)
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
		act := &a.Actions[i]
		s := Step{}
		if act.StepName != "" {
			s["name"] = act.StepName
		}
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
				minimum := sdk.ParameterFind(&act.Parameters, "minimum")
				if minimum != nil {
					coverageArgs["minimum"] = minimum.Value
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
				if application != nil && application.Value != "" {
					artifactDownloadArgs["application"] = application.Value
				}
				pattern := sdk.ParameterFind(&act.Parameters, "pattern")
				if pattern != nil && pattern.Value != "" {
					artifactDownloadArgs["pattern"] = pattern.Value
				}
				enabled := sdk.ParameterFind(&act.Parameters, "enabled")
				if enabled != nil && enabled.Value == "false" {
					artifactDownloadArgs["enabled"] = enabled.Value
				}
				pipeline := sdk.ParameterFind(&act.Parameters, "pipeline")
				if pipeline != nil && pipeline.Value != "" {
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
				destination := sdk.ParameterFind(&act.Parameters, "destination")
				if destination != nil {
					artifactUploadArgs["destination"] = destination.Value
				}
				s["artifactUpload"] = artifactUploadArgs
			case sdk.ServeStaticFiles:
				serveStaticFilesArgs := map[string]string{}
				name := sdk.ParameterFind(&act.Parameters, "name")
				if name != nil {
					serveStaticFilesArgs["name"] = name.Value
				}
				path := sdk.ParameterFind(&act.Parameters, "path")
				if path != nil {
					serveStaticFilesArgs["path"] = path.Value
				}
				entrypoint := sdk.ParameterFind(&act.Parameters, "entrypoint")
				if entrypoint != nil && entrypoint.Value != "" {
					serveStaticFilesArgs["entrypoint"] = entrypoint.Value
				}
				staticKey := sdk.ParameterFind(&act.Parameters, "static-key")
				if staticKey != nil && staticKey.Value != "" {
					serveStaticFilesArgs["static-key"] = staticKey.Value
				}
				destination := sdk.ParameterFind(&act.Parameters, "destination")
				if destination != nil {
					serveStaticFilesArgs["destination"] = destination.Value
				}
				s["serveStaticFiles"] = serveStaticFilesArgs
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
				if password != nil && password.Value != "" {
					gitCloneArgs["password"] = password.Value
				}
				privateKey := sdk.ParameterFind(&act.Parameters, "privateKey")
				if privateKey != nil && privateKey.Value != "" {
					gitCloneArgs["privateKey"] = privateKey.Value
				}
				url := sdk.ParameterFind(&act.Parameters, "url")
				if url != nil && url.Value != "" {
					gitCloneArgs["url"] = url.Value
				}
				user := sdk.ParameterFind(&act.Parameters, "user")
				if user != nil && user.Value != "" {
					gitCloneArgs["user"] = user.Value
				}
				depth := sdk.ParameterFind(&act.Parameters, "depth")
				if depth != nil && depth.Value != "" && depth.Value != "50" {
					gitCloneArgs["depth"] = depth.Value
				}
				submodules := sdk.ParameterFind(&act.Parameters, "submodules")
				if submodules != nil && submodules.Value == "false" {
					gitCloneArgs["submodules"] = submodules.Value
				}
				tag := sdk.ParameterFind(&act.Parameters, "tag")
				if tag != nil && tag.Value != "" && tag.Value != sdk.DefaultGitCloneParameterTagValue {
					gitCloneArgs["tag"] = tag.Value
				}
				s["gitClone"] = gitCloneArgs
			case sdk.GitTagAction:
				gitTagArgs := map[string]string{}
				path := sdk.ParameterFind(&act.Parameters, "path")
				if path != nil {
					gitTagArgs["path"] = path.Value
				}
				tagLevel := sdk.ParameterFind(&act.Parameters, "tagLevel")
				if tagLevel != nil {
					gitTagArgs["tagLevel"] = tagLevel.Value
				}
				tagMessage := sdk.ParameterFind(&act.Parameters, "tagMessage")
				if tagMessage != nil {
					gitTagArgs["tagMessage"] = tagMessage.Value
				}
				tagMetadata := sdk.ParameterFind(&act.Parameters, "tagMetadata")
				if tagMetadata != nil && tagMetadata.Value != "" {
					gitTagArgs["tagMetadata"] = tagMetadata.Value
				}
				tagPrerelease := sdk.ParameterFind(&act.Parameters, "tagPrerelease")
				if tagPrerelease != nil && tagPrerelease.Value != "" {
					gitTagArgs["tagPrerelease"] = tagPrerelease.Value
				}
				prefix := sdk.ParameterFind(&act.Parameters, "prefix")
				if prefix != nil && prefix.Value != "" {
					gitTagArgs["prefix"] = prefix.Value
				}
				s["gitTag"] = gitTagArgs
			case sdk.ReleaseAction:
				releaseArgs := map[string]string{}
				artifacts := sdk.ParameterFind(&act.Parameters, "artifacts")
				if artifacts != nil {
					releaseArgs["artifacts"] = artifacts.Value
				}
				releaseNote := sdk.ParameterFind(&act.Parameters, "releaseNote")
				if releaseNote != nil {
					releaseArgs["releaseNote"] = releaseNote.Value
				}
				tag := sdk.ParameterFind(&act.Parameters, "tag")
				if tag != nil {
					releaseArgs["tag"] = tag.Value
				}
				title := sdk.ParameterFind(&act.Parameters, "title")
				if title != nil && title.Value != "" {
					releaseArgs["title"] = title.Value
				}
				s["release"] = releaseArgs
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

			name := act.Name
			if act.Group != nil {
				name = fmt.Sprintf("%s/%s", act.Group.Name, act.Name)
			}

			s[name] = args

		}
		res[i] = s
	}

	return res
}

//AsScript returns the step a sdk.Action
func (s Step) AsScript() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
	}

	bI, ok := s["script"]
	if !ok {
		return nil, false, nil
	}

	bS, isString := bI.(string)
	if !isString {
		asScript, ok := bI.([]interface{})
		asScriptString := make([]string, len(asScript))
		for i := range asScript {
			asScriptString[i], ok = asScript[i].(string)
			if !ok {
				break
			}
		}
		if !ok {
			return nil, true, sdk.NewErrorFrom(sdk.ErrMalformattedStep, "script must be a string or a string array")
		}

		bS = strings.Join(asScriptString, "\n")
	}

	a := sdk.NewStepScript(bS)

	var err error
	a.StepName, err = s.Name()
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

	return &a, true, nil
}

//AsAction returns the step a sdk.Action
func (s Step) AsAction() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
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
		return nil, true, sdk.WithStack(sdk.ErrMalformattedStep)
	}

	a := sdk.NewStepDefault(actionName, argss)

	var err error
	a.StepName, err = s.Name()
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
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
	}

	bI, ok := s["jUnitReport"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)
	if !ok {
		return nil, true, sdk.NewErrorFrom(sdk.ErrMalformattedStep, "jUnitReport must be a string")
	}

	a := sdk.NewStepJUnitReport(bS)

	var err error
	a.StepName, err = s.Name()
	if err != nil {
		return nil, true, err
	}
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
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
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
		return nil, true, sdk.NewErrorWithStack(err, sdk.ErrMalformattedStep)
	}

	a := sdk.NewStepGitClone(argss)

	var err error
	a.StepName, err = s.Name()
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

	return &a, true, nil
}

//AsArtifactUpload returns the step a sdk.Action
func (s Step) AsArtifactUpload() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
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
			return nil, true, sdk.NewErrorWithStack(err, sdk.ErrMalformattedStep)
		}
		a = sdk.NewStepArtifactUpload(argss)
	} else {
		return nil, false, sdk.NewErrorFrom(sdk.ErrMalformattedStep, "unknown type")
	}

	var err error
	a.StepName, err = s.Name()
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

	return &a, true, nil
}

//AsServeStaticFiles returns the step a sdk.Action
func (s Step) AsServeStaticFiles() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
	}

	bI, ok := s["serveStaticFiles"]
	if !ok {
		return nil, false, nil
	}

	if reflect.ValueOf(bI).Kind() != reflect.Map && reflect.ValueOf(bI).Kind() != reflect.String {
		return nil, false, nil
	}

	var argss map[string]string
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WithStack(sdk.ErrMalformattedStep)
	}

	a := sdk.NewStepServeStaticFiles(argss)

	var err error
	a.StepName, err = s.Name()
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

	return &a, true, nil
}

//AsArtifactDownload returns the step a sdk.Action
func (s Step) AsArtifactDownload() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
	}

	bI, ok := s["artifactDownload"]
	if !ok {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.NewErrorWithStack(err, sdk.ErrMalformattedStep)
	}
	a := sdk.NewStepArtifactDownload(argss)

	var err error
	a.StepName, err = s.Name()
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

	return &a, true, nil
}

//AsCheckoutApplication returns the step as a sdk.Action
func (s Step) AsCheckoutApplication() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
	}
	bI, ok := s["checkout"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)
	if !ok {
		return nil, true, sdk.NewErrorFrom(sdk.ErrMalformattedStep, "checkout must be a string")
	}
	a := sdk.NewCheckoutApplication(bS)

	var err error
	a.StepName, err = s.Name()
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

	return &a, true, nil
}

//AsCoverageAction returns the step as a sdk.Action
func (s Step) AsCoverageAction() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
	}
	bI, ok := s["coverage"]
	if !ok {
		return nil, false, nil
	}

	argss := map[string]string{}
	if err := mapstructure.Decode(bI, &argss); err != nil {
		return nil, true, sdk.WithStack(sdk.ErrMalformattedStep)
	}
	a := sdk.NewCoverage(argss)

	var err error
	a.StepName, err = s.Name()
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

	return &a, true, nil
}

//AsDeployApplication returns the step as a sdk.Action
func (s Step) AsDeployApplication() (*sdk.Action, bool, error) {
	if !s.IsValid() {
		return nil, false, sdk.WithStack(sdk.ErrMalformattedStep)
	}
	bI, ok := s["deploy"]
	if !ok {
		return nil, false, nil
	}

	bS, ok := bI.(string)
	if !ok {
		return nil, true, sdk.NewErrorFrom(sdk.ErrMalformattedStep, "deploy must be a string")
	}
	a := sdk.NewDeployApplication(bS)

	var err error
	a.StepName, err = s.Name()
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
		return false, sdk.NewErrorFrom(sdk.ErrWrongRequest, "%s attribute must be true|false", flag)
	}
	return bS, nil
}

// Name returns true the step name if exist
func (s Step) Name() (string, error) {
	if stepAttr, ok := s["name"]; ok {
		if stepName, okName := stepAttr.(string); okName {
			return stepName, nil
		}
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "name must be a string")
	}
	return "", nil
}

// Action returns an sdk.Action
func (ea *Action) Action() (sdk.Action, error) {
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
