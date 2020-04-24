package ascode

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type SyncResult struct {
	FromRepository string
	MergedWorkflow []int64
}

// SyncEvents checks if workflow as to become ascode.
func SyncEvents(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, app sdk.Application, u sdk.Identifiable) (SyncResult, error) {
	var res SyncResult

	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	if vcsServer == nil {
		return res, sdk.NewErrorFrom(sdk.ErrNotFound, "no vcs server found on application %s", app.Name)
	}
	client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if err != nil {
		return res, err
	}

	fromRepo := app.FromRepository
	if fromRepo == "" {
		repo, err := client.RepoByFullname(ctx, app.RepositoryFullname)
		if err != nil {
			return res, sdk.WrapError(err, "cannot get repo %s", app.RepositoryFullname)
		}
		if app.RepositoryStrategy.ConnectionType == "ssh" {
			fromRepo = repo.SSHCloneURL
		} else {
			fromRepo = repo.HTTPCloneURL
		}
	}
	res.FromRepository = fromRepo

	tx, err := db.Begin()
	if err != nil {
		return res, sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	asCodeEvents, err := LoadAsCodeEventByRepo(ctx, tx, fromRepo)
	if err != nil {
		return res, err
	}

	eventLeft := make([]sdk.AsCodeEvent, 0)
	eventToDelete := make([]sdk.AsCodeEvent, 0)
	for _, ascodeEvt := range asCodeEvents {
		pr, err := client.PullRequest(ctx, app.RepositoryFullname, int(ascodeEvt.PullRequestID))
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return res, sdk.WrapError(err, "unable to check pull request status")
		}
		prNotFound := sdk.ErrorIs(err, sdk.ErrNotFound)

		if prNotFound {
			log.Debug("Pull request %s #%d not found", app.RepositoryFullname, int(ascodeEvt.PullRequestID))
		}

		// If the PR was merged we want to set the repo url on the workflow
		if ascodeEvt.Migrate && len(ascodeEvt.Data.Workflows) == 1 {
			for id := range ascodeEvt.Data.Workflows {
				if pr.Merged {
					res.MergedWorkflow = append(res.MergedWorkflow, id)
				}
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
		if err := DeleteAsCodeEvent(tx, ascodeEvt); err != nil {
			return res, err
		}
	}

	if err := tx.Commit(); err != nil {
		return res, sdk.WithStack(err)
	}

	for _, ed := range eventToDelete {
		event.PublishAsCodeEvent(ctx, proj.Key, ed, u)
	}

	return res, nil
}
