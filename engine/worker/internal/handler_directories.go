package internal

import (
  "context"
  "net/http"
  "path"

  "github.com/ovh/cds/engine/worker/pkg/workerruntime"
  "github.com/ovh/cds/sdk"
)

func getDirectoriesHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
    ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
    ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

    d := sdk.WorkerDirectories{
      WorkingDir: wk.workingDirAbs,
      BaseDir:    path.Dir(wk.BaseDir().Name()),
    }

    writeJSON(w, d, http.StatusOK)
  }
}
