package workflow

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// UpdateWorkflowAsCode update an as code workflow
func UpdateWorkflowAsCode(ctx context.Context, store cache.Store, db gorp.SqlExecutor, proj *sdk.Project, wf sdk.Workflow, app sdk.Application, branch string, message string, u *sdk.AuthentifiedUser) (*sdk.Operation, error) {
	if err := RenameNode(ctx, db, &wf); err != nil {
		return nil, err
	}
	if err := IsValid(ctx, store, db, &wf, proj, LoadOptions{DeepPipeline: true}); err != nil {
		return nil, err
	}

	var wp exportentities.WorkflowPulled
	buffw := new(bytes.Buffer)
	if _, err := exportWorkflow(ctx, wf, exportentities.FormatYAML, buffw, exportentities.WorkflowSkipIfOnlyOneRepoWebhook); err != nil {
		return nil, sdk.WrapError(err, "unable to export workflow")
	}
	wp.Workflow.Name = wf.Name
	wp.Workflow.Value = base64.StdEncoding.EncodeToString(buffw.Bytes())

	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return nil, sdk.WithStack(sdk.ErrApplicationNotFound)
	}

	return operation.PushOperation(ctx, db, store, proj, &app, wp, branch, message, true, u)
}

// MigrateAsCode does a workflow pull and start an operation to push cds files into the git repository
func MigrateAsCode(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, app sdk.Application, u sdk.Identifiable, encryptFunc sdk.EncryptFunc, branch, message string) (*sdk.Operation, error) {
	// Get repository
	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return nil, sdk.WithStack(sdk.ErrApplicationNotFound)
	}

	// Export workflow
	pull, err := Pull(ctx, db, store, proj, wf.Name, exportentities.FormatYAML, encryptFunc, exportentities.WorkflowSkipIfOnlyOneRepoWebhook)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot pull workflow")
	}

	if message == "" {
		if wf.FromRepository == "" {
			message = fmt.Sprintf("feat: Enable workflow as code [@%s]", u.GetUsername())
		} else {
			message = fmt.Sprintf("chore: Update workflow [@%s]", u.GetUsername())
		}
	}
	if branch == "" {
		branch = fmt.Sprintf("cdsAsCode-%d", time.Now().Unix())
	}
	return operation.PushOperation(ctx, db, store, proj, &app, pull, branch, message, false, u)
}
