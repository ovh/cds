package api

import (
	"context"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/purge"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) WorkflowRunCraft(ctx context.Context, tick time.Duration) error {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ids, err := workflow.LoadCratingWorkflowRunIDs(api.mustDB())
			if err != nil {
				log.Error(ctx, "WorkflowRunCraft> unable to start tx: %v", err)
				continue
			}
			for _, id := range ids {
				api.GoRoutines.Exec(
					ctx,
					"workflowRunCraft-"+strconv.FormatInt(id, 10),
					func(ctx context.Context) {
						ctx, span := telemetry.New(ctx, api, "api.workflowRunCraft", nil, trace.SpanKindUnspecified)
						defer span.End()
						if err := api.workflowRunCraft(ctx, id); err != nil {
							log.Error(ctx, "WorkflowRunCraft> error on workflow run %d: %v", id, err)
						}
					},
					api.PanicDump(),
				)
			}
		}
	}

}

func (api *API) workflowRunCraft(ctx context.Context, id int64) error {
	_, next := telemetry.Span(ctx, "api.workflowRunCraft.lock")
	lockKey := cache.Key("api:workflowRunCraft", strconv.FormatInt(id, 10))
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		log.Debug("api.workflowRunCraft> run %d is locked in cache", id)
		next()
		return nil
	}
	next()
	defer func() {
		_ = api.Cache.Unlock(lockKey)
	}()

	_, next = telemetry.Span(ctx, "api.workflowRunCraft.LoadRunByID")
	run, err := workflow.LoadRunByID(api.mustDB(), id, workflow.LoadRunOptions{})
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		next()
		return nil
	}
	if err != nil {
		next()
		return sdk.WrapError(err, "unable to load workflow run %d", id)
	}
	next()

	if !run.ToCraft {
		return nil
	}

	if run.ToCraftOpts == nil {
		return errors.New("unable to craft workflow run without options...")
	}

	_, next = telemetry.Span(ctx, "api.workflowRunCraft.LoadProjectByID")
	proj, err := project.LoadByID(api.mustDB(), run.ProjectID,
		project.LoadOptions.WithVariables,
		project.LoadOptions.WithIntegrations)
	if err != nil {
		next()
		return sdk.WrapError(err, "unable to load project %d", run.ProjectID)
	}
	next()

	wf, err := workflow.LoadByID(ctx, api.mustDB(), api.Cache, *proj, run.WorkflowID, workflow.LoadOptions{
		DeepPipeline:          true,
		WithAsCodeUpdateEvent: true,
		WithIcon:              true,
		WithIntegrations:      true,
		WithTemplate:          true,
	})
	if err != nil {
		return sdk.WrapError(err, "unable to load workflow %d", run.WorkflowID)
	}

	enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), purge.FeatureMaxRuns, map[string]string{"project_key": wf.ProjectKey})
	if enabled {
		countRuns, err := workflow.CountNotPendingWorkflowRunsByWorkflowID(api.mustDB(), run.WorkflowID)
		if err != nil {
			return sdk.WrapError(err, "unable to count workflow runs for workflow %d", run.WorkflowID)
		}
		if countRuns >= wf.MaxRuns {
			// check spawn infos to know if we already check this run
			for _, i := range run.Infos {
				if i.Message.ID == sdk.MsgTooMuchWorkflowRun.ID {
					return nil
				}
			}

			info := sdk.SpawnMsg{
				ID:   sdk.MsgTooMuchWorkflowRun.ID,
				Type: sdk.MsgTooMuchWorkflowRun.Type,
				Args: []interface{}{wf.MaxRuns},
			}
			workflow.AddWorkflowRunInfo(run, info)
			if err := workflow.UpdateWorkflowRun(ctx, api.mustDB(), run); err != nil {
				return err
			}
			event.PublishWorkflowRun(ctx, *run, wf.ProjectKey)
			return nil
		}
		found := false
		for i := range run.Infos {
			if run.Infos[i].Message.ID == sdk.MsgTooMuchWorkflowRun.ID {
				run.Infos[i].Type = sdk.RunInfoTypInfo
				run.Infos[i].Message.Type = sdk.RunInfoTypInfo
				found = true
				break
			}
		}
		if found {
			if err := workflow.UpdateWorkflowRun(ctx, api.mustDB(), run); err != nil {
				return err
			}
			event.PublishWorkflowRun(ctx, *run, wf.ProjectKey)
		}

	}

	log.Debug("api.workflowRunCraft> crafting workflow %s/%s #%d.%d (%d)", proj.Key, wf.Name, run.Number, run.LastSubNumber, run.ID)

	api.initWorkflowRun(ctx, proj.Key, wf, run, *run.ToCraftOpts)

	log.Info(ctx, "api.workflowRunCraft> workflow %s/%s #%d.%d (%d) crafted", proj.Key, wf.Name, run.Number, run.LastSubNumber, run.ID)

	return workflow.UpdateCraftedWorkflowRun(api.mustDB(), run.ID)
}
