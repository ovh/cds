package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ovh/cds/sdk"
)

// WorkerModelFilter for model's calls.
type WorkerModelFilter struct {
	State  string
	Binary string
}

// WorkerModelBook books a worker model for register, used by hatcheries.
func (c *client) WorkerModelBook(groupName, name string) error {
	code, err := c.PutJSON(context.Background(), fmt.Sprintf("/worker/model/%s/%s/book", groupName, name), nil, nil)
	if err != nil {
		return sdk.WithStack(err)
	}
	if code > 300 {
		return sdk.WithStack(fmt.Errorf("HTTP %d", code))
	}
	return nil
}

// WorkerModelsEnabled retrieves all worker models enabled and available to user.
func (c *client) WorkerModelsEnabled() ([]sdk.Model, error) {
	var models []sdk.Model
	if _, err := c.GetJSON(context.Background(), "/worker/model/enabled", &models); err != nil {
		return nil, err
	}
	return models, nil
}

// WorkerModels retrieves all worker models.
func (c *client) WorkerModels(filter *WorkerModelFilter) ([]sdk.Model, error) {
	var mods []RequestModifier
	if filter != nil {
		mods = []RequestModifier{
			func(req *http.Request) {
				q := req.URL.Query()
				if filter.State != "" {
					q.Add("state", url.QueryEscape(filter.State))
				}
				if filter.Binary != "" {
					q.Add("binary", url.QueryEscape(filter.Binary))
				}
				req.URL.RawQuery = q.Encode()
			},
		}
	}

	var models []sdk.Model
	if _, err := c.GetJSON(context.Background(), "/worker/model", &models, mods...); err != nil {
		return nil, err
	}
	return models, nil
}

func (c *client) WorkerModelSpawnError(groupName, name string, data sdk.SpawnErrorForm) error {
	code, err := c.PutJSON(context.Background(), fmt.Sprintf("/worker/model/%s/%s/error", groupName, name), &data, nil)
	if code > 300 && err == nil {
		return fmt.Errorf("WorkerModelSpawnError> HTTP %d", code)
	} else if err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// WorkerModelAdd create a new worker model available
func (c *client) WorkerModelAdd(name, modelType, patternName string, dockerModel *sdk.ModelDocker, vmModel *sdk.ModelVirtualMachine, groupID int64) (sdk.Model, error) {
	uri := "/worker/model"
	model := sdk.Model{
		Name:        name,
		Type:        modelType,
		GroupID:     groupID,
		PatternName: patternName,
	}

	if dockerModel == nil && vmModel == nil {
		return model, fmt.Errorf("You have to choose 1 model minimum: docker or vm model")
	}

	switch modelType {
	case sdk.Docker:
		if dockerModel == nil {
			return model, fmt.Errorf("with model %s then dockerModel parameter could not be nil", modelType)
		}
		model.ModelDocker = *dockerModel
	default:
		if vmModel == nil {
			return model, fmt.Errorf("with model %s then vmModel parameter could not be nil", modelType)
		}
		model.ModelVirtualMachine = *vmModel
	}

	modelCreated := sdk.Model{}
	code, err := c.PostJSON(context.Background(), uri, model, &modelCreated)
	if err != nil {
		return modelCreated, err
	}
	if code >= 300 {
		return modelCreated, fmt.Errorf("WorkerModelAdd> HTTP %d", code)
	}

	return modelCreated, nil
}

func (c *client) WorkerModel(groupName, name string) (sdk.Model, error) {
	uri := fmt.Sprintf("/worker/model/%s/%s", groupName, name)
	var model sdk.Model
	_, err := c.GetJSON(context.Background(), uri, &model)
	return model, err
}

func (c *client) WorkerModelDelete(groupName, name string) error {
	uri := fmt.Sprintf("/worker/model/%s/%s", groupName, name)
	_, errDelete := c.DeleteJSON(context.Background(), uri, nil)
	return errDelete
}
