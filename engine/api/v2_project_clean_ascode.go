package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
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

func (a *API) cleanWorkflowVersion(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.Config.WorkflowV2.VersionRetentionScheduling) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-ticker.C:
			workflows, err := workflow_v2.LoadDistinctWorkflowVersionByWorkflow(ctx, a.mustDB())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			inputChan := make(chan workflow_v2.V2WorkflowVersionWorkflowShort, len(workflows))
			resultChan := make(chan bool)
			for w := 0; w < 10; w++ {
				a.GoRoutines.Exec(ctx, "cleanWorkflowVersion-"+strconv.Itoa(w), func(ctx context.Context) {
					for w := range inputChan {
						if err := workerCleanWorkflowVersion(ctx, a.mustDB(), a.Cache, w, a.Config.WorkflowV2.VersionRetention); err != nil {
							log.ErrorWithStackTrace(ctx, err)
						}
						resultChan <- true
					}
				})
			}
			for _, w := range workflows {
				inputChan <- w
			}
			close(inputChan)
			for r := 0; r < len(workflows); r++ {
				<-resultChan
			}
		}
	}
}

func workerCleanWorkflowVersion(ctx context.Context, db *gorp.DbMap, store cache.Store, w workflow_v2.V2WorkflowVersionWorkflowShort, nbVersionToKeep int64) error {
	ctx = context.WithValue(ctx, cdslog.Action, "workerCleanWorkflowVersion")
	ctx = context.WithValue(ctx, "action_metadata_project_key", w.ProjectKey)
	ctx = context.WithValue(ctx, cdslog.VCSServer, w.WorkflowVCS)
	ctx = context.WithValue(ctx, cdslog.Repository, w.WorkflowRepository)
	ctx = context.WithValue(ctx, cdslog.Workflow, w.WorkflowName)

	log.Info(ctx, "Clean workflow version for "+w.String())
	lockKey := cache.Key("workflow", "version", w.String())
	locked, err := store.Lock(lockKey, 5*time.Minute, 500, 1)
	if err != nil {
		return err
	}
	if !locked {
		return nil
	}
	defer store.Unlock(lockKey)
	if err := cleanWorkflowVersion(ctx, db, store, w, nbVersionToKeep); err != nil {
		return err
	}
	return nil
}

func cleanWorkflowVersion(ctx context.Context, db *gorp.DbMap, store cache.Store, w workflow_v2.V2WorkflowVersionWorkflowShort, nbVersionToKeep int64) error {
	allVersions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, db, w.ProjectKey, w.WorkflowVCS, w.WorkflowRepository, w.WorkflowName)
	if err != nil {
		return err
	}
	if len(allVersions) < int(nbVersionToKeep) {
		return nil
	}

	versions := make([]*semver.Version, 0, len(allVersions))
	for _, v := range allVersions {
		sVer, _ := semver.NewVersion(v.Version)
		versions = append(versions, sVer)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LessThan(versions[j])
	})

	var versionsToClean []*semver.Version
	cleanAll := false
	// Check if the workflow still exists on default branch
	vcsProject, err := vcs.LoadVCSByProject(ctx, db, w.ProjectKey, w.WorkflowVCS)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}
	// If vcs doesn't exist, cleann all
	if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
		log.Info(ctx, "vcs %s doesn't exist anymore. Cleaning all versions", w.WorkflowVCS)
		cleanAll = true
	}

	if !cleanAll {
		repository, err := repository.LoadRepositoryByName(ctx, db, vcsProject.ID, w.WorkflowRepository)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
			log.Info(ctx, "repository %s doesn't exist anymore. Cleaning all versions", w.WorkflowRepository)
			cleanAll = true
		}

		if !cleanAll {
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, store, w.ProjectKey, w.WorkflowVCS)
			if err != nil {
				return err
			}
			defaultBranch, err := vcsClient.Branch(ctx, w.WorkflowRepository, sdk.VCSBranchFilters{Default: true})
			if err != nil {
				return err
			}
			_, err = entity.LoadByRefTypeNameCommit(ctx, db, repository.ID, defaultBranch.ID, sdk.EntityTypeWorkflow, w.WorkflowName, "HEAD")
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
				log.Info(ctx, "workflow %s doesn't exist anymore. Cleaning all versions", w.WorkflowName)
				cleanAll = true
			}
		}
	}

	if cleanAll {
		versionsToClean = versions
	} else {
		versionsToClean = versions[0 : len(versions)-int(nbVersionToKeep)]
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()
	for _, v := range versionsToClean {
		wkfVersion, err := workflow_v2.LoadWorkflowVersion(ctx, tx, w.ProjectKey, w.WorkflowVCS, w.WorkflowRepository, w.WorkflowName, v.String())
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		if err := workflow_v2.DeleteWorkflowVersion(ctx, tx, wkfVersion); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		log.Info(ctx, "version %s deleted for %s", v.String(), w.String())
	}
	return sdk.WithStack(tx.Commit())
}

func (a *API) cleanProjectEntities(ctx context.Context, entityRetention time.Duration) {
	ticker := time.NewTicker(time.Duration(a.Config.Entity.RoutineDelay) * time.Minute)
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
	hookServices, err := services.LoadAllByType(ctx, db, sdk.TypeHooks)
	if err != nil {
		return err
	}
	if len(hookServices) < 1 {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to find hook service")
	}

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
					if err := cleaner.cleanNonHeadEntities(ctx, db, store, branchName, currentHEAD, branchEntities, hookServices); err != nil {
						return err
					}
				} else {
					if err := cleaner.cleanEntitiesByDeletedRef(ctx, db, store, branchName, branchEntities, hookServices); err != nil {
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

	branches, err := vcsClient.Branches(ctx, c.repoName, sdk.VCSBranchesFilter{Limit: 100, NoCache: true})
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

func (c *EntitiesCleaner) cleanNonHeadEntities(ctx context.Context, db *gorp.DbMap, store cache.Store, ref string, refHeadCommit string, entitiesByBranch []sdk.Entity, hookServices []sdk.Service) error {
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
				if err := DeleteEntity(ctx, tx, &e, hookServices, DeleteEntityOps{WithHooks: false}); err != nil {
					return err
				}
				log.Info(ctx, "entity %s of type %s deleted on branch %s for commit %s", e.Name, e.Type, e.Ref, e.Commit)
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

func (c *EntitiesCleaner) cleanEntitiesByDeletedRef(ctx context.Context, db *gorp.DbMap, store cache.Store, ref string, entitiesByBranch []sdk.Entity, hookServices []sdk.Service) error {
	deletedEntities := make([]sdk.Entity, 0)

	if len(entitiesByBranch) > 0 {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		log.Info(ctx, "Deleting entities on old branches: %s / %s / %s @%s", c.projKey, c.vcsName, c.repoName, ref)
		for _, e := range entitiesByBranch {
			if err := DeleteEntity(ctx, tx, &e, hookServices, DeleteEntityOps{WithHooks: false}); err != nil {
				return err
			}
			log.Info(ctx, "entity %s of type %s deleted on branch %s for commit %s", e.Name, e.Type, e.Ref, e.Commit)
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

type DeleteEntityOps struct {
	WithHooks bool
}

func DeleteEntity(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, e *sdk.Entity, srvs []sdk.Service, opts DeleteEntityOps) error {
	if e.Type == sdk.EntityTypeWorkflow && opts.WithHooks {
		whooks, err := workflow_v2.LoadHooksByEntityID(ctx, tx, e.ID)
		if err != nil {
			return err
		}
		for _, h := range whooks {
			if h.Type != sdk.WorkflowHookTypeScheduler {
				continue
			}
			if err := DeleteAllEntitySchedulerHook(ctx, tx, h.VCSName, h.RepositoryName, h.WorkflowName, srvs); err != nil {
				return err
			}
			break
		}
	}

	if err := entity.Delete(ctx, tx, e); err != nil {
		return err
	}

	return nil
}

func DeleteAllEntitySchedulerHook(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, vcs, repo, workflow string, srvs []sdk.Service) error {
	path := fmt.Sprintf("/v2/workflow/scheduler/%s/%s/%s", vcs, url.PathEscape(repo), workflow)
	if _, _, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return err
	}
	return nil
}
