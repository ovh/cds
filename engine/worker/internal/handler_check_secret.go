package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func checkSecretHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			returnHTTPError(ctx, w, 400, errRead)
			return
		}

		var a workerruntime.FilePath
		if err := sdk.JSONUnmarshal(data, &a); err != nil {
			returnHTTPError(ctx, w, 400, fmt.Errorf("failed to unmarshal %s", data))
			return
		}

		btes, err := ioutil.ReadFile(a.Path)
		if err != nil {
			returnHTTPError(ctx, w, 400, fmt.Errorf("failed to read file %s", a.Path))
			return
		}
		sbtes := string(btes)

		var varFound string
		for _, p := range wk.currentJob.params {
			if (p.Type == sdk.SecretVariable || p.Type == sdk.KeyVariable) && len(p.Value) >= sdk.SecretMinLength && strings.Contains(sbtes, p.Value) {
				varFound = p.Name
				break
			}
		}

		if varFound != "" {
			writeByteArray(w, []byte(fmt.Sprintf("secret variable %s is used in file %s", varFound, a.Path)), http.StatusExpectationFailed)
			return
		}
	}
}
