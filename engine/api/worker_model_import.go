package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// postWorkerModelImportHandler import a worker model via file
// @title import a worker model yml/json file
// @description import a worker model yml/json file with `cdsctl worker model import mywm.yml`
// @params force=true or false. If false and if the worker model already exists, raise an error
func (api *API) postWorkerModelImportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getAPIConsumer(ctx)

		force := service.FormBool(r, "force")

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		var eWorkerModel exportentities.WorkerModel
		var errUnMarshall error
		switch contentType {
		case "application/json":
			errUnMarshall = json.Unmarshal(body, &eWorkerModel)
		case "application/x-yaml", "text/x-yam":
			errUnMarshall = yaml.Unmarshal(body, &eWorkerModel)
		default:
			return sdk.WrapError(sdk.ErrWrongRequest, "Unsupported content-type: %s", contentType)
		}
		if errUnMarshall != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errUnMarshall)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}
		defer tx.Rollback() //nolint

		data := eWorkerModel.GetWorkerModel()

		// group name should be set
		if data.Group == nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing group name")
		}

		// check that the user is admin on the given template's group
		grp, err := group.LoadByName(ctx, tx, data.Group.Name, group.LoadOptions.WithMembers)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		data.GroupID = grp.ID
		if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "you should be admin of the group to import a worker model")
		}

		// validate worker model fields
		if err := data.IsValid(); err != nil {
			return err
		}

		var newModel *sdk.Model

		// check if a model already exists for given info, if exists but not force update returns an error
		old, err := workermodel.LoadByNameAndGroupID(ctx, tx, data.Name, grp.ID)
		if err != nil {
			if !isAdmin(ctx) {
				// if current user is not admin and model is not restricted, a pattern should be given
				if !data.Restricted && data.PatternName == "" {
					return sdk.NewErrorFrom(sdk.ErrWorkerModelNoPattern, "missing model pattern name")
				}
			}

			// validate worker model type fields
			if err := data.IsValidType(); err != nil {
				return err
			}

			newModel, err = workermodel.Create(ctx, tx, data, consumer)
			if err != nil {
				return err
			}
		} else if force {
			if !isAdmin(ctx) {
				if err := workermodel.CopyModelTypeData(old, &data); err != nil {
					return err
				}
			}

			// validate worker model type fields
			if err := data.IsValidType(); err != nil {
				return err
			}

			newModel, err = workermodel.Update(ctx, tx, old, data)
			if err != nil {
				return err
			}
		} else {
			return sdk.NewErrorFrom(sdk.ErrModelNameExist, "worker model already exists with name %s for group %s", data.Name, grp.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		newModel, err = workermodel.LoadByID(ctx, api.mustDB(), newModel.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return err
		}

		newModel.Editable = true

		return service.WriteJSON(w, newModel, http.StatusOK)
	}
}
