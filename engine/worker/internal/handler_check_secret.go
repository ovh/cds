package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func checkSecretHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var a workerruntime.FilePath
		if err := json.Unmarshal(data, &a); err != nil {
			wk.SendLog(workerruntime.LevelError, fmt.Sprintf("failed to unmarshal %s", data))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		btes, err := ioutil.ReadFile(a.Path)
		if err != nil {
			wk.SendLog(workerruntime.LevelError, fmt.Sprintf("failed to read file %s", a.Path))
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
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
		wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("no secret found in file %s", a.Path))
	}
}
