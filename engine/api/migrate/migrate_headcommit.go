package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func MigrateHeadCommit(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	// Migrate entity with commit HEAD
	entities, err := entity.LoadUnmigratedHeadEntities(ctx, db)
	if err != nil {
		return err
	}

	for _, e := range entities {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		e.Head = true
		if err := entity.Update(ctx, tx, &e); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return sdk.WithStack(err)
		}
	}

	// Migrate non repositorywebhook hooks
	hooks, err := workflow_v2.LoadHeadHookToMigrate(ctx, db)
	if err != nil {
		return err
	}
	repoCache := make(map[string]*sdk.VCSBranch)
	hooksToUpdate := make([]sdk.V2WorkflowHook, 0)
	for _, h := range hooks {
		repoCacheKey := h.VCSName + "/" + h.RepositoryName
		defaultBranch, has := repoCache[repoCacheKey]
		if !has {
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, store, h.ProjectKey, h.VCSName)
			if err != nil {
				return err
			}
			defaultBranch, err = vcsClient.Branch(ctx, h.RepositoryName, sdk.VCSBranchFilters{Default: true})
			if err != nil {
				return err
			}
			repoCache[repoCacheKey] = defaultBranch
		}
		if h.Commit == defaultBranch.LatestCommit {
			hooksToUpdate = append(hooksToUpdate, h)
		}
	}

	for _, h := range hooksToUpdate {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		h.Head = true
		if err := workflow_v2.UpdateWorkflowHook(ctx, tx, &h); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return sdk.WithStack(err)
		}
	}
	return nil
}
