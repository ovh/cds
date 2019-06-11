package action

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sguiheux/go-coverage"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func RunParseCoverageResultAction(ctx context.Context, wk workerruntime.Runtime, a *sdk.Action, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
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
	default:
		return res, fmt.Errorf("coverage parser: unknown format %s", mode)
	}
	parser := coverage.New(p, parserMode)
	report, errR := parser.Parse()
	if errR != nil {
		return res, fmt.Errorf("coverage parser: unable to parse report: %v", errR)
	}

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return res, err
	}

	uri := fmt.Sprintf("/queue/workflows/%d/coverage", jobID)
	code, err := wk.Client().(cdsclient.Raw).PostJSON(ctx, uri, report, nil)
	if err == nil && code > 300 {
		err = fmt.Errorf("HTTP %d", code)
	}

	if err != nil {
		return res, fmt.Errorf("coverage parser: failed to send coverage details: %s", err)
	}

	if minReq > 0 {
		covPercent := (float64(report.CoveredLines) / float64(report.TotalLines)) * 100
		if covPercent < minReq {
			return res, fmt.Errorf("coverage: minimum coverage failed: %.2f%% < %.2f%%", covPercent, minReq)
		}
	}

	res.Status = sdk.StatusSuccess
	return res, nil
}
