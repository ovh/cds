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

// EventType type for as code events.
type EventType string

// AsCodeEventType values.
const (
	PipelineEvent EventType = "pipeline"
	WorkflowEvent EventType = "workflow"
)

type EntityData struct {
	Type      EventType
	ID        int64
	Name      string
	FromRepo  string
	Operation *sdk.Operation
}

// UpdateAsCodeResult pulls repositories operation and the create pullrequest + update workflow
func UpdateAsCodeResult(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, app sdk.Application, ed EntityData, u sdk.Identifiable) *sdk.AsCodeEvent {
	tick := time.NewTicker(2 * time.Second)
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer func() {
		cancel()
		ed.Operation.RepositoryStrategy.SSHKeyContent = ""
		_ = store.SetWithTTL(cache.Key(operation.CacheOperationKey, ed.Operation.UUID), ed.Operation, 300)
	}()
forLoop:
	for {
		select {
		case <-ctx.Done():
			ed.Operation.Status = sdk.OperationStatusError
			ed.Operation.Error = "Unable to enable workflow as code"
			return nil
		case <-tick.C:
			if err := operation.GetRepositoryOperation(ctx, db, ed.Operation); err != nil {
				log.Error(ctx, "unable to get repository operation %s: %v", ed.Operation.UUID, err)
				continue
			}

			if ed.Operation.Status == sdk.OperationStatusError {
				log.Error(ctx, "operation in error %s: %s", ed.Operation.UUID, ed.Operation.Error)
				break forLoop
			}
			if ed.Operation.Status == sdk.OperationStatusDone {
				vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
				if vcsServer == nil {
					log.Error(ctx, "postWorkflowAsCodeHandler> No vcsServer found")
					ed.Operation.Status = sdk.OperationStatusError
					ed.Operation.Error = "No vcsServer found"
					return nil
				}
				client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
				if errclient != nil {
					log.Error(ctx, "postWorkflowAsCodeHandler> unable to create repositories manager client: %v", errclient)
					ed.Operation.Status = sdk.OperationStatusError
					ed.Operation.Error = "unable to create repositories manager client"
					return nil
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

				// Try to reuse a PR for the branche if exists else create a new one
				var pr *sdk.VCSPullRequest
				prs, err := client.PullRequests(ctx, app.RepositoryFullname, sdk.VCSRequestModifierWithState(sdk.VCSPullRequestStateOpen))
				if err != nil {
					log.Error(ctx, "postWorkflowAsCodeHandler> unable to list pull request: %v", err)
					ed.Operation.Status = sdk.OperationStatusError
					ed.Operation.Error = "unable to list pull request"
					return nil
				}
				for _, prItem := range prs {
					if prItem.Base.Branch.DisplayID == ed.Operation.Setup.Push.ToBranch && prItem.Head.Branch.DisplayID == ed.Operation.Setup.Push.FromBranch {
						pr = &prItem
						break
					}
				}
				if pr == nil {
					newPR, err := client.PullRequestCreate(ctx, app.RepositoryFullname, request)
					if err != nil {
						log.Error(ctx, "postWorkflowAsCodeHandler> unable to create pull request: %v", err)
						ed.Operation.Status = sdk.OperationStatusError
						ed.Operation.Error = "unable to create pull request"
						return nil
					}
					pr = &newPR
				}

				ed.Operation.Setup.Push.PRLink = pr.URL

				// Find existing ascode event with this pullrequest
				asCodeEvent, err := LoadAsCodeByPRID(ctx, db, int64(pr.ID))
				if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
					log.Error(ctx, "UpdateAsCodeResult> unable to save pull request: %v", err)
					ed.Operation.Status = sdk.OperationStatusError
					ed.Operation.Error = "unable to load pull request"
					return nil
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
				case WorkflowEvent:
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
				case PipelineEvent:
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
				if err := InsertOrUpdateAsCodeEvent(db, &asCodeEvent); err != nil {
					log.Error(ctx, "postWorkflowAsCodeHandler> unable to insert as code event: %v", err)
					ed.Operation.Status = sdk.OperationStatusError
					ed.Operation.Error = "unable to insert as code event"
					return nil
				}
				return &asCodeEvent
			}
		}
	}
	return nil
}
