package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
)

func runRelease(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		artifactList := sdk.ParameterFind(a.Parameters, "artifacts")
		tag := sdk.ParameterFind(a.Parameters, "tag")
		title := sdk.ParameterFind(a.Parameters, "title")
		releaseNote := sdk.ParameterFind(a.Parameters, "releaseNote")

		pkey := sdk.ParameterFind(*params, "cds.project")
		wName := sdk.ParameterFind(*params, "cds.workflow")
		workflowNum := sdk.ParameterFind(*params, "cds.run.number")

		if pkey == nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "cds.project variable not found.",
			}
			sendLog(res.Reason)
			return res
		}

		if wName == nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "cds.workflow variable not found.",
			}
			sendLog(res.Reason)
			return res
		}

		if workflowNum == nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "cds.run.number variable not found.",
			}
			sendLog(res.Reason)
			return res
		}

		if tag == nil || tag.Value == "" {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Tag name is not set. Nothing to perform.",
			}
			sendLog(res.Reason)
			return res
		}

		if title == nil || title.Value == "" {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Release title is not set.",
			}
			sendLog(res.Reason)
			return res
		}

		if releaseNote == nil || releaseNote.Value == "" {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Release not is not set.",
			}
			sendLog(res.Reason)
			return res
		}

		wRunNumber, errI := strconv.ParseInt(workflowNum.Value, 10, 64)
		if errI != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Workflow number is not a number. Got %s: %s", workflowNum.Value, errI),
			}
			sendLog(res.Reason)
			return res
		}

		artSplitted := strings.Split(artifactList.Value, ",")
		req := sdk.WorkflowNodeRunRelease{
			ReleaseContent: releaseNote.Value,
			ReleaseTitle:   title.Value,
			TagName:        tag.Value,
			Artifacts:      artSplitted,
		}

		if err := w.client.WorkflowNodeRunRelease(pkey.Value, wName.Value, wRunNumber, w.currentJob.wJob.WorkflowNodeRunID, req); err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Cannot make workflow node run release: %s", err),
			}
			sendLog(res.Reason)
			return res
		}

		return sdk.Result{Status: sdk.StatusSuccess.String()}
	}
}
