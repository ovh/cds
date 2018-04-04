package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// WorkerModelBook books a worker model for register, used by hatcheries
func (c *client) WorkerModelBook(id int64) error {
	code, err := c.PutJSON(fmt.Sprintf("/worker/model/book/%d", id), nil, nil)
	if code > 300 && err == nil {
		return fmt.Errorf("WorkerModelBook> HTTP %d", code)
	} else if err != nil {
		return sdk.WrapError(err, "WorkerModelBook> Error")
	}
	return nil
}

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

// WorkerModelAdd create a new worker model available
func (c *client) WorkerModelAdd(name string, modelType string, image string, groupID int64) (sdk.Model, error) {
	uri := fmt.Sprintf("/worker/model")
	model := sdk.Model{
		Name:          name,
		Type:          modelType,
		Image:         image,
		GroupID:       groupID,
		Communication: "http",
	}

	modelCreated := sdk.Model{}
	code, err := c.PostJSON(uri, model, &modelCreated)
	if err != nil {
		return modelCreated, err
	}
	if code >= 300 {
		return modelCreated, fmt.Errorf("WorkerModelAdd> HTTP %d", code)
	}

	return modelCreated, nil
}

func (c *client) WorkerModel(name string) (sdk.Model, error) {
	uri := fmt.Sprintf("/worker/model?name=" + name)
	var model sdk.Model
	_, err := c.GetJSON(uri, &model)
	return model, err
}

func (c *client) WorkerModelDelete(name string) error {
	wm, err := c.WorkerModel(name)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("/worker/model/%d", wm.ID)
	_, errDelete := c.DeleteJSON(uri, nil)
	return errDelete
}
