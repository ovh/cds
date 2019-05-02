package workflow

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

var CacheOperationKey = cache.Key("repositories", "operation", "push")

// UpdateAsCode does a workflow pull and start an operation to push cds files into the git repository
func UpdateAsCode(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, encryptFunc sdk.EncryptFunc, u *sdk.User) (*sdk.Operation, error) {
	// Get repository
	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return nil, sdk.WithStack(sdk.ErrApplicationNotFound)
	}

	app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
	if app.VCSServer == "" || app.RepositoryFullname == "" {
		return nil, sdk.WithStack(sdk.ErrRepoNotFound)
	}

	client, errclient := createVCSClientFromRootNode(ctx, db, store, proj, wf)
	if errclient != nil {
		return nil, errclient
	}

	repo, errR := client.RepoByFullname(ctx, app.RepositoryFullname)
	if errR != nil {
		return nil, sdk.WrapError(errR, "cannot get repo %s", app.RepositoryFullname)
	}

	// Export workflow
	pull, err := Pull(ctx, db, store, proj, wf.Name, exportentities.FormatYAML, encryptFunc, u, exportentities.WorkflowSkipIfOnlyOneRepoWebhook)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot pull workflow")
	}

	buf := new(bytes.Buffer)
	if err := pull.Tar(buf); err != nil {
		return nil, sdk.WrapError(err, "cannot tar pulled workflow")
	}

	var vcsStrategy = app.RepositoryStrategy

	if vcsStrategy.SSHKey != "" {
		key := proj.GetSSHKey(vcsStrategy.SSHKey)
		if key == nil {
			return nil, fmt.Errorf("unable to find key %s on project %s", vcsStrategy.SSHKey, proj.Key)
		}
		vcsStrategy.SSHKeyContent = key.Private
	} else {
		if err := application.DecryptVCSStrategyPassword(&app); err != nil {
			return nil, sdk.WrapError(err, "unable to decrypt vcs strategy")
		}
		vcsStrategy = app.RepositoryStrategy
	}

	// Create VCS Operation
	ope := sdk.Operation{
		VCSServer:          app.VCSServer,
		RepoFullName:       app.RepositoryFullname,
		URL:                "",
		RepositoryStrategy: vcsStrategy,
		Setup: sdk.OperationSetup{
			Push: sdk.OperationPush{
				FromBranch: fmt.Sprintf("cdsAsCode-%d", time.Now().Unix()),
				Message:    "",
			},
		},
		User: sdk.User{
			Username: u.Username,
			Email:    u.Email,
		},
	}

	if app.RepositoryStrategy.ConnectionType == "ssh" {
		ope.URL = repo.SSHCloneURL
	} else {
		ope.URL = repo.HTTPCloneURL
	}

	if wf.FromRepository == "" {
		ope.Setup.Push.Message = fmt.Sprintf("feat: Enable workflow as code [@%s]", u.Username)
	} else {
		ope.Setup.Push.Message = fmt.Sprintf("chore: Update workflow [@%s]", u.Username)
	}

	multipartData := &services.MultiPartData{
		Reader:      buf,
		ContentType: "application/tar",
	}
	if err := PostRepositoryOperation(ctx, db, *proj, &ope, multipartData); err != nil {
		return nil, sdk.WrapError(err, "unable to post repository operation")
	}
	store.SetWithTTL(cache.Key(CacheOperationKey, ope.UUID), ope, 300)

	return &ope, nil
}

// CheckPullRequestStatus checks the status of the pull request
func CheckPullRequestStatus(ctx context.Context, client sdk.VCSAuthorizedClient, repoFullName string, prID int64) (bool, bool, error) {
	pr, err := client.PullRequest(ctx, repoFullName, int(prID))
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, true, nil
		}
		return false, false, sdk.WrapError(err, "unable to check pull request status")
	}
	return pr.Merged, pr.Closed, nil
}

// SyncAsCodeEvent checks if workflow as to become ascode
func SyncAsCodeEvent(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, u *sdk.User) error {
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
			if err := deleteAsCodeEvent(db, event); err != nil {
				return err
			}
		} else {
			eventLeft = append(eventLeft, event)
		}
	}
	wf.AsCodeEvent = eventLeft
	event.PublishWorkflowUpdate(proj.Key, *wf, *wf, u)
	return nil
}

// UpdateWorkflowAsCodeResult pulls repositories operation and the create pullrequest + update workflow
func UpdateWorkflowAsCodeResult(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, ope *sdk.Operation, wf *sdk.Workflow, u *sdk.User) {
	counter := 0
	defer func() {
		store.SetWithTTL(cache.Key(CacheOperationKey, ope.UUID), ope, 300)
	}()
	for {
		counter++
		if err := GetRepositoryOperation(ctx, db, ope); err != nil {
			log.Error("unable to get repository operation %s: %v", ope.UUID, err)
			continue
		}
		if ope.Status == sdk.OperationStatusError {
			log.Error("operation in error %s: %s", ope.UUID, ope.Error)
			break
		}
		if ope.Status == sdk.OperationStatusDone {
			app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
			vcsServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
			if vcsServer == nil {
				log.Error("postWorkflowAsCodeHandler> No vcsServer found")
				ope.Status = sdk.OperationStatusError
				ope.Error = "No vcsServer found"
				return
			}
			client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, vcsServer)
			if errclient != nil {
				log.Error("postWorkflowAsCodeHandler> unable to create repositories manager client: %v", errclient)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to create repositories manager client"
				return
			}

			request := sdk.VCSPullRequest{
				Title: ope.Setup.Push.Message,
				Head: sdk.VCSPushEvent{
					Branch: sdk.VCSBranch{
						DisplayID: ope.Setup.Push.FromBranch,
					},
					Repo: app.RepositoryFullname,
				},
				Base: sdk.VCSPushEvent{
					Branch: sdk.VCSBranch{
						DisplayID: ope.Setup.Push.ToBranch,
					},
					Repo: app.RepositoryFullname,
				},
			}
			pr, err := client.PullRequestCreate(ctx, app.RepositoryFullname, request)
			if err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to create pull request: %v", err)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to create pull request"
				return
			}
			ope.Setup.Push.PRLink = pr.URL

			asCodeEvent := sdk.AsCodeEvent{
				PullRequestID:  int64(pr.ID),
				WorkflowID:     wf.ID,
				PullRequestURL: pr.URL,
				Username:       u.Username,
				CreationDate:   time.Now(),
			}

			// add repo webhook
			found := false
			for _, h := range wf.WorkflowData.GetHooks() {
				if h.HookModelName == sdk.RepositoryWebHookModelName {
					found = true
					break
				}
			}

			if !found {
				h := sdk.WorkflowNodeHook{
					Config:            sdk.RepositoryWebHookModel.DefaultConfig,
					WorkflowHookModel: sdk.RepositoryWebHookModel,
				}
				wf.Root.Hooks = append(wf.Root.Hooks, h)
			}

			// Update the workflow
			oldW, err := LoadByID(db, store, p, wf.ID, u, LoadOptions{})
			if err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to load workflow %s: %v", wf.Name, err)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to load workflow"
				return
			}

			tx, err := db.Begin()
			if err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to start transaction: %v", err)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to start transaction"
				return
			}
			defer tx.Rollback() // nolint

			if err := insertAsCodeEvent(tx, asCodeEvent); err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to insert as code event: %v", err)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to insert as code event"
				return
			}

			if err := Update(ctx, tx, store, wf, oldW, p, u); err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to update workflow: %v", err)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to update workflow"
				return
			}

			if errHr := HookRegistration(ctx, tx, store, oldW, *wf, p); errHr != nil {
				log.Error("postWorkflowAsCodeHandler> unable to update hook registration: %v", errHr)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to update hook"
				return
			}

			if err := tx.Commit(); err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to commit workflow: %v", err)
				ope.Status = sdk.OperationStatusError
				ope.Error = "unable to commit transaction"
				return
			}
			return
		}

		if counter == 30 {
			ope.Status = sdk.OperationStatusError
			ope.Error = "Unable to enable workflow as code"
			break
		}
		time.Sleep(2 * time.Second)
	}
}
