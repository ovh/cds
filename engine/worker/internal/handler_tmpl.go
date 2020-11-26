package internal

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
)

func tmplHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
			writeError(w, r, newError)
			return
		}

		var a workerruntime.TmplPath
		if err := json.Unmarshal(data, &a); err != nil {
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
