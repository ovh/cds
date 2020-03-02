package sync

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SyncAsCodeEvent checks if workflow as to become ascode
func SyncAsCodeEvent(ctx context.Context, db *gorp.DbMap, store cache.Store, proj sdk.Project, app *sdk.Application, u sdk.Identifiable) ([]sdk.AsCodeEvent, string, error) {
	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	if vcsServer == nil {
		return nil, "", sdk.NewErrorFrom(sdk.ErrNotFound, "no vcsserver found on application %s", app.Name)
	}
	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if errclient != nil {
		return nil, "", errclient
	}

	fromRepo := app.FromRepository
	if fromRepo == "" {
		repo, errR := client.RepoByFullname(ctx, app.RepositoryFullname)
		if errR != nil {
			return nil, fromRepo, sdk.WrapError(errR, "cannot get repo %s", app.RepositoryFullname)
		}
		if app.RepositoryStrategy.ConnectionType == "ssh" {
			fromRepo = repo.SSHCloneURL
		} else {
			fromRepo = repo.HTTPCloneURL
		}
	}

	asCodeEvents, err := ascode.LoadAsCodeEventByRepo(ctx, db, fromRepo)
	if err != nil {
		return nil, fromRepo, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fromRepo, sdk.WrapError(err, "unable to start transaction")
	}
	defer tx.Rollback() //nolint

	eventLeft := make([]sdk.AsCodeEvent, 0)
	eventDeleted := make([]sdk.AsCodeEvent, 0)
	for _, ascodeEvt := range asCodeEvents {

		merged, closed, err := CheckPullRequestStatus(ctx, client, app.RepositoryFullname, ascodeEvt.PullRequestID)
		if err != nil {
			return nil, fromRepo, err
		}
		// If event ended, delete it from db
		if merged || closed {
			if err := ascode.DeleteAsCodeEvent(tx, ascodeEvt); err != nil {
				return nil, fromRepo, err
			}
			eventDeleted = append(eventDeleted, ascodeEvt)
			if ascodeEvt.Migrate && len(ascodeEvt.Data.Workflows) == 1 {
				for id := range ascodeEvt.Data.Workflows {
					if err := workflow.UpdateFromRepository(db, id, fromRepo); err != nil {
						return nil, "", err
					}
				}
			}
		} else {
			eventLeft = append(eventLeft, ascodeEvt)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fromRepo, sdk.WrapError(err, "unable to commit transaction")
	}

	for _, ed := range eventDeleted {
		event.PublishAsCodeEvent(ctx, proj.Key, ed, u)
	}
	return eventLeft, fromRepo, nil
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
