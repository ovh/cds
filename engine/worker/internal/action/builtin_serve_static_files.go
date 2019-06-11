package action

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunServeStaticFiles(ctx context.Context, wk workerruntime.Runtime, a *sdk.Action, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return res, err
	}

	path := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "path"))
	if path == "" {
		path = "."
	}

	name := sdk.ParameterFind(a.Parameters, "name")
	if name == nil || name.Value == "" {
		return res, errors.New("name parameter is empty. aborting")
	}
	staticKey := sdk.ParameterValue(a.Parameters, "static-key")

	// Global all files matching filePath
	filesPath, err := filepath.Glob(path)
	if err != nil {
		return res, fmt.Errorf("cannot perform globbing of pattern '%s': %s", path, err)
	}

	if len(filesPath) == 0 {
		return res, fmt.Errorf("pattern '%s' matched no file", path)
	}

	entrypoint := sdk.ParameterFind(a.Parameters, "entrypoint")
	if entrypoint == nil {
		entrypoint = &sdk.Parameter{}
	}

	// To set entrypoint dynamically when the path is a single file
	if entrypoint.Value == "" && len(filesPath) == 1 {
		fileStat, errS := os.Stat(filesPath[0])
		if errS != nil {
			return res, fmt.Errorf("cannot stat file %s : %v", filesPath[0], errS)
		}
		if !fileStat.IsDir() {
			entrypoint.Value = filepath.Base(filesPath[0])
		}
	}

	if entrypoint.Value == "" {
		entrypoint.Value = "index.html"
	}

	wk.SendLog(workerruntime.LevelInfo, "Fetching files in progress...")
	file, _, err := sdk.CreateTarFromPaths(wk.Workspace(), path, filesPath, &sdk.TarOptions{TrimDirName: filepath.Dir(path)})
	if err != nil {
		return res, fmt.Errorf("cannot tar files: %v", err)
	}

	integrationName := sdk.DefaultIfEmptyStorage(strings.TrimSpace(sdk.ParameterValue(a.Parameters, "destination")))
	projectKey := sdk.ParameterValue(params, "cds.project")

	wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf(`Upload and serving files in progress... with entrypoint "%s"`, entrypoint.Value))
	publicURL, _, _, err := wk.Client().QueueStaticFilesUpload(ctx, projectKey, integrationName, jobID, name.Value, entrypoint.Value, staticKey, file)
	if err != nil {
		return res, fmt.Errorf("Cannot upload static files: %v", err)
	}

	wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("Your files are serving at this URL: %s", publicURL))
	wk.SendLog(workerruntime.LevelInfo, "If you are in the CDS UI you can find all your static files in the artifact tab")

	return res, nil
}
