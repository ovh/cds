package action

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sguiheux/go-coverage"
	"github.com/spf13/afero"

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

	mode := sdk.ParameterValue(a.Parameters, "format")
	if mode == "" {
		return res, fmt.Errorf("coverage parser: format not provided")
	}

	var minReq float64
	minimum := sdk.ParameterValue(a.Parameters, "minimum")
	if minimum == "" {
		minReq = -1
	} else {
		f, errMin := strconv.ParseFloat(minimum, 64)
		if errMin != nil {
			return res, fmt.Errorf("coverage parser: wrong value for 'minimum': %s", errMin)
		}
		minReq = f
	}

	var parserMode coverage.CoverageMode
	switch mode {
	case string(coverage.COBERTURA):
		parserMode = coverage.COBERTURA
	case string(coverage.LCOV):
		parserMode = coverage.LCOV
	case string(coverage.CLOVER):
		parserMode = coverage.CLOVER
	case "other":
	default:
		return res, fmt.Errorf("coverage parser: unknown format %s", mode)
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

	pluginArtifactManagement := wk.GetPlugin(sdk.GRPCPluginUploadArtifact)
	if pluginArtifactManagement != nil {
		if err := uploadArtifactByIntegrationPlugin(fpath, ctx, wk, pluginArtifactManagement, sdk.ArtifactFileTypeCoverage); err != nil {
			return res, fmt.Errorf("coverage parser: unable to upload in artifact manager: %v", err)
		}
	} else {
		duration, err := wk.Client().CDNItemUpload(ctx, wk.CDNHttpURL(), sig, afero.NewOsFs(), fpath)
		if err != nil {
			return res, fmt.Errorf("coverage parser: unable to upload coverage report: %v", err)
		}
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS CDN", name, duration.Seconds()))
	}

	if parserMode != "" {
		parser := coverage.New(fpath, parserMode)
		report, err := parser.Parse()
		if err != nil {
			return res, fmt.Errorf("coverage parser: unable to parse report: %v", err)
		}

		jobID, err := workerruntime.JobID(ctx)
		if err != nil {
			return res, err
		}

		if err := wk.Client().QueueSendCoverage(ctx, jobID, report); err != nil {
			return res, fmt.Errorf("coverage parser: failed to send coverage details: %s", err)
		}
		if parserMode != "" && minReq > 0 {
			covPercent := (float64(report.CoveredLines) / float64(report.TotalLines)) * 100
			if covPercent < minReq {
				return res, fmt.Errorf("coverage: minimum coverage failed: %.2f%% < %.2f%%", covPercent, minReq)
			}
		}
	}

	res.Status = sdk.StatusSuccess
	return res, nil
}
