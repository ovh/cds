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
