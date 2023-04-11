package action

import (
	"context"
	"fmt"
	"github.com/spf13/afero"
	"os"
	"path/filepath"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunParseCoverageResultAction(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	var res sdk.Result
	res.Status = sdk.StatusFail
	p := sdk.ParameterValue(a.Parameters, "path")
	if p == "" {
		return res, fmt.Errorf("coverage parser: path not provided")
	}

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return res, err
	}

	var fpath string
	var abs string
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		abs, _ = x.RealPath(workdir.Name())
	} else {
		abs = workdir.Name()
	}

	if !sdk.PathIsAbs(p) {
		fpath = filepath.Join(abs, p)
	} else {
		fpath = p
	}

	_, name := filepath.Split(fpath)
	fileMode, err := os.Stat(fpath)
	if err != nil {
		return res, fmt.Errorf("coverage parser: failed to get file stat: %v", err)
	}
	sig, err := wk.RunResultSignature(name, uint32(fileMode.Mode().Perm()), sdk.WorkflowRunResultTypeCoverage)
	if err != nil {
		return res, fmt.Errorf("coverage parser: unable to create signature: %v", err)
	}

	pluginArtifactManagement := wk.GetIntegrationPlugin(sdk.GRPCPluginUploadArtifact)
	if pluginArtifactManagement != nil {
		if err := uploadArtifactByIntegrationPlugin(fpath, ctx, wk, sdk.GRPCPluginUploadArtifact, sdk.ArtifactFileTypeCoverage); err != nil {
			return res, fmt.Errorf("coverage parser: unable to upload in artifact manager: %v", err)
		}
	} else {
		duration, err := wk.Client().CDNItemUpload(ctx, wk.CDNHttpURL(), sig, afero.NewOsFs(), fpath)
		if err != nil {
			return res, fmt.Errorf("coverage parser: unable to upload coverage report: %v", err)
		}
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS CDN", name, duration.Seconds()))
	}
	res.Status = sdk.StatusSuccess
	return res, nil
}
