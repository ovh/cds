package api

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/worker_v2"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkerV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workerGet),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			workerName := vars["workerName"]
			withKey := service.FormBool(req, "withKey")

			var wkr *sdk.V2Worker
			var err error
			if withKey {
				wkr, err = worker_v2.LoadWorkerByName(ctx, api.mustDB(), workerName, gorpmapping.GetOptions.WithDecryption)
				if wkr != nil {
					encoded := base64.StdEncoding.EncodeToString(wkr.PrivateKey)
					wkr.PrivateKey = []byte(encoded)
				}
			} else {
				wkr, err = worker_v2.LoadWorkerByName(ctx, api.mustDB(), workerName)
			}
			if err != nil {
				return err
			}
			return service.WriteJSON(w, wkr, http.StatusOK)
		}
}
