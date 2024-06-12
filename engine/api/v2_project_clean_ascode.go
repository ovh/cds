package api

import (
	"context"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

type EntitiesCleaner struct {
	projKey   string
	vcsName   string
	repoName  string
	refs      map[string]string
	retention time.Duration
}

func (a *API) cleanProjectEntities(ctx context.Context, delay time.Duration, entityRetention time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-ticker.C:
			projects, err := project.LoadAll(ctx, a.mustDB(), a.Cache)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			inputChan := make(chan string, len(projects))
			resultChan := make(chan bool)
			for w := 0; w < 10; w++ {
				a.GoRoutines.Exec(ctx, "cleanProjectEntities-"+strconv.Itoa(w), func(ctx context.Context) {
					for pKey := range inputChan {
						if err := workerCleanProject(ctx, a.mustDB(), a.Cache, pKey, entityRetention); err != nil {
							log.ErrorWithStackTrace(ctx, err)
						}
						resultChan <- true
					}
				})
			}
			for _, p := range projects {
				inputChan <- p.Key
			}
			close(inputChan)
			for r := 0; r < len(projects); r++ {
				<-resultChan
			}
		}
	}
}

func workerCleanProject(ctx context.Context, db *gorp.DbMap, store cache.Store, pKey string, entityRetention time.Duration) error {
	ctx = context.WithValue(ctx, cdslog.Action, "workerCleanProject")
	ctx = context.WithValue(ctx, "action_metadata_project_key", pKey)
	log.Info(ctx, "Clean ascode entities on project %s", pKey)
	lockKey := cache.Key("ascode", "clean", pKey)
	locked, err := store.Lock(lockKey, 5*time.Minute, 500, 1)
	if err != nil {
		return err
	}
	if !locked {
		return nil
	}
	defer store.Unlock(lockKey)
	if err := cleanAscodeProject(ctx, db, store, pKey, entityRetention); err != nil {
		return err
	}
	return nil
}

func cleanAscodeProject(ctx context.Context, db *gorp.DbMap, store cache.Store, pKey string, entityRetention time.Duration) error {
	vcsRepos, err := vcs.LoadAllVCSByProject(ctx, db, pKey)
	if err != nil {
		return err
	}
	for _, vcsServer := range vcsRepos {
		ctx = context.WithValue(ctx, cdslog.VCSServer, vcsServer.Name)
		repos, err := repository.LoadAllRepositoriesByVCSProjectID(ctx, db, vcsServer.ID)
		if err != nil {
			return err
		}

		for _, r := range repos {
			ctx = context.WithValue(ctx, cdslog.Repository, r.Name)
			entities, err := entity.LoadByRepository(ctx, db, r.ID)
			if err != nil {
				return err
			}

			// Sort by ref
			entitiesByRef := make(map[string][]sdk.Entity)
			for _, e := range entities {
				ents, has := entitiesByRef[e.Ref]
				if !has {
					ents = make([]sdk.Entity, 0, 1)
				}
				ents = append(ents, e)
				entitiesByRef[e.Ref] = ents
			}

			cleaner := &EntitiesCleaner{
				projKey:   pKey,
				vcsName:   vcsServer.Name,
				repoName:  r.Name,
				refs:      make(map[string]string),
				retention: entityRetention,
			}
			if err := cleaner.getBranches(ctx, db, store); err != nil {
				return err
			}

			for branchName, branchEntities := range entitiesByRef {
				// Clean entities that exists on deleted branches
				if currentHEAD, has := cleaner.refs[branchName]; has {
					// Clean non head commits on existing branch
					if err := cleaner.cleanNonHeadEntities(ctx, db, store, branchName, currentHEAD, branchEntities); err != nil {
						return err
					}
				} else {
					if err := cleaner.cleanEntitiesByDeletedRef(ctx, db, store, branchName, branchEntities); err != nil {
						return err
					}
				}

			}
		}
	}
	return nil
}

func (c *EntitiesCleaner) getBranches(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, store, c.projKey, c.vcsName)
	if err != nil {
		return err
	}

	branches, err := vcsClient.Branches(ctx, c.repoName, sdk.VCSBranchesFilter{Limit: 50})
	if err != nil {
		return err
	}

	c.refs = make(map[string]string)
	for _, b := range branches {
		c.refs[b.ID] = b.LatestCommit
	}

	tags, err := vcsClient.Tags(ctx, c.repoName)
	if err != nil {
		return err
	}
	for _, t := range tags {
		c.refs[sdk.GitRefTagPrefix+t.Tag] = t.Hash
	}
	return nil
}

func (c *EntitiesCleaner) cleanNonHeadEntities(ctx context.Context, db *gorp.DbMap, store cache.Store, ref string, refHeadCommit string, entitiesByBranch []sdk.Entity) error {
	deletedEntities := make([]sdk.Entity, 0)

	if len(entitiesByBranch) > 0 {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		log.Info(ctx, "Deleting non head entities on %s / %s / %s @%s", c.projKey, c.vcsName, c.repoName, ref)
		for _, e := range entitiesByBranch {
			if e.Commit != "HEAD" && e.Commit != refHeadCommit && time.Since(e.LastUpdate) > c.retention {
				if err := entity.Delete(ctx, tx, &e); err != nil {
					return err
				}
				log.Info(ctx, "entity %s of type %s deleted", e.Name, e.Type)
				deletedEntities = append(deletedEntities, e)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(tx.Commit())
		}
	}

	for _, e := range deletedEntities {
		event_v2.PublishEntityEvent(ctx, store, sdk.EventEntityDeleted, c.vcsName, c.repoName, e, nil)
	}
	return nil
}

func (c *EntitiesCleaner) cleanEntitiesByDeletedRef(ctx context.Context, db *gorp.DbMap, store cache.Store, ref string, entitiesByBranch []sdk.Entity) error {
	deletedEntities := make([]sdk.Entity, 0)

	if len(entitiesByBranch) > 0 {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		log.Info(ctx, "Deleting entities on old branches: %s / %s / %s @%s", c.projKey, c.vcsName, c.repoName, ref)
		for _, e := range entitiesByBranch {
			if err := entity.Delete(ctx, tx, &e); err != nil {
				return err
			}
			log.Info(ctx, "entity %s of type %s deleted", e.Name, e.Type)
			deletedEntities = append(deletedEntities, e)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(tx.Commit())
		}
	}

	for _, e := range deletedEntities {
		event_v2.PublishEntityEvent(ctx, store, sdk.EventEntityDeleted, c.vcsName, c.repoName, e, nil)
	}
	return nil
}
