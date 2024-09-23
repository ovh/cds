package hatchery

import (
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

// ModelInterpolateSecrets interpolates secrets that can exists inside given model.
func ModelInterpolateSecrets(hWithModels InterfaceWithModels, model *sdk.Model) error {
	// For now only docker registry password can be interpolate
	if model.Type != sdk.VSphere && (model.Type != sdk.Docker || !model.ModelDocker.Private) {
		return nil
	}

	// Test ascode model : can't be interpolate
	// group is Name: projKey + "/" + vcsName + "/" + repoName on model v2
	groupName := strings.Split(model.Group.Name, "/")
	if model.Type == sdk.VSphere && model.Type != sdk.VSphere && len(groupName) >= 3 {
		return nil
	}

	modelSecrets, err := hWithModels.WorkerModelSecretList(*model)
	if err != nil {
		return sdk.WrapError(err, "cannot load secrets for model %s", model.Path())
	}

	switch {
	case model.Type == sdk.Docker && model.ModelDocker.Private:
		model.ModelDocker.Password, err = interpolate.Do(model.ModelDocker.Password, modelSecrets.ToMap())
		if err != nil {
			return sdk.WrapError(err, "cannot interpolate registry password for model %s", model.Path())
		}
	case model.Type == sdk.VSphere:
		model.ModelVirtualMachine.Password, err = interpolate.Do(model.ModelVirtualMachine.Password, modelSecrets.ToMap())
		if err != nil {
			return sdk.WrapError(err, "cannot interpolate vm password for model %s", model.Path())
		}
	}

	return nil
}
