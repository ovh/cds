package internal

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

func tmplHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
			writeError(w, r, newError)
			return
		}

		var a workerruntime.TmplPath
		if err := sdk.JSONUnmarshal(data, &a); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}

		btes, err := ioutil.ReadFile(a.Path)
		if err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}

		tmpvars := map[string]string{}
		for _, v := range wk.currentJob.newVariables {
			tmpvars[v.Name] = v.Value
		}
		for _, v := range wk.currentJob.params {
			tmpvars[v.Name] = v.Value
		}
		for _, v := range wk.currentJob.secrets {
			tmpvars[v.Name] = v.Value
		}

		res, err := interpolate.Do(string(btes), tmpvars)
		if err != nil {
			log.Error(ctx, "tmpl> Unable to interpolate: %v", err)
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}

		if err := ioutil.WriteFile(a.Destination, []byte(res), os.FileMode(0644)); err != nil {
			log.Error(ctx, "tmpl> Unable to write file: %v", err)
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}
	}
}
