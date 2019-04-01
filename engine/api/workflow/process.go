package workflow

import (
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/luascript"
)

func setValuesGitInBuildParameters(run *sdk.WorkflowNodeRun, vcsInfos vcsInfos) {
	run.VCSRepository = vcsInfos.Repository
	run.VCSBranch = vcsInfos.Branch
	run.VCSTag = vcsInfos.Tag
	run.VCSHash = vcsInfos.Hash
	run.VCSServer = vcsInfos.Server

	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitRepository, sdk.StringParameter, run.VCSRepository)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitBranch, sdk.StringParameter, run.VCSBranch)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitTag, sdk.StringParameter, run.VCSTag)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHash, sdk.StringParameter, run.VCSHash)
	if len(run.VCSHash) >= 7 {
		sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHashShort, sdk.StringParameter, run.VCSHash[:7])
	}
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitAuthor, sdk.StringParameter, vcsInfos.Author)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitMessage, sdk.StringParameter, vcsInfos.Message)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitURL, sdk.StringParameter, vcsInfos.URL)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHTTPURL, sdk.StringParameter, vcsInfos.HTTPUrl)
}

func checkNodeRunCondition(wr *sdk.WorkflowRun, conditions sdk.WorkflowNodeConditions, params []sdk.Parameter) bool {
	var conditionsOK bool
	var errc error
	if conditions.LuaScript == "" {
		conditionsOK, errc = sdk.WorkflowCheckConditions(conditions.PlainConditions, params)
	} else {
		luacheck, err := luascript.NewCheck()
		if err != nil {
			log.Warning("processWorkflowNodeRun> WorkflowCheckConditions error: %s", err)
			AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{fmt.Sprintf("Error init LUA System: %v", err)},
			})
		}
		luacheck.SetVariables(sdk.ParametersToMap(params))
		errc = luacheck.Perform(conditions.LuaScript)
		conditionsOK = luacheck.Result
	}
	if errc != nil {
		log.Warning("processWorkflowNodeRun> WorkflowCheckConditions error: %s", errc)
		AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{fmt.Sprintf("Error on LUA Condition: %v", errc)},
		})
		return false
	}
	return conditionsOK
}

// AddWorkflowRunInfo add WorkflowRunInfo on a WorkflowRun
func AddWorkflowRunInfo(run *sdk.WorkflowRun, isError bool, infos ...sdk.SpawnMsg) {
	for _, i := range infos {
		run.Infos = append(run.Infos, sdk.WorkflowRunInfo{
			APITime:   time.Now(),
			Message:   i,
			IsError:   isError,
			SubNumber: run.LastSubNumber,
		})
	}
}

// computeRunStatus is useful to compute number of runs in success, building and fail
type statusCounter struct {
	success, building, failed, stoppped, skipped, disabled int
}

// getRunStatus return the status depending on number of runs in success, building, stopped and fail
func getRunStatus(counter statusCounter) string {
	switch {
	case counter.building > 0:
		return sdk.StatusBuilding.String()
	case counter.failed > 0:
		return sdk.StatusFail.String()
	case counter.stoppped > 0:
		return sdk.StatusStopped.String()
	case counter.success > 0:
		return sdk.StatusSuccess.String()
	case counter.skipped > 0:
		return sdk.StatusSkipped.String()
	case counter.disabled > 0:
		return sdk.StatusDisabled.String()
	default:
		return sdk.StatusNeverBuilt.String()
	}
}

func computeRunStatus(status string, counter *statusCounter) {
	switch status {
	case sdk.StatusSuccess.String():
		counter.success++
	case sdk.StatusBuilding.String(), sdk.StatusWaiting.String():
		counter.building++
	case sdk.StatusFail.String():
		counter.failed++
	case sdk.StatusStopped.String():
		counter.stoppped++
	case sdk.StatusSkipped.String():
		counter.skipped++
	case sdk.StatusDisabled.String():
		counter.disabled++
	}
}

// MaxSubNumber returns the MaxSubNumber of workflowNodeRuns
func MaxSubNumber(workflowNodeRuns map[int64][]sdk.WorkflowNodeRun) int64 {
	var maxsn int64
	for _, wNodeRuns := range workflowNodeRuns {
		for _, wNodeRun := range wNodeRuns {
			if maxsn < wNodeRun.SubNumber {
				maxsn = wNodeRun.SubNumber
			}
		}
	}

	return maxsn
}

func lastSubNumber(workflowNodeRuns []sdk.WorkflowNodeRun) int64 {
	var lastSn int64
	for _, wNodeRun := range workflowNodeRuns {
		if lastSn < wNodeRun.SubNumber {
			lastSn = wNodeRun.SubNumber
		}
	}
	return lastSn
}
