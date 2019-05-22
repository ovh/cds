package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/worker"
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
		force := FormBool(r, "force")

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

		wm, err := worker.ParseAndImport(tx, api.Cache, &eWorkerModel, force, getAPIConsumer(ctx))
		if err != nil {
			return sdk.WrapError(err, "cannot parse and import worker model")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, *wm, http.StatusOK)
	}
}
