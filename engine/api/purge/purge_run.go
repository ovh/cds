package purge

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/luascript"
	"github.com/ovh/cds/sdk/telemetry"
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
	ctx, end := telemetry.Span(ctx, "purge.markWorkflowRunsToDelete")
	defer end()

	dao := new(workflow.WorkflowDAO)
	wfs, err := dao.LoadAll(ctx, db)
	if err != nil {
		return err
	}
	for _, wf := range wfs {
		_, enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, db, sdk.FeaturePurgeName, map[string]string{"project_key": wf.ProjectKey})
		if !enabled {
			continue
		}
		if err := ApplyRetentionPolicyOnWorkflow(ctx, store, db, wf, MarkAsDeleteOptions{DryRun: false}, nil); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "%v", err)
		}
	}
	workflow.CountWorkflowRunsMarkToDelete(ctx, db, workflowRunsMarkToDelete)
	return nil
}

func ApplyRetentionPolicyOnWorkflow(ctx context.Context, store cache.Store, db *gorp.DbMap, wf sdk.Workflow, opts MarkAsDeleteOptions, u *sdk.AuthentifiedUser) error {
	ctx, end := telemetry.Span(ctx, "purge.ApplyRetentionPolicyOnWorkflow")
	defer end()

	var vcsClient sdk.VCSAuthorizedClientService
	var app sdk.Application
	if wf.WorkflowData.Node.Context != nil {
		appID := wf.WorkflowData.Node.Context.ApplicationID
		if appID != 0 {
			appDB, err := application.LoadByID(ctx, db, appID)
			if err != nil {
				return err
			}
			app = *appDB
			if app.RepositoryFullname != "" {
				tx, err := db.Begin()
				if err != nil {
					return sdk.WithStack(err)
				}
				//Get the RepositoriesManager Client
				vcsClient, err = repositoriesmanager.AuthorizedClient(ctx, tx, store, wf.ProjectKey, app.VCSServer)
				if err != nil {
					_ = tx.Rollback()
					return sdk.WithStack(err)
				}
				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	}

	branchesMap := make(map[string]struct{})
	if vcsClient != nil {
		var err error
		branchesMap, err = getBranches(ctx, app.RepositoryFullname, vcsClient)
		if err != nil {
			return err
		}
	}

	runs := make([]sdk.WorkflowRunToKeep, 0)
	var nbRunsAnalyzed int64
	limit := 50
	offset := 0
	eventErrorMsg := make([]string, 0)
	for {

		wfRuns, _, _, count, err := workflow.LoadRunsSummaries(ctx, db, wf.ProjectKey, wf.Name, offset, limit, nil)
		if err != nil {
			return err
		}

		nbRunsAnalyzed = int64(len(wfRuns))
		for _, run := range wfRuns {

			payload, err := extractPayload(run)
			if err != nil {
				return err
			}

			var forkBranches map[string]struct{}
			isFork := false
			if gitRepo, has := payload["git.repository"]; has {
				if gitRepo != app.RepositoryFullname {
					isFork = true
					if vcsClient != nil {
						forkBranches, err = getBranches(ctx, gitRepo, vcsClient)
						if err != nil {
							log.ErrorWithStackTrace(ctx, err)
							version := strconv.FormatInt(run.Number, 10)
							if run.Version != nil {
								version = *run.Version
							}
							eventErrorMsg = append(eventErrorMsg, fmt.Sprintf("unable to get branch from fork %s for run %s", gitRepo, version))
							continue
						}
					}
				}
			}

			var keep bool
			if isFork {
				keep, err = applyRetentionPolicyOnRun(ctx, db, wf, run, payload, forkBranches, app, vcsClient, opts)
			} else {
				keep, err = applyRetentionPolicyOnRun(ctx, db, wf, run, payload, branchesMap, app, vcsClient, opts)
			}
			if keep {
				runs = append(runs, sdk.WorkflowRunToKeep{ID: run.ID, Num: run.Number, Status: run.Status})
			}
			if err != nil {
				log.Error(ctx, "error on run %v:%d err:%v", wf.Name, run.Number, err)
				version := strconv.FormatInt(run.Number, 10)
				if run.Version != nil {
					version = *run.Version
				}
				eventErrorMsg = append(eventErrorMsg, fmt.Sprintf("unable to apply retention policy for run %s", version))
				continue
			}
		}

		if count > offset+limit {
			offset += limit
			if u != nil {
				event.PublishWorkflowRetentionDryRun(ctx, wf.ProjectKey, wf.Name, "INCOMING", "", nil, runs, nbRunsAnalyzed, u)
				runs = runs[:0]
			}
			continue
		}
		break
	}
	if u != nil {
		event.PublishWorkflowRetentionDryRun(ctx, wf.ProjectKey, wf.Name, "DONE", "", eventErrorMsg, runs, nbRunsAnalyzed, u)
	}
	return nil
}

func getBranches(ctx context.Context, repo string, vcsClient sdk.VCSAuthorizedClientService) (map[string]struct{}, error) {
	branchesMap := make(map[string]struct{})
	branches, err := vcsClient.Branches(ctx, repo, sdk.VCSBranchesFilter{})
	if err != nil {
		return nil, sdk.WrapError(err, "cannot retrieve branches for repo %q", repo)
	}
	log.Info(ctx, "Purge getting branch for repo %s - count: %d", repo, len(branches))
	defaultBranchFound := false
	for _, b := range branches {
		branchesMap[b.DisplayID] = struct{}{}
		if b.Default {
			log.Info(ctx, "Getting default branch for repo %s - %s", repo, b.DisplayID)
			defaultBranchFound = true
		}
	}

	if !defaultBranchFound {
		log.Warn(ctx, "Getting default branch for repo %s - not found", repo)
	}
	return branchesMap, nil
}

func extractPayload(run sdk.WorkflowRunSummary) (map[string]string, error) {
	vars := make(map[string]string)
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
					return nil, sdk.WithStack(err)
				}
				for k, v := range tmpVars {
					vars[k] = v
				}
			}
		case run.ToCraftOpts.Hook != nil && run.ToCraftOpts.Hook.Payload != nil:
			vars = run.ToCraftOpts.Hook.Payload
		}
	}
	return vars, nil
}

func applyRetentionPolicyOnRun(ctx context.Context, db *gorp.DbMap, wf sdk.Workflow, run sdk.WorkflowRunSummary, payload map[string]string, branchesMap map[string]struct{}, app sdk.Application, vcsClient sdk.VCSAuthorizedClientService, opts MarkAsDeleteOptions) (bool, error) {
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

	if err := purgeComputeVariables(ctx, luaCheck, run, payload, branchesMap, app, vcsClient); err != nil {
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

func purgeComputeVariables(ctx context.Context, luaCheck *luascript.Check, run sdk.WorkflowRunSummary, payload map[string]string, branchesMap map[string]struct{}, app sdk.Application, vcsClient sdk.VCSAuthorizedClientService) error {
	vars := payload
	varsFloats := make(map[string]float64)

	// git_branch_exist var is often used in the default rule retention
	// this will avoid
	//  <string>:9: variable 'git_branch_exist' is not declared
	// when it's not defined
	if _, ok := vars[RunGitBranchExist]; !ok {
		vars[RunGitBranchExist] = "false"
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
	if vcsClient != nil {
		// Only inject the "git_branch_exist" variable if a vcs client exists to make sure that its value is accurate
		vars[RunGitBranchExist] = strconv.FormatBool(exist)
	}
	vars[RunHasGitBranch] = strconv.FormatBool(has)

	vars[RunStatus] = run.Status

	varsFloats[RunDaysBefore] = math.Floor(time.Now().Sub(run.LastModified).Hours() / 24)

	luaCheck.SetVariables(vars)
	luaCheck.SetFloatVariables(varsFloats)
	return nil
}
