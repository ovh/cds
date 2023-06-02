package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/luascript"
)

func setValuesGitInBuildParameters(run *sdk.WorkflowNodeRun, runContext nodeRunContext, vcsInfos vcsInfos) {
	if run.ApplicationID != 0 {
		run.VCSRepository = vcsInfos.Repository
		if vcsInfos.Tag == "" {
			run.VCSBranch = vcsInfos.Branch
		}

		run.VCSTag = vcsInfos.Tag
		run.VCSHash = vcsInfos.Hash
		run.VCSServer = vcsInfos.Server
	}

	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitRepository, sdk.StringParameter, vcsInfos.Repository)

	if vcsInfos.Tag == "" {
		sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitBranch, sdk.StringParameter, vcsInfos.Branch)
	}

	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitTag, sdk.StringParameter, vcsInfos.Tag)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHash, sdk.StringParameter, vcsInfos.Hash)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHashShort, sdk.StringParameter, sdk.StringFirstN(run.VCSHash, 7))
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitAuthor, sdk.StringParameter, vcsInfos.Author)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitMessage, sdk.StringParameter, vcsInfos.Message)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitURL, sdk.StringParameter, vcsInfos.URL)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHTTPURL, sdk.StringParameter, vcsInfos.HTTPUrl)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitServer, sdk.StringParameter, vcsInfos.Server)

	gitContext := sdk.GitContext{
		Hash:       vcsInfos.Hash,
		Repository: vcsInfos.Repository,
		Branch:     vcsInfos.Branch,
		Tag:        vcsInfos.Tag,
		Author:     vcsInfos.Author,
		Message:    vcsInfos.Message,
		URL:        vcsInfos.URL,
		Server:     vcsInfos.Server,
	}
	if runContext.Application.RepositoryStrategy.ConnectionType != "" {
		gitContext.Connection = runContext.Application.RepositoryStrategy.ConnectionType
		gitContext.SSHKey = runContext.Application.RepositoryStrategy.SSHKey
		gitContext.PGPKey = runContext.Application.RepositoryStrategy.PGPKey
		gitContext.HttpUser = runContext.Application.RepositoryStrategy.User

		for _, v := range run.BuildParameters {
			if v.Name == "git.hook" {
				gitContext.EventName = v.Value
				break
			}
		}
	}
	run.Contexts.Git = gitContext
}

func checkCondition(ctx context.Context, wr *sdk.WorkflowRun, conditions sdk.WorkflowNodeConditions, params []sdk.Parameter) bool {
	var conditionsOK bool
	var errc error
	if conditions.LuaScript == "" {
		conditionsOK, errc = sdk.WorkflowCheckConditions(conditions.PlainConditions, params)
	} else {
		luacheck, err := luascript.NewCheck()
		if err != nil {
			log.Warn(ctx, "processWorkflowNodeRun> WorkflowCheckConditions error: %s", err)
			AddWorkflowRunInfo(wr, sdk.SpawnMsgNew(*sdk.MsgWorkflowError, fmt.Sprintf("Error init LUA System: %v", err)))
		}
		luacheck.SetVariables(sdk.ParametersToMap(params))
		errc = luacheck.Perform(conditions.LuaScript)
		conditionsOK = luacheck.Result
	}
	if errc != nil {
		log.Warn(ctx, "processWorkflowNodeRun> WorkflowCheckConditions error: %s", errc)
		AddWorkflowRunInfo(wr, sdk.SpawnMsgNew(*sdk.MsgWorkflowError, fmt.Sprintf("Error on LUA Condition: %v", errc)))
		return false
	}
	return conditionsOK
}

// AddWorkflowRunInfo add WorkflowRunInfo on a WorkflowRun
func AddWorkflowRunInfo(run *sdk.WorkflowRun, infos ...sdk.SpawnMsg) {
	for _, i := range infos {
		run.Infos = append(run.Infos, sdk.WorkflowRunInfo{
			APITime:     time.Now(),
			Message:     i,
			Type:        i.Type,
			SubNumber:   run.LastSubNumber,
			UserMessage: i.DefaultUserMessage(),
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
		return sdk.StatusBuilding
	case counter.failed > 0:
		return sdk.StatusFail
	case counter.stoppped > 0:
		return sdk.StatusStopped
	case counter.success > 0:
		return sdk.StatusSuccess
	case counter.skipped > 0:
		return sdk.StatusSkipped
	case counter.disabled > 0:
		return sdk.StatusDisabled
	default:
		return sdk.StatusNeverBuilt
	}
}

func computeRunStatus(status string, counter *statusCounter) {
	switch status {
	case sdk.StatusSuccess:
		counter.success++
	case sdk.StatusBuilding, sdk.StatusWaiting:
		counter.building++
	case sdk.StatusFail:
		counter.failed++
	case sdk.StatusStopped:
		counter.stoppped++
	case sdk.StatusSkipped:
		counter.skipped++
	case sdk.StatusDisabled:
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
