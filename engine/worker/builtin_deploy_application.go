package main

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func runDeployApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		sendLog("# Starting application deployment...")
		for _, p := range *params {
			sendLog("#  " + p.Name + ": " + p.Value)
		}
		sendLog("# Downloading plugin from deployment platform...")
		return sdk.Result{
			Status: sdk.StatusSuccess.String(),
		}
	}
}
