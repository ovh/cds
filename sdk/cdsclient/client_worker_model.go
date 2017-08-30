package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// WorkerModelConfigParam configure the update operation
type WorkerModelConfigParam func(m *sdk.Model)

// WorkerModelOpts list all the options for worker model
var WorkerModelOpts = struct {
	WithoutRegistrationNeed func() WorkerModelConfigParam
}{
	WithoutRegistrationNeed: withoutRegistrationNeed,
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

func (c *client) WorkerModelUpdate(id int64, name string, t string, value string, opts ...WorkerModelConfigParam) error {
	data := sdk.Model{ID: id, Name: name, Type: t, Image: value, NeedRegistration: true}
	uri := fmt.Sprintf("/worker/model/%d", id)

	for _, opt := range opts {
		opt(&data)
	}

	if !data.NeedRegistration {
		uri += "?needRegistration=false"
	}

	code, err := c.PutJSON(uri, &data, nil)
	if code > 300 && err == nil {
		return fmt.Errorf("WorkerModelUpdate> HTTP %d", code)
	} else if err != nil {
		return sdk.WrapError(err, "WorkerModelUpdate> Error")
	}
	return nil
}

func withoutRegistrationNeed() WorkerModelConfigParam {
	return func(m *sdk.Model) {
		m.NeedRegistration = false
	}
}
