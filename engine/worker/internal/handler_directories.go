package internal

import (
	"context"
	"net/http"
	"path"

	"github.com/ovh/cds/sdk"
)

func getDirectoriesHandler(_ context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d := sdk.WorkerDirectories{
			WorkingDir: wk.workingDirAbs,
			BaseDir:    path.Dir(wk.BaseDir().Name()),
		}

		writeJSON(w, d, http.StatusOK)
	}
}
