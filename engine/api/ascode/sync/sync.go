package sync

import (
	"context"

	"github.com/ovh/cds/engine/api/workflow"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SyncAsCodeEvent checks if workflow as to become ascode
func SyncAsCodeEvent(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, app sdk.Application, u sdk.Identifiable) ([]sdk.AsCodeEvent, string, error) {
	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	if vcsServer == nil {
		return nil, "", sdk.NewErrorFrom(sdk.ErrNotFound, "no vcs server found on application %s", app.Name)
	}
	client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if err != nil {
		return nil, "", err
	}

	fromRepo := app.FromRepository
	if fromRepo == "" {
		repo, err := client.RepoByFullname(ctx, app.RepositoryFullname)
		if err != nil {
			return nil, fromRepo, sdk.WrapError(err, "cannot get repo %s", app.RepositoryFullname)
		}
		if app.RepositoryStrategy.ConnectionType == "ssh" {
			fromRepo = repo.SSHCloneURL
		} else {
			fromRepo = repo.HTTPCloneURL
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fromRepo, sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	asCodeEvents, err := ascode.LoadAsCodeEventByRepo(ctx, tx, fromRepo)
	if err != nil {
		return nil, fromRepo, err
	}

	eventLeft := make([]sdk.AsCodeEvent, 0)
	eventToDelete := make([]sdk.AsCodeEvent, 0)
	for _, ascodeEvt := range asCodeEvents {
		pr, err := client.PullRequest(ctx, app.RepositoryFullname, int(ascodeEvt.PullRequestID))
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, fromRepo, sdk.WrapError(err, "unable to check pull request status")
		}
		prNotFound := sdk.ErrorIs(err, sdk.ErrNotFound)

		if prNotFound {
			log.Debug("Pull request %s #%d not found", app.RepositoryFullname, int(ascodeEvt.PullRequestID))
		}

		// If event ended, delete it from db
		if prNotFound || pr.Merged || pr.Closed {
			eventToDelete = append(eventToDelete, ascodeEvt)
		} else {
			eventLeft = append(eventLeft, ascodeEvt)
		}
	}

	for _, ascodeEvt := range eventToDelete {
		if err := ascode.DeleteAsCodeEvent(tx, ascodeEvt); err != nil {
			return nil, fromRepo, err
		}
		if ascodeEvt.Migrate && len(ascodeEvt.Data.Workflows) == 1 {
			for id := range ascodeEvt.Data.Workflows {
				if err := workflow.UpdateFromRepository(tx, id, fromRepo); err != nil {
					return nil, fromRepo, err
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fromRepo, sdk.WithStack(err)
	}

	for _, ed := range eventToDelete {
		event.PublishAsCodeEvent(ctx, proj.Key, ed, u)
	}

	return eventLeft, fromRepo, nil
}
