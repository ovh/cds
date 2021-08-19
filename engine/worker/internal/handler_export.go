package internal

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func addBuildVarHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		// Get body
		data, errra := ioutil.ReadAll(r.Body)
		if errra != nil {
			log.Error(ctx, "addBuildVarHandler> Cannot ReadAll err: %s", errra)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var v sdk.Variable
		if err := sdk.JSONUnmarshal(data, &v); err != nil {
			log.Error(ctx, "addBuildVarHandler> Cannot Unmarshal err: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		v.Name = "cds.build." + v.Name

		wk.currentJob.newVariables = append(wk.currentJob.newVariables, v)
		log.Debug(ctx, "Variable %s added to %+v", v.Name, wk.currentJob.newVariables)
	}
}
