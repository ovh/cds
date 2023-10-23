package exportentities

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ovh/cds/sdk"

	"github.com/fsamin/go-dump"
)

func NewStep(act sdk.Action) Step {
	s := Step{
		Name: act.StepName,
	}

	if !act.Enabled {
		s.Enabled = &sdk.False
	}
	if act.Optional {
		s.Optional = &sdk.True
	}
	if act.AlwaysExecuted {
		s.AlwaysExecuted = &sdk.True
	}

	switch act.Type {
	case sdk.BuiltinAction:
		switch act.Name {
		case sdk.ScriptAction:
			script := sdk.ParameterFind(act.Parameters, "script")
			if script != nil {
				s.Script = strings.SplitN(script.Value, "\n", -1)
			}
		case sdk.CoverageAction:
			s.Coverage = &StepCoverage{}
			path := sdk.ParameterFind(act.Parameters, "path")
			if path != nil {
				s.Coverage.Path = path.Value
			}
		case sdk.ArtifactDownload:
			s.ArtifactDownload = &StepArtifactDownload{}
			path := sdk.ParameterFind(act.Parameters, "path")
			if path != nil {
				s.ArtifactDownload.Path = path.Value
			}
			tag := sdk.ParameterFind(act.Parameters, "tag")
			if tag != nil {
				s.ArtifactDownload.Tag = tag.Value
			}
			pattern := sdk.ParameterFind(act.Parameters, "pattern")
			if pattern != nil {
				s.ArtifactDownload.Pattern = pattern.Value
			}
		case sdk.ArtifactUpload:
			s.ArtifactUpload = &StepArtifactUpload{}
			path := sdk.ParameterFind(act.Parameters, "path")
			if path != nil {
				s.ArtifactUpload.Path = path.Value
			}
			tag := sdk.ParameterFind(act.Parameters, "tag")
			if tag != nil {
				s.ArtifactUpload.Tag = tag.Value
			}
			destination := sdk.ParameterFind(act.Parameters, "destination")
			if destination != nil {
				s.ArtifactUpload.Destination = destination.Value
			}
		case sdk.GitCloneAction:
			s.GitClone = &StepGitClone{}
			branch := sdk.ParameterFind(act.Parameters, "branch")
			if branch != nil {
				s.GitClone.Branch = branch.Value
			}
			commit := sdk.ParameterFind(act.Parameters, "commit")
			if commit != nil {
				s.GitClone.Commit = commit.Value
			}
			directory := sdk.ParameterFind(act.Parameters, "directory")
			if directory != nil {
				s.GitClone.Directory = directory.Value
			}
			password := sdk.ParameterFind(act.Parameters, "password")
			if password != nil {
				s.GitClone.Password = password.Value
			}
			privateKey := sdk.ParameterFind(act.Parameters, "privateKey")
			if privateKey != nil {
				s.GitClone.PrivateKey = privateKey.Value
			}
			url := sdk.ParameterFind(act.Parameters, "url")
			if url != nil {
				s.GitClone.URL = url.Value
			}
			user := sdk.ParameterFind(act.Parameters, "user")
			if user != nil {
				s.GitClone.User = user.Value
			}
			depth := sdk.ParameterFind(act.Parameters, "depth")
			if depth != nil && depth.Value != "50" {
				s.GitClone.Depth = depth.Value
			}
			submodules := sdk.ParameterFind(act.Parameters, "submodules")
			if submodules != nil && submodules.Value != "true" {
				s.GitClone.SubModules = submodules.Value
			}
			tag := sdk.ParameterFind(act.Parameters, "tag")
			if tag != nil && tag.Value != sdk.DefaultGitCloneParameterTagValue {
				s.GitClone.Tag = &tag.Value
			}
		case sdk.GitTagAction:
			s.GitTag = &StepGitTag{}
			path := sdk.ParameterFind(act.Parameters, "path")
			if path != nil {
				s.GitTag.Path = path.Value
			}
			tagLevel := sdk.ParameterFind(act.Parameters, "tagLevel")
			if tagLevel != nil {
				s.GitTag.TagLevel = tagLevel.Value
			}
			tagMessage := sdk.ParameterFind(act.Parameters, "tagMessage")
			if tagMessage != nil {
				s.GitTag.TagMessage = tagMessage.Value
			}
			tagMetadata := sdk.ParameterFind(act.Parameters, "tagMetadata")
			if tagMetadata != nil {
				s.GitTag.TagMetadata = tagMetadata.Value
			}
			tagPrerelease := sdk.ParameterFind(act.Parameters, "tagPrerelease")
			if tagPrerelease != nil {
				s.GitTag.TagPrerelease = tagPrerelease.Value
			}
			prefix := sdk.ParameterFind(act.Parameters, "prefix")
			if prefix != nil {
				s.GitTag.Prefix = prefix.Value
			}
		case sdk.PromoteAction:
			s.Promote = &StepPromote{}
			artifacts := sdk.ParameterFind(act.Parameters, "artifacts")
			if artifacts != nil {
				s.Promote.Artifacts = artifacts.Value
			}
			srcMaturity := sdk.ParameterFind(act.Parameters, "srcMaturity")
			if srcMaturity != nil {
				s.Promote.SrcMaturity = srcMaturity.Value
			}
			destMaturity := sdk.ParameterFind(act.Parameters, "destMaturity")
			if destMaturity != nil {
				s.Promote.DestMaturity = destMaturity.Value
			}
			setProperties := sdk.ParameterFind(act.Parameters, "setProperties")
			if setProperties != nil {
				s.Promote.SetProperties = setProperties.Value
			}
		case sdk.ReleaseAction:
			s.Release = &StepRelease{}
			artifacts := sdk.ParameterFind(act.Parameters, "artifacts")
			if artifacts != nil {
				s.Release.Artifacts = artifacts.Value
			}
			releaseNote := sdk.ParameterFind(act.Parameters, "releaseNote")
			if releaseNote != nil {
				s.Release.ReleaseNote = releaseNote.Value
			}
			srcMaturity := sdk.ParameterFind(act.Parameters, "srcMaturity")
			if srcMaturity != nil {
				s.Release.SrcMaturity = srcMaturity.Value
			}
			destMaturity := sdk.ParameterFind(act.Parameters, "destMaturity")
			if destMaturity != nil {
				s.Release.DestMaturity = destMaturity.Value
			}
			setProperties := sdk.ParameterFind(act.Parameters, "setProperties")
			if setProperties != nil {
				s.Release.SetProperties = setProperties.Value
			}
		case sdk.ReleaseVCSAction:
			s.ReleaseVCS = &StepReleaseVCS{}
			artifacts := sdk.ParameterFind(act.Parameters, "artifacts")
			if artifacts != nil {
				s.ReleaseVCS.Artifacts = artifacts.Value
			}
			releaseNote := sdk.ParameterFind(act.Parameters, "releaseNote")
			if releaseNote != nil {
				s.ReleaseVCS.ReleaseNote = releaseNote.Value
			}
			tag := sdk.ParameterFind(act.Parameters, "tag")
			if tag != nil {
				s.ReleaseVCS.Tag = tag.Value
			}
			title := sdk.ParameterFind(act.Parameters, "title")
			if title != nil {
				s.ReleaseVCS.Title = title.Value
			}
		case sdk.JUnitAction:
			var step StepJUnitReport
			path := sdk.ParameterFind(act.Parameters, "path")
			if path != nil {
				step = StepJUnitReport(path.Value)
			}
			s.JUnitReport = &step
		case sdk.CheckoutApplicationAction:
			var step StepCheckout
			directory := sdk.ParameterFind(act.Parameters, "directory")
			if directory != nil {
				step = StepCheckout(directory.Value)
			}
			s.Checkout = &step
		case sdk.InstallKeyAction:
			key := sdk.ParameterFind(act.Parameters, "key")
			file := sdk.ParameterFind(act.Parameters, "file")
			switch {
			case key != nil && file != nil && file.Value != "":
				step := StepInstallKey(map[string]string{
					"name": key.Value,
					"file": file.Value,
				})
				s.InstallKey = &step

			case key != nil:
				step := StepInstallKey(string(key.Value))
				s.InstallKey = &step
			}
		case sdk.PushBuildInfo:
			step := StepPushBuildInfo("{{.cds.workflow}}")
			s.PushBuildInfo = &step
		case sdk.DeployApplicationAction:
			step := StepDeploy("{{.cds.application}}")
			s.Deploy = &step
		}
	case sdk.AsCodeAction:
		step := make(StepAscodeAction)
		for _, v := range act.Parameters {
			step[v.Name] = v.Value
		}
		s.AsCodeAction = &step
	default:
		args := make(StepParameters)
		for _, p := range act.Parameters {
			if p.Value != "" {
				args[p.Name] = p.Value
			}
		}

		name := act.Name
		// Do not export "shared.infra" group name
		if act.Group != nil && act.Group.Name != sdk.SharedInfraGroupName {
			name = fmt.Sprintf("%s/%s", act.Group.Name, act.Name)
		}

		s.StepCustom = StepCustom{
			name: args,
		}
	}
	return s
}

// StepParameters represents exported custom step parameters.
type StepParameters map[string]string

// StepCustom represents exported custom step.
type StepCustom map[string]StepParameters

type StepPushBuildInfo string

// StepCoverage represents exported coverage step.
type StepCoverage struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// StepArtifactDownload represents exported artifact download step.
type StepArtifactDownload struct {
	Path    string `json:"path,omitempty" yaml:"path,omitempty" jsonschema:"required"`
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Tag     string `json:"tag,omitempty" yaml:"tag,omitempty" jsonschema:"required"`
}

// StepArtifactUpload represents exported artifact upload step.
type StepArtifactUpload struct {
	Destination string `json:"destination,omitempty" yaml:"destination,omitempty"`
	Path        string `json:"path,omitempty" yaml:"path,omitempty" jsonschema:"required"`
	Tag         string `json:"tag,omitempty" yaml:"tag,omitempty" jsonschema:"required"`
}

// StepGitClone represents exported git clone step.
type StepGitClone struct {
	Branch     string  `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit     string  `json:"commit,omitempty" yaml:"commit,omitempty"`
	Depth      string  `json:"depth,omitempty" yaml:"depth,omitempty"`
	Directory  string  `json:"directory,omitempty" yaml:"directory,omitempty"`
	Password   string  `json:"password,omitempty" yaml:"password,omitempty"`
	PrivateKey string  `json:"privateKey,omitempty" yaml:"privateKey,omitempty"`
	SubModules string  `json:"submodules,omitempty" yaml:"submodules,omitempty"`
	Tag        *string `json:"tag,omitempty" yaml:"tag,omitempty"`
	URL        string  `json:"url,omitempty" yaml:"url,omitempty" jsonschema:"required"`
	User       string  `json:"user,omitempty" yaml:"user,omitempty"`
}

// StepPromote represents exported promote step.
type StepPromote struct {
	Artifacts     string `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	SrcMaturity   string `json:"srcMaturity,omitempty" yaml:"srcMaturity,omitempty"`
	DestMaturity  string `json:"destMaturity,omitempty" yaml:"destMaturity,omitempty"`
	SetProperties string `json:"setProperties,omitempty" yaml:"setProperties,omitempty"`
}

// StepRelease represents exported release step.
type StepRelease struct {
	Artifacts     string `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	ReleaseNote   string `json:"releaseNote,omitempty" yaml:"releaseNote,omitempty"`
	SrcMaturity   string `json:"srcMaturity,omitempty" yaml:"srcMaturity,omitempty"`
	DestMaturity  string `json:"destMaturity,omitempty" yaml:"destMaturity,omitempty"`
	SetProperties string `json:"setProperties,omitempty" yaml:"setProperties,omitempty"`
}

// StepReleaseVCS represents exported release step.
type StepReleaseVCS struct {
	Artifacts   string `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	ReleaseNote string `json:"releaseNote,omitempty" yaml:"releaseNote,omitempty"`
	Tag         string `json:"tag,omitempty" yaml:"tag,omitempty" jsonschema:"required"`
	Title       string `json:"title,omitempty" yaml:"title,omitempty" jsonschema:"required"`
}

// StepGitTag represents exported git tag step.
type StepGitTag struct {
	Path          string `json:"path,omitempty" yaml:"path,omitempty"`
	Prefix        string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	TagLevel      string `json:"tagLevel,omitempty" yaml:"tagLevel,omitempty" jsonschema:"required"`
	TagMessage    string `json:"tagMessage,omitempty" yaml:"tagMessage,omitempty"`
	TagMetadata   string `json:"tagMetadata,omitempty" yaml:"tagMetadata,omitempty"`
	TagPrerelease string `json:"tagPrerelease,omitempty" yaml:"tagPrerelease,omitempty"`
}

// StepJUnitReport represents exported junit report step.
type StepJUnitReport string

// StepCheckout represents exported checkout step.
type StepCheckout string

// StepInstallKey represents exported installKey step.
type StepInstallKey interface{}

// StepDeploy represents exported push build info step.
type StepDeploy string

type StepAscodeAction map[string]string

// Step represents exported step used in a job.
type Step struct {
	// common step data
	Name           string `json:"name,omitempty" yaml:"name,omitempty" jsonschema_description:"The name for this step."`
	Enabled        *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Optional       *bool  `json:"optional,omitempty" yaml:"optional,omitempty"`
	AlwaysExecuted *bool  `json:"always_executed,omitempty" yaml:"always_executed,omitempty"`
	// step specific data, only one option should be set
	StepCustom       `json:"-" yaml:",inline"`
	Script           interface{}           `json:"script,omitempty" yaml:"script,omitempty" jsonschema:"oneof_type=string;array,oneof_required=actionScript" jsonschema_description:"Script.\nhttps://ovh.github.io/cds/docs/actions/builtin-script"`
	PushBuildInfo    *StepPushBuildInfo    `json:"pushBuildInfo,omitempty" yaml:"pushBuildInfo,omitempty" jsonschema:"oneof_required=actionPushBuildInfo" jsonschema_description:"Push build info.\nhttps://ovh.github.io/cds/docs/actions/builtin-push-build-info"`
	Coverage         *StepCoverage         `json:"coverage,omitempty" yaml:"coverage,omitempty" jsonschema:"oneof_required=actionCoverage" jsonschema_description:"Parse coverage report.\nhttps://ovh.github.io/cds/docs/actions/builtin-coverage"`
	ArtifactDownload *StepArtifactDownload `json:"artifactDownload,omitempty" yaml:"artifactDownload,omitempty" jsonschema:"oneof_required=actionArtifactDownload" jsonschema_description:"Download artifacts in workspace.\nhttps://ovh.github.io/cds/docs/actions/builtin-artifact-download"`
	ArtifactUpload   *StepArtifactUpload   `json:"artifactUpload,omitempty" yaml:"artifactUpload,omitempty" jsonschema:"oneof_required=actionArtifactUpload" jsonschema_description:"Upload artifacts from workspace.\nhttps://ovh.github.io/cds/docs/actions/builtin-artifact-upload"`
	GitClone         *StepGitClone         `json:"gitClone,omitempty" yaml:"gitClone,omitempty" jsonschema:"oneof_required=actionGitClone" jsonschema_description:"Clone a git repository.\nhttps://ovh.github.io/cds/docs/actions/builtin-gitclone"`
	GitTag           *StepGitTag           `json:"gitTag,omitempty" yaml:"gitTag,omitempty" jsonschema:"oneof_required=actionGitTag" jsonschema_description:"Create a git tag.\nhttps://ovh.github.io/cds/docs/actions/builtin-gittag"`
	ReleaseVCS       *StepReleaseVCS       `json:"releaseVCS,omitempty" yaml:"releaseVCS,omitempty" jsonschema:"oneof_required=actionReleaseVCS" jsonschema_description:"Release an application.\nhttps://ovh.github.io/cds/docs/actions/builtin-releasevcs"`
	Release          *StepRelease          `json:"release,omitempty" yaml:"release,omitempty" jsonschema:"oneof_required=actionRelease" jsonschema_description:"Release an application.\nhttps://ovh.github.io/cds/docs/actions/builtin-release"`
	Promote          *StepPromote          `json:"promote,omitempty" yaml:"promote,omitempty" jsonschema:"oneof_required=actionPromote" jsonschema_description:"Promote artifacts.\nhttps://ovh.github.io/cds/docs/actions/builtin-promote"`
	JUnitReport      *StepJUnitReport      `json:"jUnitReport,omitempty" yaml:"jUnitReport,omitempty" jsonschema:"oneof_required=actionJUNit" jsonschema_description:"Parse JUnit report.\nhttps://ovh.github.io/cds/docs/actions/builtin-junit"`
	Checkout         *StepCheckout         `json:"checkout,omitempty" yaml:"checkout,omitempty" jsonschema:"oneof_required=actionCheckout" jsonschema_description:"Checkout repository for an application.\nhttps://ovh.github.io/cds/docs/actions/builtin-checkoutapplication"`
	InstallKey       *StepInstallKey       `json:"installKey,omitempty" yaml:"installKey,omitempty" jsonschema:"oneof_required=actionInstallKey" jsonschema_description:"Install a key (GPG, SSH) in your current workspace.\nhttps://ovh.github.io/cds/docs/actions/builtin-installkey"`
	Deploy           *StepDeploy           `json:"deploy,omitempty" yaml:"deploy,omitempty" jsonschema:"oneof_required=actionDeploy" jsonschema_description:"Deploy an application.\nhttps://ovh.github.io/cds/docs/actions/builtin-deployapplication"`
	AsCodeAction     *StepAscodeAction     `json:"asCodeAction,omitempty" yaml:"asCodeAction,omitempty" jsonschema:"oneof_required=asCodeAction" jsonschema_description:"ascode action"`
}

// MarshalJSON custom marshal json impl to inline custom step.
func (s Step) MarshalJSON() ([]byte, error) {
	type StepAlias Step // prevent recursion
	sa := StepAlias(s)

	if sa.StepCustom == nil {
		return json.Marshal(sa)
	}

	b, err := json.Marshal(sa)
	if err != nil {
		return nil, err
	}

	var m map[string]json.RawMessage
	if err := sdk.JSONUnmarshal(b, &m); err != nil {
		return nil, err
	}

	for k, v := range sa.StepCustom {
		// do not override builtin action key
		if _, ok := m[k]; ok {
			continue
		}

		b, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = b
	}

	return json.Marshal(m)
}

// UnmarshalJSON custom unmarshal json impl to get custom step data.
func (s *Step) UnmarshalJSON(data []byte) error {
	type StepAlias Step // prevent recursion
	var sa StepAlias
	if err := sdk.JSONUnmarshal(data, &sa); err != nil {
		return err
	}
	*s = Step(sa)

	var m map[string]json.RawMessage
	if err := sdk.JSONUnmarshal(data, &m); err != nil {
		return err
	}

	jsonFields := make(map[string]struct{})

	typ := reflect.TypeOf(s).Elem()
	countFields := typ.NumField()
	for i := 0; i < countFields; i++ {
		jsonName := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if jsonName != "" {
			jsonFields[jsonName] = struct{}{}
		}
	}

	for k, v := range m {
		if _, ok := jsonFields[k]; !ok {
			var sp StepParameters
			if err := sdk.JSONUnmarshal(v, &sp); err != nil {
				return err
			}
			if s.StepCustom == nil {
				s.StepCustom = make(StepCustom)
			}
			s.StepCustom[k] = sp
		}
	}

	return nil
}

// IsValid returns true is the step is valid
func (s Step) IsValid() bool {
	// only one option should not be nil or the custom step length should equals 1

	var count int
	if s.isArtifactDownload() {
		count++
	}
	if s.isArtifactUpload() {
		count++
	}
	if s.isJUnitReport() {
		count++
	}
	if s.isGitClone() {
		count++
	}
	if s.isGitTag() {
		count++
	}
	if s.isReleaseVCS() {
		count++
	}
	if s.isRelease() {
		count++
	}
	if s.isPromote() {
		count++
	}
	if s.isCheckout() {
		count++
	}
	if s.isDeploy() {
		count++
	}
	if s.isCoverage() {
		count++
	}
	if s.isScript() {
		count++
	}
	if s.isPushBuildInfo() {
		count++
	}
	if s.isAscodeAction() {
		count++
	}
	count += len(s.StepCustom)

	return count == 1
}

func (s Step) toAction() (*sdk.Action, error) {
	if !s.IsValid() {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "malformatted step")
	}

	var a sdk.Action
	var err error
	if s.isArtifactDownload() {
		a, err = s.asArtifactDownload()
	} else if s.isArtifactUpload() {
		a, err = s.asArtifactUpload()
	} else if s.isJUnitReport() {
		a = s.asJUnitReport()
	} else if s.isGitClone() {
		a, err = s.asGitClone()
	} else if s.isGitTag() {
		a, err = s.asGitTag()
	} else if s.isReleaseVCS() {
		a, err = s.asReleaseVCS()
	} else if s.isPromote() {
		a, err = s.asPromote()
	} else if s.isRelease() {
		a, err = s.asRelease()
	} else if s.isCheckout() {
		a = s.asCheckoutApplication()
	} else if s.isDeploy() {
		a = s.asDeployApplication()
	} else if s.isCoverage() {
		a, err = s.asCoverage()
	} else if s.isScript() {
		a, err = s.asScript()
	} else if s.isPushBuildInfo() {
		a = s.asPushBuildInfo()
	} else if s.isAscodeAction() {
		a = s.asAscodeAction()
	} else {
		a = s.asAction()
	}
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot convert step to action step"))
	}

	a.StepName = s.Name
	a.Enabled = s.Enabled == nil || *s.Enabled == sdk.True // enabled is true by default
	a.Optional = s.Optional != nil && *s.Optional == sdk.True
	a.AlwaysExecuted = s.AlwaysExecuted != nil && *s.AlwaysExecuted == sdk.True

	return &a, nil
}

func (s Step) isScript() bool { return s.Script != nil }

func (s Step) asScript() (sdk.Action, error) {
	var a sdk.Action
	// TODO use typed value for script
	// val := strings.Join(*s.Script, "\n")

	var val string
	if script, ok := s.Script.([]interface{}); ok {
		lines := make([]string, len(script))
		for i := range script {
			if line, okString := script[i].(string); okString {
				lines[i] = line
			}
		}
		val = strings.Join(lines, "\n")
	} else if script, ok := s.Script.([]string); ok {
		val = strings.Join(script, "\n")
	} else if script, ok := s.Script.(string); ok {
		val = script
	} else {
		return a, sdk.NewErrorFrom(sdk.ErrMalformattedStep, "invalid given data for script action")
	}

	a = sdk.Action{
		Name: sdk.ScriptAction,
		Type: sdk.BuiltinAction,
		Parameters: []sdk.Parameter{
			{
				Name:  "script",
				Value: val,
				Type:  sdk.TextParameter,
			},
		},
	}

	return a, nil
}

func (s Step) isArtifactDownload() bool { return s.ArtifactDownload != nil }

func (s Step) asArtifactDownload() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.ArtifactDownload)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.ArtifactDownload,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) isCheckout() bool { return s.Checkout != nil }

func (s Step) asCheckoutApplication() sdk.Action {
	return sdk.Action{
		Name: sdk.CheckoutApplicationAction,
		Type: sdk.BuiltinAction,
		Parameters: []sdk.Parameter{
			{
				Name:  "directory",
				Value: string(*s.Checkout),
				Type:  sdk.StringParameter,
			},
		},
	}
}

func (s Step) isPushBuildInfo() bool { return s.PushBuildInfo != nil }

func (s Step) isAscodeAction() bool { return s.AsCodeAction != nil }

func (s Step) asPushBuildInfo() sdk.Action {
	return sdk.Action{
		Name: sdk.PushBuildInfo,
		Type: sdk.BuiltinAction,
	}
}

func (s Step) asAscodeAction() sdk.Action {
	return sdk.Action{
		Name:       sdk.AsCodeAction,
		Type:       sdk.AsCodeAction,
		StepName:   s.Name,
		Parameters: sdk.ParametersFromMap(*s.AsCodeAction),
	}
}

func (s Step) isCoverage() bool { return s.Coverage != nil }

func (s Step) asCoverage() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.Coverage)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.CoverageAction,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) isDeploy() bool { return s.Deploy != nil }

func (s Step) asDeployApplication() sdk.Action {
	return sdk.Action{
		Name: sdk.DeployApplicationAction,
		Type: sdk.BuiltinAction,
	}
}

func (s Step) isJUnitReport() bool { return s.JUnitReport != nil }

func (s Step) asJUnitReport() sdk.Action {
	return sdk.Action{
		Name: sdk.JUnitAction,
		Type: sdk.BuiltinAction,
		Parameters: []sdk.Parameter{
			{
				Name:  "path",
				Value: string(*s.JUnitReport),
				Type:  sdk.StringParameter,
			},
		},
	}
}

func (s Step) isGitClone() bool { return s.GitClone != nil }

func (s Step) asGitClone() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.GitClone)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.GitCloneAction,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) isGitTag() bool { return s.GitTag != nil }

func (s Step) asGitTag() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.GitTag)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.GitTagAction,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) isPromote() bool { return s.Promote != nil }

func (s Step) asPromote() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.Promote)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.PromoteAction,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) isRelease() bool { return s.Release != nil }

func (s Step) asRelease() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.Release)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.ReleaseAction,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) isReleaseVCS() bool { return s.ReleaseVCS != nil }

func (s Step) asReleaseVCS() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.ReleaseVCS)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.ReleaseVCSAction,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) isArtifactUpload() bool { return s.ArtifactUpload != nil }

func (s Step) asArtifactUpload() (sdk.Action, error) {
	var a sdk.Action
	m, err := stepToMap(s.ArtifactUpload)
	if err != nil {
		return a, err
	}
	a = sdk.Action{
		Name:       sdk.ArtifactUpload,
		Type:       sdk.BuiltinAction,
		Parameters: sdk.ParametersFromMap(m),
	}
	return a, nil
}

func (s Step) asAction() sdk.Action {
	var name string
	for k := range s.StepCustom {
		name = k
		break
	}

	a := sdk.Action{
		Name:       name,
		Parameters: []sdk.Parameter{},
	}

	splitted := strings.Split(name, "/")
	if len(splitted) == 2 {
		a.Name = splitted[1]
		a.Group = &sdk.Group{Name: splitted[0]}
	}

	a.Parameters = sdk.ParametersFromMap(s.StepCustom[name])

	return a
}

func stepToMap(i interface{}) (map[string]string, error) {
	buf, err := json.Marshal(i)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	var m map[string]string
	if err := sdk.JSONUnmarshal(buf, &m); err != nil {
		return nil, sdk.WithStack(err)
	}
	return m, nil
}

func (s Step) String() string {
	buf := new(bytes.Buffer)
	dp := dump.NewEncoder(buf)
	_ = dp.Fdump(s)
	return buf.String()
}
