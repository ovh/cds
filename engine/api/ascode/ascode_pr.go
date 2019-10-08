package ascode

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	AsCodePipeline = "pipeline"
	AsCodeWorkflow = "workflow"
)

type EntityData struct {
	Type      string
	ID        int64
	Name      string
	FromRepo  string
	Operation *sdk.Operation
}

// SyncAsCodeEvent checks if workflow as to become ascode
func SyncAsCodeEvent(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, app *sdk.Application) (bool, []sdk.AsCodeEvent, string, error) {
	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	if vcsServer == nil {
		return false, nil, "", sdk.WrapError(sdk.ErrNotFound, "no vcsserver found on application %s", app.Name)
	}
	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if errclient != nil {
		return false, nil, "", errclient
	}

	fromRepo := app.FromRepository
	if fromRepo == "" {
		repo, errR := client.RepoByFullname(ctx, app.RepositoryFullname)
		if errR != nil {
			return false, nil, fromRepo, sdk.WrapError(errR, "cannot get repo %s", app.RepositoryFullname)
		}
		if app.RepositoryStrategy.ConnectionType == "ssh" {
			fromRepo = repo.SSHCloneURL
		} else {
			fromRepo = repo.HTTPCloneURL
		}
	}

	asCodeEvents, err := LoadAsCodeEventByRepo(db, fromRepo)
	if err != nil {
		return false, nil, fromRepo, err
	}

	eventLeft := make([]sdk.AsCodeEvent, 0)
	migrateEnded := false
	for _, ascodeEvt := range asCodeEvents {
		merged, closed, err := CheckPullRequestStatus(ctx, client, app.RepositoryFullname, ascodeEvt.PullRequestID)
		if err != nil {
			return false, nil, fromRepo, err
		}
		// If event ended, delete it from db
		if merged || closed {
			if err := DeleteAsCodeEvent(db, ascodeEvt); err != nil {
				return false, nil, fromRepo, err
			}
			if ascodeEvt.Migrate {
				migrateEnded = true
			}
		} else {
			eventLeft = append(eventLeft, ascodeEvt)
		}
	}
	return migrateEnded, eventLeft, fromRepo, nil
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

// UpdateAsCodeResult pulls repositories operation and the create pullrequest + update workflow
func UpdateAsCodeResult(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, app *sdk.Application, ed EntityData, u sdk.Identifiable) {
	counter := 0
	defer func() {
		ed.Operation.RepositoryStrategy.SSHKeyContent = ""
		_ = store.SetWithTTL(cache.Key(operation.CacheOperationKey, ed.Operation.UUID), ed.Operation, 300)
	}()
	for {
		counter++
		if err := operation.GetRepositoryOperation(ctx, db, ed.Operation); err != nil {
			log.Error("unable to get repository operation %s: %v", ed.Operation.UUID, err)
			continue
		}

		if ed.Operation.Status == sdk.OperationStatusError {
			log.Error("operation in error %s: %s", ed.Operation.UUID, ed.Operation.Error)
			break
		}
		if ed.Operation.Status == sdk.OperationStatusDone {
			vcsServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
			if vcsServer == nil {
				log.Error("postWorkflowAsCodeHandler> No vcsServer found")
				ed.Operation.Status = sdk.OperationStatusError
				ed.Operation.Error = "No vcsServer found"
				return
			}
			client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, p.Key, vcsServer)
			if errclient != nil {
				log.Error("postWorkflowAsCodeHandler> unable to create repositories manager client: %v", errclient)
				ed.Operation.Status = sdk.OperationStatusError
				ed.Operation.Error = "unable to create repositories manager client"
				return
			}

			request := sdk.VCSPullRequest{
				Title: ed.Operation.Setup.Push.Message,
				Head: sdk.VCSPushEvent{
					Branch: sdk.VCSBranch{
						DisplayID: ed.Operation.Setup.Push.FromBranch,
					},
					Repo: app.RepositoryFullname,
				},
				Base: sdk.VCSPushEvent{
					Branch: sdk.VCSBranch{
						DisplayID: ed.Operation.Setup.Push.ToBranch,
					},
					Repo: app.RepositoryFullname,
				},
			}
			pr, err := client.PullRequestCreate(ctx, app.RepositoryFullname, request)
			if err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to create pull request: %v", err)
				ed.Operation.Status = sdk.OperationStatusError
				ed.Operation.Error = "unable to create pull request"
				return
			}
			if pr.URL == "" {
				prs, err := client.PullRequests(ctx, app.RepositoryFullname)
				if err != nil {
					log.Error("postWorkflowAsCodeHandler> unable to list pull request: %v", err)
					ed.Operation.Status = sdk.OperationStatusError
					ed.Operation.Error = "unable to list pull request"
					return
				}
				for _, prItem := range prs {
					if prItem.Base.Branch.DisplayID == ed.Operation.Setup.Push.ToBranch && prItem.Head.Branch.DisplayID == ed.Operation.Setup.Push.FromBranch {
						pr = prItem
						break
					}
				}
			}
			ed.Operation.Setup.Push.PRLink = pr.URL

			// Find existing ascode event with this pullrequest
			asCodeEvent, err := LoadAsCodeByPRID(db, int64(pr.ID))
			if err != nil && err != sdk.ErrNotFound {
				log.Error("UpdateAsCodeResult> unable to save pull request: %v", err)
				return
			}
			if asCodeEvent.ID == 0 {
				asCodeEvent = sdk.AsCodeEvent{
					PullRequestID:  int64(pr.ID),
					PullRequestURL: pr.URL,
					Username:       u.GetUsername(),
					CreateDate:     time.Now(),
					FromRepo:       ed.FromRepo,
					Migrate:        !ed.Operation.Setup.Push.Update,
				}
			}

			switch ed.Type {
			case AsCodeWorkflow:
				if asCodeEvent.Data.Workflows == nil {
					asCodeEvent.Data.Workflows = make(map[int64]string)
				}
				found := false
				for k := range asCodeEvent.Data.Workflows {
					if k == ed.ID {
						found = true
						break
					}
				}
				if !found {
					asCodeEvent.Data.Workflows[ed.ID] = ed.Name
				}
			case AsCodePipeline:
				if asCodeEvent.Data.Pipelines == nil {
					asCodeEvent.Data.Pipelines = make(map[int64]string)
				}
				found := false
				for k := range asCodeEvent.Data.Pipelines {
					if k == ed.ID {
						found = true
						break
					}
				}
				if !found {
					asCodeEvent.Data.Pipelines[ed.ID] = ed.Name
				}
			}
			if err := insertOrUpdateAsCodeEvent(db, &asCodeEvent); err != nil {
				log.Error("postWorkflowAsCodeHandler> unable to insert as code event: %v", err)
				ed.Operation.Status = sdk.OperationStatusError
				ed.Operation.Error = "unable to insert as code event"
				return
			}
			return
		}

		if counter == 30 {
			ed.Operation.Status = sdk.OperationStatusError
			ed.Operation.Error = "Unable to enable workflow as code"
			break
		}
		time.Sleep(2 * time.Second)
	}
}
