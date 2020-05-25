package hatchery

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

// ModelInterpolateSecrets interpolates secrets that can exists inside given model.
func ModelInterpolateSecrets(hWithModels InterfaceWithModels, model *sdk.Model) error {
	// For now only docker registry password can be interpolate
	if model.Type != sdk.Docker || !model.ModelDocker.Private {
		return nil
	}

	modelSecrets, err := hWithModels.WorkerModelSecretList(*model)
	if err != nil {
		return sdk.WrapError(err, "cannot load secrets for model %s", model.Path())
	}

	model.ModelDocker.Password, err = interpolate.Do(model.ModelDocker.Password, modelSecrets.ToMap())
	if err != nil {
		return sdk.WrapError(err, "cannot interpolate registry password for model %s", model.Path())
	}

	return nil
}
