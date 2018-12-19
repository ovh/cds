package workflow

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

var cacheOperationKey = cache.Key("repositories", "operation", "push")

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

	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, vcsServer)
	if errclient != nil {
		return nil, sdk.WrapError(errclient, "Cannot get client")
	}

	repo, errR := client.RepoByFullname(ctx, app.RepositoryFullname)
	if errR != nil {
		return nil, sdk.WrapError(errR, "cannot get repo %s", app.RepositoryFullname)
	}

	// Export workflow
	buf := new(bytes.Buffer)
	if err := Pull(ctx, db, store, proj, wf.Name, exportentities.FormatYAML, encryptFunc, u, buf, exportentities.WorkflowSkipIfOnlyOneRepoWebhook); err != nil {
		return nil, sdk.WrapError(err, "cannot pull workflow")
	}

	// Create VCS Operation
	ope := sdk.Operation{
		VCSServer:          app.VCSServer,
		RepoFullName:       app.RepositoryFullname,
		URL:                "",
		RepositoryStrategy: app.RepositoryStrategy,
		Setup: sdk.OperationSetup{
			Push: sdk.OperationPush{
				FromBranch: fmt.Sprintf("cdsAsCode-%d", time.Now().Unix()),
				Message:    "",
			},
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
		ope.Setup.Push.Message = fmt.Sprintf("fix: Update workflow [@%s]", u.Username)
	}

	if err := PostRepositoryOperation(ctx, db, store, *proj, &ope, buf); err != nil {
		return nil, sdk.WrapError(err, "unable to post repository operation")
	}

	store.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)

	return &ope, nil
}

// UpdateWorkflowAsCodeResult pulls repositories operation and the create pullrequest + update workflow
func UpdateWorkflowAsCodeResult(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, ope *sdk.Operation, wf *sdk.Workflow, u *sdk.User) {
	counter := 0
	for {
		counter++
		if err := GetRepositoryOperation(ctx, db, ope); err != nil {
			log.Error("unable to get repository operation %s: %v", ope.UUID, err)
			continue
		}
		if ope.Status == sdk.OperationStatusError || ope.Status == sdk.OperationStatusDone {
			store.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)
		}

		if ope.Status == sdk.OperationStatusDone {
			app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
			vcsServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
			if vcsServer == nil {
				log.Error("postWorkflowAsCodeHandler> No vcsServer found")
				return
			}
			client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, vcsServer)
			if errclient != nil {
				log.Error("postWorkflowAsCodeHandler> unable to create repositories manager client: %v", errclient)
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
				log.Error("postWorkflowAsCodeHandler> unable to create pull request")
				return
			}
			ope.Setup.Push.PRLink = pr.URL
			store.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)
			wf.FromRepository = ope.URL

			// add repo webhook
			found := false
			for _, h := range wf.WorkflowData.GetHooks() {
				if h.HookModelName == sdk.RepositoryWebHookModelName {
					found = true
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
			oldW, err := LoadByID(db, store, p, wf.ID, u, LoadOptions{WithIcon: true})
			if err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to load workflow %s: %v", wf.Name, err)
				return
			}

			tx, err := db.Begin()
			if err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to start transaction")
				return
			}
			defer tx.Rollback()
			if err := Update(ctx, tx, store, wf, oldW, p, u); err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to update workflow")
				return
			}

			if errHr := HookRegistration(ctx, tx, store, oldW, *wf, p); errHr != nil {
				log.Error("postWorkflowAsCodeHandler> unable to update hook registration")
				return
			}

			if err := tx.Commit(); err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to commit workflow")
				return
			}
			store.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)

			log.Info("Migration as code finished.")
			return
		}

		if counter == 30 {
			ope.Status = sdk.OperationStatusError
			ope.Error = "Unable to enable workflow as code"
			store.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)
			break
		}
		time.Sleep(2 * time.Second)
	}
}
