package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
	"go.opencensus.io/trace"
)

func (api *API) WorkflowTemplateBulk(ctx context.Context, tick time.Duration, chanOperation chan WorkflowTemplateBulkOperation) error {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			bs, err := workflowtemplate.GetBulksPending(ctx, api.mustDBWithCtx(ctx))
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for _, b := range bs {
				api.GoRoutines.Exec(ctx, "workflowTemplateBulk-"+strconv.FormatInt(b.ID, 10),
					func(ctx context.Context) {
						ctx = telemetry.New(ctx, api, "api.workflowTemplateBulk", nil, trace.SpanKindUnspecified)
						if err := api.workflowTemplateBulk(ctx, b.ID, chanOperation); err != nil {
							log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "error on workflow template bulk %d", b.ID))
						}
					},
				)
			}
		}
	}
}

type WorkflowTemplateBulkOperations []WorkflowTemplateBulkOperation

func (o *WorkflowTemplateBulkOperations) Push(op WorkflowTemplateBulkOperation) {
	*o = append(*o, op)
}

func (o *WorkflowTemplateBulkOperations) Pop() WorkflowTemplateBulkOperation {
	old := *o
	n := len(old)
	x := old[n-1]
	*o = old[0 : n-1]
	return x
}

type WorkflowTemplateBulkOperation struct {
	BulkID             int64
	AuthConsumerID     string
	WorkflowTemplateID int64
	Operation          sdk.WorkflowTemplateBulkOperation
	chanResult         chan sdk.WorkflowTemplateBulkOperation
}

func (api *API) workflowTemplateBulk(ctx context.Context, bulkID int64, chanOperation chan WorkflowTemplateBulkOperation) error {
	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	b, err := workflowtemplate.GetAndLockBulkByID(ctx, tx, bulkID)
	if err != nil {
		return err
	}

	b.Status = sdk.OperationStatusProcessing

	if err := workflowtemplate.UpdateBulk(tx, b); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	chanResult := make(chan sdk.WorkflowTemplateBulkOperation)
	defer close(chanResult)

	operationTodo := make(WorkflowTemplateBulkOperations, 0, len(b.Operations))

	// Enqueue operations
	for _, o := range b.Operations {
		if o.Status != sdk.OperationStatusPending {
			continue
		}
		operationTodo.Push(WorkflowTemplateBulkOperation{
			BulkID:             b.ID,
			AuthConsumerID:     b.AuthConsumerID,
			WorkflowTemplateID: b.WorkflowTemplateID,
			Operation:          o,
			chanResult:         chanResult,
		})
	}

	if len(operationTodo) == 0 {
		b.Status = sdk.OperationStatusDone
		return workflowtemplate.UpdateBulk(api.mustDBWithCtx(ctx), b)
	}

	if b.Parallel {
		for len(operationTodo) > 0 {
			chanOperation <- operationTodo.Pop()
		}
	} else {
		chanOperation <- operationTodo.Pop()
	}

	// Wait for all operation to complete, if context is cancelled, set bulk status back to pending so antother API will be able to end the process
	for {
		select {
		case res := <-chanResult:
			b.UpdateOperation(res)
			if err := workflowtemplate.UpdateBulk(api.mustDBWithCtx(ctx), b); err != nil {
				return err
			}
			if b.Status == sdk.OperationStatusDone {
				break
			}
			if (res.Status == sdk.OperationStatusDone || res.Status == sdk.OperationStatusError) && len(operationTodo) > 0 {
				chanOperation <- operationTodo.Pop()
			}
		case <-ctx.Done():
			b.Status = sdk.OperationStatusPending
			return workflowtemplate.UpdateBulk(api.mustDBWithCtx(ctx), b)
		}
	}
}

func (api *API) WorkflowTemplateBulkOperation(ctx context.Context, routineCount int64, chanOperation chan WorkflowTemplateBulkOperation) {
	if routineCount == 0 {
		routineCount = 10
	}
	for i := int64(0); i < routineCount; i++ {
		api.GoRoutines.RunWithRestart(ctx, fmt.Sprintf("api.WorkflowTemplateBulkOperation-%d", i), func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case op := <-chanOperation:
					api.workflowTemplateBulkOperation(ctx, op)
				}
			}
		})
	}
}

func (api *API) workflowTemplateBulkOperation(ctx context.Context, op WorkflowTemplateBulkOperation) {
	if op.Operation.Status != sdk.OperationStatusPending {
		return
	}
	op.Operation.Status = sdk.OperationStatusProcessing
	op.chanResult <- op.Operation

	errorDefer := func(err error) error {
		if err != nil {
			err = sdk.WrapError(err, "error occurred in template bulk with id %d", op.BulkID)
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "%v", err)
			op.Operation.Status = sdk.OperationStatusError
			op.Operation.Error = fmt.Sprintf("%s", sdk.Cause(err))
			op.chanResult <- op.Operation
		}
		return nil
	}

	consumer, err := authentication.LoadUserConsumerByID(ctx, api.mustDB(), op.AuthConsumerID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUserWithContacts,
		authentication.LoadUserConsumerOptions.WithConsumerGroups)
	if err != nil {
		if errD := errorDefer(err); errD != nil {
			log.Error(ctx, "%v", errD)
		}
		return
	}

	// load project with key
	p, err := project.Load(ctx, api.mustDB(), op.Operation.Request.ProjectKey,
		project.LoadOptions.WithGroups,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithIntegrations,
		project.LoadOptions.WithClearKeys,
	)
	if err != nil {
		if errD := errorDefer(err); errD != nil {
			log.Error(ctx, "%v", errD)
		}
		return
	}

	wt, err := workflowtemplate.LoadByID(ctx, api.mustDB(), op.WorkflowTemplateID, workflowtemplate.LoadOptions.Default)
	if err != nil {
		if errD := errorDefer(err); errD != nil {
			log.Error(ctx, "%v", errD)
		}
		return
	}

	// apply and import workflow
	data := exportentities.WorkflowComponents{
		Template: exportentities.TemplateInstance{
			Name:       op.Operation.Request.WorkflowName,
			From:       wt.PathWithVersion(),
			Parameters: op.Operation.Request.Parameters,
		},
	}

	// In case we want to update a workflow that is ascode, we want to create a PR instead of pushing directly the new workflow.
	wti, err := workflowtemplate.LoadInstanceByTemplateIDAndProjectIDAndRequestWorkflowName(ctx, api.mustDB(), wt.ID, p.ID, data.Template.Name)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		if errD := errorDefer(err); errD != nil {
			log.Error(ctx, "%v", errD)
		}
		return
	}
	if wti != nil && wti.WorkflowID != nil {
		existingWorkflow, err := workflow.LoadByID(ctx, api.mustDB(), api.Cache, *p, *wti.WorkflowID, workflow.LoadOptions{})
		if err != nil {
			if errD := errorDefer(err); errD != nil {
				log.Error(ctx, "%v", errD)
			}
			return
		}
		if existingWorkflow.FromRepository != "" {
			var rootApp *sdk.Application
			if existingWorkflow.WorkflowData.Node.Context != nil && existingWorkflow.WorkflowData.Node.Context.ApplicationID != 0 {
				rootApp, err = application.LoadByIDWithClearVCSStrategyPassword(ctx, api.mustDB(), existingWorkflow.WorkflowData.Node.Context.ApplicationID)
				if err != nil {
					if errD := errorDefer(err); errD != nil {
						log.Error(ctx, "%v", errD)
					}
					return
				}
			}
			if rootApp == nil {
				if errD := errorDefer(sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot find the root application of the workflow")); errD != nil {
					log.Error(ctx, "%v", errD)
				}
				return
			}

			if op.Operation.Request.Branch == "" || op.Operation.Request.Message == "" {
				if errD := errorDefer(sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing branch or message data")); errD != nil {
					log.Error(ctx, "%v", errD)
				}
				return
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				if errD := errorDefer(err); errD != nil {
					log.Error(ctx, "%v", errD)
				}
				return
			}
			ope, err := operation.PushOperationUpdate(ctx, tx, api.Cache, *p, data, rootApp.VCSServer, rootApp.RepositoryFullname, op.Operation.Request.Branch, op.Operation.Request.Message, rootApp.RepositoryStrategy, consumer)
			if err != nil {
				tx.Rollback() // nolint
				if errD := errorDefer(err); errD != nil {
					log.Error(ctx, "%v", errD)
				}
				return
			}
			if err := tx.Commit(); err != nil {
				tx.Rollback() // nolint
				if errD := errorDefer(err); errD != nil {
					log.Error(ctx, "%v", errD)
				}
				return
			}

			ed := ascode.EntityData{
				Name:          existingWorkflow.Name,
				ID:            existingWorkflow.ID,
				Type:          ascode.WorkflowEvent,
				FromRepo:      existingWorkflow.FromRepository,
				OperationUUID: ope.UUID,
			}
			ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, api.GoRoutines, *p, *existingWorkflow, *rootApp, ed, consumer)

			op.Operation.Status = sdk.OperationStatusDone
			op.chanResult <- op.Operation
		}
	}

	mods := []workflowtemplate.TemplateRequestModifierFunc{
		workflowtemplate.TemplateRequestModifiers.DefaultKeys(*p),
	}
	_, wti, err = workflowtemplate.CheckAndExecuteTemplate(ctx, api.mustDB(), api.Cache, *consumer, *p, &data, mods...)
	if err != nil {
		if errD := errorDefer(err); errD != nil {
			log.Error(ctx, "%v", errD)
		}
		return
	}

	_, wkf, _, _, err := workflow.Push(ctx, api.mustDB(), api.Cache, p, data, nil, consumer, project.DecryptWithBuiltinKey)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrEnvironmentExist) {
			err = sdk.NewErrorFrom(err, "conflict when creating an environment with same name on multiple workflow")
		}
		if errD := errorDefer(sdk.WrapError(err, "cannot push generated workflow")); errD != nil {
			log.Error(ctx, "%v", errD)
		}
		return
	}

	if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, api.mustDB(), *wkf, *consumer, wti); err != nil {
		if errD := errorDefer(err); errD != nil {
			log.Error(ctx, "%v", errD)
		}
		return
	}

	op.Operation.Status = sdk.OperationStatusDone
	op.chanResult <- op.Operation
}
