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
	projKey  string
	vcsName  string
	repoName string
	branches map[string]struct{}
}

func (a *API) cleanProjectEntities(ctx context.Context, delay time.Duration) {
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
						if err := workerCleanProject(ctx, a.mustDB(), a.Cache, pKey); err != nil {
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

func workerCleanProject(ctx context.Context, db *gorp.DbMap, store cache.Store, pKey string) error {
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
	if err := cleanAscodeProject(ctx, db, store, pKey); err != nil {
		return err
	}
	return nil
}

func cleanAscodeProject(ctx context.Context, db *gorp.DbMap, store cache.Store, pKey string) error {
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

			// Sort by branch
			entitiesByBranch := make(map[string][]sdk.Entity)
			for _, e := range entities {
				ents, has := entitiesByBranch[e.Branch]
				if !has {
					ents = make([]sdk.Entity, 0, 1)
				}
				ents = append(ents, e)
				entitiesByBranch[e.Branch] = ents
			}

			cleaner := &EntitiesCleaner{
				projKey:  pKey,
				vcsName:  vcsServer.Name,
				repoName: r.Name,
				branches: make(map[string]struct{}),
			}
			if err := cleaner.getBranches(ctx, db, store); err != nil {
				return err
			}

			for branchName, branchEntities := range entitiesByBranch {
				if err := cleaner.cleanEntitiesByBranch(ctx, db, store, branchName, branchEntities); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *EntitiesCleaner) getBranches(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()

	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, store, c.projKey, c.vcsName)
	if err != nil {
		return err
	}

	branches, err := vcsClient.Branches(ctx, c.repoName, sdk.VCSBranchesFilter{Limit: 50})
	if err != nil {
		return err
	}

	c.branches = make(map[string]struct{})
	for _, b := range branches {
		c.branches[b.DisplayID] = struct{}{}
	}
	return sdk.WithStack(tx.Commit())
}

func (c *EntitiesCleaner) cleanEntitiesByBranch(ctx context.Context, db *gorp.DbMap, store cache.Store, branchName string, entitiesByBranch []sdk.Entity) error {
	deletedEntities := make([]sdk.Entity, 0)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()

	if _, has := c.branches[branchName]; !has {
		log.Info(ctx, "Deleting entities on  %s / %s / %s @%s", c.projKey, c.vcsName, c.repoName, branchName)
		for _, e := range entitiesByBranch {
			if err := entity.Delete(ctx, tx, &e); err != nil {
				return err
			}
			deletedEntities = append(deletedEntities, e)
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(tx.Commit())
	}

	for _, e := range deletedEntities {
		event_v2.PublishEntityDeleteEvent(ctx, store, c.vcsName, c.repoName, e, nil)
	}
	return nil
}
