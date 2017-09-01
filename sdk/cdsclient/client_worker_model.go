package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// WorkerModelsEnabled retrieves all worker models enabled and available to user
func (c *client) WorkerModelsEnabled() ([]sdk.Model, error) {
	return c.workerModels(false)
}

// WorkerModels retrieves all worker models available to user (enabled or not)
func (c *client) WorkerModels() ([]sdk.Model, error) {
	return c.workerModels(true)
}

func (c *client) workerModels(withDisabled bool) ([]sdk.Model, error) {
	var uri string
	if withDisabled {
		uri = fmt.Sprintf("/worker/model")
	} else {
		uri = fmt.Sprintf("/worker/model/enabled")
	}

	var models []sdk.Model
	if _, errr := c.GetJSON(uri, &models); errr != nil {
		return nil, errr
	}
	return models, nil
}

func (c *client) WorkerModelSpawnError(id int64, info string) error {
	data := sdk.SpawnErrorForm{Error: info}
	code, err := c.PutJSON(fmt.Sprintf("/worker/model/error/%d", id), &data, nil)
	if code > 300 && err == nil {
		return fmt.Errorf("WorkerModelSpawnError> HTTP %d", code)
	} else if err != nil {
		return sdk.WrapError(err, "WorkerModelSpawnError> Error")
	}
	return nil
}
