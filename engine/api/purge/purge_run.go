package purge

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/luascript"
)

type MarkAsDeleteOptions struct {
	DryRun bool
}

const (
	RunStatus         = "run_status"
	RunDateBefore     = "run_days_before"
	RunGitBranchExist = "git_branch_exist"
)

func GetRetetionPolicyVariables() []string {
	return []string{RunDateBefore, RunStatus, RunGitBranchExist}
}

func markWorkflowRunsToDelete(ctx context.Context, store cache.Store, db *gorp.DbMap, workflowRunsMarkToDelete *stats.Int64Measure) error {
	dao := new(workflow.WorkflowDAO)
	wfs, err := dao.LoadAll(ctx, db)
	if err != nil {
		return err
	}
	for _, wf := range wfs {
		enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, db, FeaturePurgeName, map[string]string{"project_key": wf.ProjectKey})
		if !enabled {
			continue
		}
		if err := ApplyRetentionPolicyOnWorkflow(ctx, store, db, wf, MarkAsDeleteOptions{DryRun: false}, nil); err != nil {
			log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
		}
	}
	workflow.CountWorkflowRunsMarkToDelete(ctx, db, workflowRunsMarkToDelete)
	return nil
}

func ApplyRetentionPolicyOnWorkflow(ctx context.Context, store cache.Store, db *gorp.DbMap, wf sdk.Workflow, opts MarkAsDeleteOptions, u *sdk.AuthentifiedUser) error {
	limit := 50
	offset := 0

	branches, err := getBranchesForWorkflow(ctx, store, db, wf)
	if err != nil {
		return err
	}
	branchesMap := make(map[string]struct{})
	for _, b := range branches {
		branchesMap[b.DisplayID] = struct{}{}
	}
	runs := make([]sdk.WorkflowRunToKeep, 0)
	var nbRunsAnalyzed int64
	for {
		wfRuns, _, _, count, err := workflow.LoadRuns(db, wf.ProjectKey, wf.Name, offset, limit, nil)
		if err != nil {
			return err
		}

		nbRunsAnalyzed = int64(len(wfRuns))
		for _, run := range wfRuns {
			keep, err := applyRetentionPolicyOnRun(db, wf, run, branchesMap, opts)
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

func applyRetentionPolicyOnRun(db *gorp.DbMap, wf sdk.Workflow, run sdk.WorkflowRun, branchesMap map[string]struct{}, opts MarkAsDeleteOptions) (bool, error) {
	if wf.ToDelete && !opts.DryRun {
		if err := workflow.MarkWorkflowRunsAsDelete(db, []int64{run.ID}); err != nil {
			return true, sdk.WithStack(err)
		}
		return false, nil
	}
	luacheck, err := luascript.NewCheck()
	if err != nil {
		return true, sdk.WithStack(err)
	}

	if err := purgeComputeVariables(luacheck, run, branchesMap); err != nil {
		return true, err
	}

	if err := luacheck.Perform(wf.RetentionPolicy); err != nil {
		return true, sdk.NewErrorFrom(sdk.ErrWrongRequest, "%v", err)
	}

	if luacheck.Result {
		return true, nil
	}
	if !opts.DryRun {
		if err := workflow.MarkWorkflowRunsAsDelete(db, []int64{run.ID}); err != nil {
			return true, sdk.WithStack(err)
		}
	}
	return false, nil
}

func purgeComputeVariables(luaCheck *luascript.Check, run sdk.WorkflowRun, branchesMap map[string]struct{}) error {
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

	// If we have a branch in payload, check if it exists on repository branches list
	if b, has := vars["git.branch"]; has {
		_, exist := branchesMap[b]
		vars[RunGitBranchExist] = strconv.FormatBool(exist)
	}
	vars[RunStatus] = run.Status

	varsFloats[RunDateBefore] = math.Floor(time.Now().Sub(run.LastModified).Hours() / 24)

	luaCheck.SetVariables(vars)
	luaCheck.SetFloatVariables(varsFloats)
	return nil
}

func getBranchesForWorkflow(ctx context.Context, store cache.Store, db *gorp.DbMap, wf sdk.Workflow) ([]sdk.VCSBranch, error) {
	appID := wf.WorkflowData.Node.Context.ApplicationID
	if appID != 0 {
		app := wf.Applications[appID]
		if app.RepositoryFullname != "" {
			tx, err := db.Begin()
			if err != nil {
				return nil, sdk.WithStack(err)
			}
			defer tx.Rollback()
			//Get the RepositoriesManager Client
			vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, tx, wf.ProjectKey, app.VCSServer)
			if err != nil {
				log.Debug("SendVCSEvent> No vcsServer found: %v", err)
				return nil, err
			}
			client, err := repositoriesmanager.AuthorizedClient(ctx, tx, store, wf.ProjectKey, vcsServer)
			if err != nil {
				return nil, sdk.WithStack(err)
			}

			branches, err := client.Branches(ctx, app.RepositoryFullname)
			return branches, err
		}
	}
	return nil, nil
}
