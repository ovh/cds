package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovh/cds/sdk"
)

func runServeStaticFiles(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		res := sdk.Result{Status: sdk.StatusSuccess.String()}

		path := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "path"))
		if path == "" {
			path = "."
		}

		name := sdk.ParameterFind(&a.Parameters, "name")
		if name == nil || name.Value == "" {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("name parameter is empty. aborting")
			sendLog(res.Reason)
			return res
		}
		staticKey := sdk.ParameterValue(a.Parameters, "static-key")

		// Global all files matching filePath
		filesPath, err := filepath.Glob(path)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("cannot perform globbing of pattern '%s': %s", path, err)
			sendLog(res.Reason)
			return res
		}

		if len(filesPath) == 0 {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Pattern '%s' matched no file", path)
			sendLog(res.Reason)
			return res
		}

		entrypoint := sdk.ParameterFind(&a.Parameters, "entrypoint")
		if entrypoint == nil {
			entrypoint = &sdk.Parameter{}
		}

		// To set entrypoint dynamically when the path is a single file
		if entrypoint.Value == "" && len(filesPath) == 1 {
			fileStat, errS := os.Stat(filesPath[0])
			if errS != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("Cannot stat file %s : %v", filesPath[0], errS)
				sendLog(res.Reason)
				return res
			}
			if !fileStat.IsDir() {
				entrypoint.Value = filepath.Base(filesPath[0])
			}
		}

		if entrypoint.Value == "" {
			entrypoint.Value = "index.html"
		}

		sendLog("Fetching files in progress...")
		file, err := sdk.CreateTarFromPaths(filepath.Join(w.currentJob.workingDirectory, filepath.Dir(path)), filesPath, &sdk.TarOptions{TrimDirName: filepath.Dir(path)})
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Cannot tar files: %v", err)
			sendLog(res.Reason)
			return res
		}

		integrationName := sdk.DefaultIfEmptyStorage(strings.TrimSpace(sdk.ParameterValue(a.Parameters, "destination")))
		projectKey := sdk.ParameterValue(*params, "cds.project")

		sendLog(fmt.Sprintf(`Upload and serving files in progress... with entrypoint "%s"`, entrypoint.Value))
		publicURL, _, _, err := w.client.QueueStaticFilesUpload(ctx, projectKey, integrationName, buildID, name.Value, entrypoint.Value, staticKey, file)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Cannot upload static files: %v", err)
			sendLog(res.Reason)
			return res
		}

		sendLog(fmt.Sprintf("Your files are serving at this URL: %s", publicURL))
		sendLog("If you are in the CDS UI you can find all your static files in the artifact tab")

		return res
	}
}
