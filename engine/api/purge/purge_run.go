package purge

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/luascript"
)

type MarkAsDeleteOptions struct {
	DryRun bool
}

const (
	RunStatus          = "run_status"
	RunDaysBefore      = "run_days_before"
	RunHasGitBranch    = "has_git_branch"
	RunGitBranchExist  = "git_branch_exist"
	RunChangeExist     = "gerrit_change_exist"
	RunChangeMerged    = "gerrit_change_merged"
	RunChangeAbandoned = "gerrit_change_abandoned"
	RunChangeDayBefore = "gerrit_change_days_before"
)

func GetRetentionPolicyVariables() []string {
	return []string{RunDaysBefore, RunStatus, RunHasGitBranch, RunGitBranchExist, RunChangeMerged, RunChangeAbandoned, RunChangeDayBefore, RunChangeExist}
}

func markWorkflowRunsToDelete(ctx context.Context, store cache.Store, db *gorp.DbMap, workflowRunsMarkToDelete *stats.Int64Measure) error {
	dao := new(workflow.WorkflowDAO)
	wfs, err := dao.LoadAll(ctx, db)
	if err != nil {
		return err
	}
	for _, wf := range wfs {
		if err := ApplyRetentionPolicyOnWorkflow(ctx, store, db, wf, MarkAsDeleteOptions{DryRun: false}, nil); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "%v", err)
		}
	}
	workflow.CountWorkflowRunsMarkToDelete(ctx, db, workflowRunsMarkToDelete)
	return nil
}

func ApplyRetentionPolicyOnWorkflow(ctx context.Context, store cache.Store, db *gorp.DbMap, wf sdk.Workflow, opts MarkAsDeleteOptions, u *sdk.AuthentifiedUser) error {
	var vcsClient sdk.VCSAuthorizedClientService
	var app sdk.Application
	if wf.WorkflowData.Node.Context != nil {
		appID := wf.WorkflowData.Node.Context.ApplicationID
		if appID != 0 {
			app = wf.Applications[appID]
			if app.RepositoryFullname != "" {
				tx, err := db.Begin()
				if err != nil {
					return sdk.WithStack(err)
				}
				//Get the RepositoriesManager Client
				vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, tx, wf.ProjectKey, app.VCSServer)
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				vcsClient, err = repositoriesmanager.AuthorizedClient(ctx, tx, store, wf.ProjectKey, vcsServer)
				if err != nil {
					_ = tx.Rollback()
					return sdk.WithStack(err)
				}
			}
		}
	}

	branchesMap := make(map[string]struct{})
	if vcsClient != nil {
		branches, err := vcsClient.Branches(ctx, app.RepositoryFullname)
		if err != nil {
			return err
		}
		for _, b := range branches {
			branchesMap[b.DisplayID] = struct{}{}
		}
	}

	runs := make([]sdk.WorkflowRunToKeep, 0)
	var nbRunsAnalyzed int64
	limit := 50
	offset := 0
	for {
		wfRuns, _, _, count, err := workflow.LoadRunsSummaries(db, wf.ProjectKey, wf.Name, offset, limit, nil)
		if err != nil {
			return err
		}

		nbRunsAnalyzed = int64(len(wfRuns))
		for _, run := range wfRuns {
			keep, err := applyRetentionPolicyOnRun(ctx, db, wf, run, branchesMap, app, vcsClient, opts)
			if err != nil {
				return err
			}
			if keep {
				runs = append(runs, sdk.WorkflowRunToKeep{ID: run.ID, Num: run.Number, Status: run.Status})
			}
		}

		if count > offset+limit {
			offset += limit
			if u != nil {
				event.PublishWorkflowRetentionDryRun(ctx, wf.ProjectKey, wf.Name, "INCOMING", "", runs, nbRunsAnalyzed, u)
				runs = runs[:0]
			}
			continue
		}
		break
	}
	if u != nil {
		event.PublishWorkflowRetentionDryRun(ctx, wf.ProjectKey, wf.Name, "DONE", "", runs, nbRunsAnalyzed, u)
	}
	return nil
}

func applyRetentionPolicyOnRun(ctx context.Context, db *gorp.DbMap, wf sdk.Workflow, run sdk.WorkflowRunSummary, branchesMap map[string]struct{}, app sdk.Application, vcsClient sdk.VCSAuthorizedClientService, opts MarkAsDeleteOptions) (bool, error) {
	if wf.ToDelete && !opts.DryRun {
		if err := workflow.MarkWorkflowRunsAsDelete(db, []int64{run.ID}); err != nil {
			return true, sdk.WithStack(err)
		}
		return false, nil
	}

	luaCheck, err := luascript.NewCheck()
	if err != nil {
		return true, sdk.WithStack(err)
	}

	if err := purgeComputeVariables(ctx, luaCheck, run, branchesMap, app, vcsClient); err != nil {
		return true, err
	}

	retentionPolicy := defaultRunRetentionPolicy
	if wf.RetentionPolicy != "" {
		retentionPolicy = wf.RetentionPolicy
	}

	// Enabling strict checks on variables to prevent errors on rule definition
	if err := luaCheck.EnableStrict(); err != nil {
		return true, sdk.WithStack(err)
	}

	if err := luaCheck.Perform(retentionPolicy); err != nil {
		return true, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to apply retention policy on workflow %s/%s: %v", wf.ProjectKey, wf.Name, err)
	}

	if luaCheck.Result {
		return true, nil
	}
	if !opts.DryRun {
		if err := workflow.MarkWorkflowRunsAsDelete(db, []int64{run.ID}); err != nil {
			return true, sdk.WithStack(err)
		}
	}
	return false, nil
}

func purgeComputeVariables(ctx context.Context, luaCheck *luascript.Check, run sdk.WorkflowRunSummary, branchesMap map[string]struct{}, app sdk.Application, vcsClient sdk.VCSAuthorizedClientService) error {
	vars := make(map[string]string)
	varsFloats := make(map[string]float64)

	// Add payload as variable
	if run.ToCraftOpts != nil {
		switch {
		case run.ToCraftOpts.Manual != nil:
			payload := run.ToCraftOpts.Manual.Payload
			if payload != nil {
				// COMPUTE PAYLOAD
				e := dump.NewDefaultEncoder()
				e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
				e.ExtraFields.DetailedMap = false
				e.ExtraFields.DetailedStruct = false
				e.ExtraFields.Len = false
				e.ExtraFields.Type = false
				tmpVars, err := e.ToStringMap(payload)
				if err != nil {
					return sdk.WithStack(err)
				}
				for k, v := range tmpVars {
					vars[k] = v
				}
			}
		case run.ToCraftOpts.Hook != nil && run.ToCraftOpts.Hook.Payload != nil:
			vars = run.ToCraftOpts.Hook.Payload
		}
	}

	// If we have gerrit change id, check status
	if changeID, ok := vars["gerrit.change.id"]; ok && vcsClient != nil {
		ch, err := vcsClient.PullRequest(ctx, app.RepositoryFullname, changeID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			vars[RunChangeExist] = "false"
		} else {
			vars[RunChangeExist] = "true"
			vars[RunChangeMerged] = strconv.FormatBool(ch.Merged)
			vars[RunChangeAbandoned] = strconv.FormatBool(ch.Closed)
			varsFloats[RunChangeDayBefore] = math.Floor(time.Now().Sub(ch.Updated).Hours())
		}

	}

	// If we have a branch in payload, check if it exists on repository branches list
	b, has := vars["git.branch"]
	var exist bool
	if has {
		_, exist = branchesMap[b]
	}
	vars[RunHasGitBranch] = strconv.FormatBool(has)
	vars[RunGitBranchExist] = strconv.FormatBool(exist)

	vars[RunStatus] = run.Status

	varsFloats[RunDaysBefore] = math.Floor(time.Now().Sub(run.LastModified).Hours() / 24)

	luaCheck.SetVariables(vars)
	luaCheck.SetFloatVariables(varsFloats)
	return nil
}
