package ascode

import (
	"context"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type SyncResult struct {
	FromRepository string
	Merged         bool
}

// SyncEvents checks if workflow as to become ascode.
func SyncEvents(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, workflowHolder sdk.Workflow, u sdk.Identifiable) (SyncResult, error) {
	var res SyncResult

	if workflowHolder.WorkflowData.Node.Context.ApplicationID == 0 {
		return res, sdk.NewErrorFrom(sdk.ErrWrongRequest, "no application found on the root node of the workflow")
	}
	rootApp := workflowHolder.Applications[workflowHolder.WorkflowData.Node.Context.ApplicationID]

	tx, err := db.Begin()
	if err != nil {
		return res, sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, tx, proj.Key, rootApp.VCSServer)
	if err != nil {
		return res, err
	}
	client, err := repositoriesmanager.AuthorizedClient(ctx, tx, store, proj.Key, vcsServer)
	if err != nil {
		return res, err
	}

	fromRepo := rootApp.FromRepository
	if fromRepo == "" {
		repo, err := client.RepoByFullname(ctx, rootApp.RepositoryFullname)
		if err != nil {
			return res, sdk.WrapError(err, "cannot get repo %s", rootApp.RepositoryFullname)
		}
		if rootApp.RepositoryStrategy.ConnectionType == "ssh" {
			fromRepo = repo.SSHCloneURL
		} else {
			fromRepo = repo.HTTPCloneURL
		}
	}
	res.FromRepository = fromRepo

	asCodeEvents, err := LoadEventsByWorkflowID(ctx, tx, workflowHolder.ID)
	if err != nil {
		return res, err
	}

	eventLeft := make([]sdk.AsCodeEvent, 0)
	eventToDelete := make([]sdk.AsCodeEvent, 0)
	for _, ascodeEvt := range asCodeEvents {
		pr, err := client.PullRequest(ctx, rootApp.RepositoryFullname, strconv.Itoa(int(ascodeEvt.PullRequestID)))
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return res, sdk.WrapError(err, "unable to check pull request status")
		}
		prNotFound := sdk.ErrorIs(err, sdk.ErrNotFound)

		if prNotFound {
			log.Debug("Pull request %s #%d not found", rootApp.RepositoryFullname, int(ascodeEvt.PullRequestID))
		}

		// If the PR was merged we want to set the repo url on the workflow
		if ascodeEvt.Migrate && len(ascodeEvt.Data.Workflows) == 1 {
			if pr.Merged {
				res.Merged = true
			}
		}

		// If event ended, delete it from db
		if prNotFound || pr.Merged || pr.Closed {
			eventToDelete = append(eventToDelete, ascodeEvt)
		} else {
			eventLeft = append(eventLeft, ascodeEvt)
		}
	}

	for _, ascodeEvt := range eventToDelete {
		if err := deleteEvent(tx, &ascodeEvt); err != nil {
			return res, err
		}
	}

	if err := tx.Commit(); err != nil {
		return res, sdk.WithStack(err)
	}

	for _, ed := range eventToDelete {
		event.PublishAsCodeEvent(ctx, proj.Key, workflowHolder.Name, ed, u)
	}

	return res, nil
}
