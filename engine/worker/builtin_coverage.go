package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sguiheux/go-coverage"

	"github.com/ovh/cds/sdk"
)

func runParseCoverageResultAction(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		var res sdk.Result
		res.Status = sdk.StatusFail.String()

		p := sdk.ParameterValue(a.Parameters, "path")
		if p == "" {
			res.Reason = fmt.Sprintf("Coverage parser: path not provided")
			sendLog(res.Reason)
			return res
		}

		mode := sdk.ParameterValue(a.Parameters, "format")
		if mode == "" {
			res.Reason = fmt.Sprintf("Coverage parser: format not provided")
			sendLog(res.Reason)
			return res
		}

		var minReq float64
		minimum := sdk.ParameterValue(a.Parameters, "minimum")
		if minimum == "" {
			minReq = -1
		} else {
			f, errMin := strconv.ParseFloat(minimum, 64)
			if errMin != nil {
				res.Reason = fmt.Sprintf("Coverage parser: wrong value for 'minimum': %s", errMin)
				sendLog(res.Reason)
				return res
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
			res.Reason = fmt.Sprintf("Coverage parser: unknown format %s", mode)
			sendLog(res.Reason)
			return res
		}
		parser := coverage.New(p, parserMode)
		report, errR := parser.Parse()
		if errR != nil {
			res.Reason = fmt.Sprintf("Coverage parser: unable to parse report: %v", errR)
			sendLog(res.Reason)
			return res
		}

		data, errM := json.Marshal(report)
		if errM != nil {
			res.Reason = fmt.Sprintf("Coverage parser: failed to marshal report for cds api: %v", errM)
			res.Status = sdk.StatusFail.String()
			sendLog(res.Reason)
			return res
		}

		uri := fmt.Sprintf("/queue/workflows/%d/coverage", w.currentJob.wJob.ID)

		_, code, err := sdk.Request("POST", uri, data)
		if err == nil && code > 300 {
			err = fmt.Errorf("HTTP %d", code)
		}

		if err != nil {
			res.Reason = fmt.Sprintf("Coverage parser: failed to send coverage details: %s", err)
			res.Status = sdk.StatusFail.String()
			sendLog(res.Reason)
			return res
		}

		if minReq > 0 {
			covPercent := (float64(report.CoveredLines) / float64(report.TotalLines)) * 100
			if covPercent < minReq {
				res.Reason = fmt.Sprintf("Coverage: minimum coverage failed: %.2f%% < %.2f%%", covPercent, minReq)
				res.Status = sdk.StatusFail.String()
				sendLog(res.Reason)
				return res
			}
		}

		res.Status = sdk.StatusSuccess.String()
		return res
	}
}
