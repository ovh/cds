package workflow

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
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
	if _, err := exportWorkflow(wf, exportentities.FormatYAML, buffw, exportentities.WorkflowSkipIfOnlyOneRepoWebhook); err != nil {
		return nil, sdk.WrapError(err, "unable to export workflow")
	}
	wp.Workflow.Name = wf.Name
	wp.Workflow.Value = base64.StdEncoding.EncodeToString(buffw.Bytes())

	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return nil, sdk.WithStack(sdk.ErrApplicationNotFound)
	}

	return operation.PushOperation(ctx, db, store, proj, &app, wp, branch, message, u)
}

// MigrateAsCode does a workflow pull and start an operation to push cds files into the git repository
func MigrateAsCode(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, app sdk.Application, u sdk.Identifiable, encryptFunc sdk.EncryptFunc) (*sdk.Operation, error) {
	// Get repository
	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return nil, sdk.WithStack(sdk.ErrApplicationNotFound)
	}

	// Export workflow
	pull, err := Pull(ctx, db, store, proj, wf.Name, exportentities.FormatYAML, encryptFunc, exportentities.WorkflowSkipIfOnlyOneRepoWebhook)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot pull workflow")
	}

	var message string
	if wf.FromRepository == "" {
		message = fmt.Sprintf("feat: Enable workflow as code [@%s]", u.GetUsername())
	} else {
		message = fmt.Sprintf("chore: Update workflow [@%s]", u.GetUsername())
	}
	return operation.PushOperation(ctx, db, store, proj, &app, pull, fmt.Sprintf("cdsAsCode-%d", time.Now().Unix()), message, u)
}

// CheckPullRequestStatus checks the status of the pull request
func CheckPullRequestStatus(ctx context.Context, client sdk.VCSAuthorizedClient, repoFullName string, prID int64) (bool, bool, error) {
	pr, err := client.PullRequest(ctx, repoFullName, int(prID))
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			log.Debug("Pull request %s #%d not found", repoFullName, int(prID))
			return false, true, nil
		}
		return false, false, sdk.WrapError(err, "unable to check pull request status")
	}
	return pr.Merged, pr.Closed, nil
}

// SyncAsCodeEvent checks if workflow as to become ascode
func SyncAsCodeEvent(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, u sdk.Identifiable) error {
	if len(wf.AsCodeEvent) == 0 {
		return nil
	}

	client, errclient := createVCSClientFromRootNode(ctx, db, store, proj, wf)
	if errclient != nil {
		return errclient
	}
	app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]

	eventLeft := make([]sdk.AsCodeEvent, 0)
	for _, event := range wf.AsCodeEvent {
		merged, closed, err := CheckPullRequestStatus(ctx, client, app.RepositoryFullname, event.PullRequestID)
		if err != nil {
			return err
		}
		// Event merged and workflow not as code yet:  change it to as code workflow
		if merged && wf.FromRepository == "" {
			repo, errR := client.RepoByFullname(ctx, app.RepositoryFullname)
			if errR != nil {
				return sdk.WrapError(errR, "cannot get repo %s", app.RepositoryFullname)
			}
			if app.RepositoryStrategy.ConnectionType == "ssh" {
				wf.FromRepository = repo.SSHCloneURL
			} else {
				wf.FromRepository = repo.HTTPCloneURL
			}
			wf.LastModified = time.Now()

			if err := updateFromRepository(db, wf.ID, wf.FromRepository); err != nil {
				return sdk.WrapError(err, "unable to update workflow from_repository")
			}
		}
		// If event ended, delete it from db
		if merged || closed {
			if err := ascode.DeleteAsCodeEvent(db, event); err != nil {
				return err
			}
		} else {
			eventLeft = append(eventLeft, event)
		}
	}
	wf.AsCodeEvent = eventLeft
	log.Debug("workflow.SyncAsCodeEvent> events left: %v", wf.AsCodeEvent)
	event.PublishWorkflowUpdate(proj.Key, *wf, *wf, u)
	return nil
}
