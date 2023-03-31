package workflowv3

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ovh/cds/sdk/exportentities"
)

type Step struct {
	exportentities.StepCustom `json:"-" yaml:",inline"`
	Script                    *StepScript                          `json:"script,omitempty" yaml:"script,omitempty"`
	PushBuildInfo             *exportentities.StepPushBuildInfo    `json:"pushBuildInfo,omitempty" yaml:"pushBuildInfo,omitempty"`
	Coverage                  *exportentities.StepCoverage         `json:"coverage,omitempty" yaml:"coverage,omitempty"`
	ArtifactDownload          *exportentities.StepArtifactDownload `json:"artifactDownload,omitempty" yaml:"artifactDownload,omitempty"`
	ArtifactUpload            *exportentities.StepArtifactUpload   `json:"artifactUpload,omitempty" yaml:"artifactUpload,omitempty"`
	GitClone                  *exportentities.StepGitClone         `json:"gitClone,omitempty" yaml:"gitClone,omitempty"`
	GitTag                    *exportentities.StepGitTag           `json:"gitTag,omitempty" yaml:"gitTag,omitempty"`
	ReleaseVCS                *exportentities.StepReleaseVCS       `json:"releaseVCS,omitempty" yaml:"releaseVCS,omitempty"`
	Release                   *exportentities.StepRelease          `json:"release,omitempty" yaml:"release,omitempty"`
	JUnitReport               *exportentities.StepJUnitReport      `json:"jUnitReport,omitempty" yaml:"jUnitReport,omitempty"`
	Checkout                  *exportentities.StepCheckout         `json:"checkout,omitempty" yaml:"checkout,omitempty"`
	InstallKey                *exportentities.StepInstallKey       `json:"installKey,omitempty" yaml:"installKey,omitempty"`
	Deploy                    *exportentities.StepDeploy           `json:"deploy,omitempty" yaml:"deploy,omitempty"`
	Promote                   *exportentities.StepPromote          `json:"promote,omitempty" yaml:"promote,omitempty"`
	AsCodeAction              *exportentities.StepAscodeAction     `json:"AsCodeAction,omitempty" yaml:"AsCodeAction,omitempty"`
}

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
	if err := json.Unmarshal(b, &m); err != nil {
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

func (s *Step) UnmarshalJSON(data []byte) error {
	type StepAlias Step // prevent recursion
	var sa StepAlias
	if err := json.Unmarshal(data, &sa); err != nil {
		return err
	}
	*s = Step(sa)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
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
			var sp exportentities.StepParameters
			if err := json.Unmarshal(v, &sp); err != nil {
				return err
			}
			if s.StepCustom == nil {
				s.StepCustom = make(exportentities.StepCustom)
			}
			s.StepCustom[k] = sp
		}
	}

	return nil
}

type StepScript interface{}

func (s Step) Validate(w Workflow) (ExternalDependencies, error) {
	var extDep ExternalDependencies

	// Check action type
	var actionTypes []string
	if s.Script != nil {
		actionTypes = append(actionTypes, "script")
	}
	if s.PushBuildInfo != nil {
		actionTypes = append(actionTypes, "pushBuildInfo")
	}
	if s.Deploy != nil {
		actionTypes = append(actionTypes, "deploy")
	}
	if s.ArtifactDownload != nil {
		actionTypes = append(actionTypes, "artifactDownload")
	}
	if s.ArtifactUpload != nil {
		actionTypes = append(actionTypes, "artifactUpload")
	}
	if s.JUnitReport != nil {
		actionTypes = append(actionTypes, "jUnitReport")
	}
	if s.GitClone != nil {
		actionTypes = append(actionTypes, "gitClone")
	}
	if s.GitTag != nil {
		actionTypes = append(actionTypes, "gitTag")
	}
	if s.ReleaseVCS != nil {
		actionTypes = append(actionTypes, "releaseVCS")
	}
	if s.Release != nil {
		actionTypes = append(actionTypes, "release")
	}
	if s.Checkout != nil {
		actionTypes = append(actionTypes, "checkout")
	}
	if s.InstallKey != nil {
		actionTypes = append(actionTypes, "installKey")
	}
	if s.Coverage != nil {
		actionTypes = append(actionTypes, "coverage")
	}
	if s.Promote != nil {
		actionTypes = append(actionTypes, "promote")
	}
	for aName := range s.StepCustom {
		actionTypes = append(actionTypes, aName)
	}
	if len(actionTypes) == 0 {
		return extDep, fmt.Errorf("cannot read action name")
	}
	if len(actionTypes) > 1 {
		return extDep, fmt.Errorf("multiple action defined for the same step %q", actionTypes)
	}

	// Check that custom action exists
	for aName := range s.StepCustom {
		targetAction := strings.TrimPrefix(aName, "@")
		isExternal := targetAction != aName
		if isExternal {
			extDep.Actions = append(extDep.Actions, targetAction)
		} else {
			if !w.Actions.ExistAction(targetAction) {
				return extDep, fmt.Errorf("unknown action %q", targetAction)
			}
		}
	}

	// For deploy action check if deployments exists
	if s.Deploy != nil {
		targetDeployment := strings.TrimPrefix(string(*s.Deploy), "@")
		isExternal := targetDeployment != string(*s.Deploy)
		if isExternal {
			extDep.Deployments = append(extDep.Deployments, targetDeployment)
		} else {
			if !w.Deployments.ExistDeployment(targetDeployment) {
				return extDep, fmt.Errorf("unknown deployment %q", targetDeployment)
			}
		}
	}

	return extDep, nil
}
