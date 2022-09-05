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
		return err
	}
	if code > 300 {
		return newError(fmt.Errorf("HTTP %d", code))
	}
	return nil
}

// WorkerModelsEnabled retrieves all worker models enabled and available to user.
func (c *client) WorkerModelEnabledList() ([]sdk.Model, error) {
	var models []sdk.Model
	if _, err := c.GetJSON(context.Background(), "/worker/model/enabled", &models); err != nil {
		return nil, err
	}
	return models, nil
}

// WorkerModels retrieves all worker models.
func (c *client) WorkerModelList(filter *WorkerModelFilter) ([]sdk.Model, error) {
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
		return newAPIError(fmt.Errorf("WorkerModelSpawnError> HTTP %d", code))
	} else if err != nil {
		return err
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
		return model, newError(fmt.Errorf("You have to choose 1 model minimum: docker or vm model"))
	}

	switch modelType {
	case sdk.Docker:
		if dockerModel == nil {
			return model, newError(fmt.Errorf("with model %s then dockerModel parameter could not be nil", modelType))
		}
		model.ModelDocker = *dockerModel
	default:
		if vmModel == nil {
			return model, newError(fmt.Errorf("with model %s then vmModel parameter could not be nil", modelType))
		}
		model.ModelVirtualMachine = *vmModel
	}

	modelCreated := sdk.Model{}
	code, err := c.PostJSON(context.Background(), uri, model, &modelCreated)
	if err != nil {
		return modelCreated, err
	}
	if code >= 300 {
		return modelCreated, newAPIError(fmt.Errorf("WorkerModelAdd> HTTP %d", code))
	}

	return modelCreated, nil
}

func (c *client) WorkerModelGet(groupName, name string) (sdk.Model, error) {
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

// WorkerModelSecretList retrieves all worker model's secrets.
func (c *client) WorkerModelSecretList(groupName, name string) (sdk.WorkerModelSecrets, error) {
	url := fmt.Sprintf("/worker/model/%s/%s/secret", groupName, name)
	var secrets sdk.WorkerModelSecrets
	if _, err := c.GetJSON(context.Background(), url, &secrets); err != nil {
		return nil, err
	}
	return secrets, nil
}

type WorkerModelV2Filter struct {
	Branch string
}

func (c *client) WorkerModelv2List(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, filter *WorkerModelV2Filter) ([]sdk.V2WorkerModel, error) {
	var mods []RequestModifier
	if filter != nil {
		mods = []RequestModifier{
			func(req *http.Request) {
				q := req.URL.Query()
				if filter.Branch != "" {
					q.Add("branch", url.QueryEscape(filter.Branch))
				}
				req.URL.RawQuery = q.Encode()
			},
		}
	}
	var models []sdk.V2WorkerModel
	uri := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workermodel", projKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier))
	if _, err := c.GetJSON(ctx, uri, &models, mods...); err != nil {
		return nil, err
	}
	return models, nil
}
