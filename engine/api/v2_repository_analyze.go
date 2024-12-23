package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type createAnalysisRequest struct {
	proj          sdk.Project
	vcsProject    sdk.VCSProject
	repo          sdk.ProjectRepository
	ref           string
	commit        string
	hookEventUUID string
	hookEventKey  string
	user          *sdk.AuthentifiedUser
	adminMFA      bool
}

func (api *API) cleanRepositoryAnalysis(ctx context.Context, delay time.Duration) {
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
			repositories, err := repository.LoadAllRepositories(ctx, api.mustDB())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for _, r := range repositories {
				nb, err := repository.CountAnalysesByRepo(api.mustDB(), r.ID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					break
				}
				if nb > 50 {
					toDelete := int(nb - 50)
					tx, err := api.mustDB().Begin()
					if err != nil {
						log.ErrorWithStackTrace(ctx, err)
						break
					}
					for i := 0; i < toDelete; i++ {
						if err := repository.DeleteOldestAnalysis(ctx, tx, r.ID); err != nil {
							log.ErrorWithStackTrace(ctx, err)
							break
						}
					}

					if err := tx.Commit(); err != nil {
						log.ErrorWithStackTrace(ctx, err)
						break
					}
				}
			}
		}
	}
}

func (api *API) getProjectRepositoryAnalysesHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.analysisRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			analyses, err := repository.LoadAnalysesByRepo(ctx, api.mustDB(), repo.ID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, analyses, http.StatusOK)
		}
}

func (api *API) getProjectRepositoryAnalysisHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.analysisRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			analysisID := vars["analysisID"]

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			analysis, err := repository.LoadRepositoryAnalysisById(ctx, api.mustDB(), repo.ID, analysisID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, analysis, http.StatusOK)
		}
}

// postRepositoryAnalysisHandler Trigger repository analysis
func (api *API) postRepositoryAnalysisHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.triggerAnalysis),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			projKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}

			vcs, err := api.getVCSByIdentifier(ctx, projKey, vcsIdentifier)
			if err != nil {
				return err
			}
			repo, err := api.getRepositoryByIdentifier(ctx, vcs.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			var analysis sdk.AnalysisRequest
			if err := service.UnmarshalBody(req, &analysis); err != nil {
				return err
			}

			// check path inputs
			if analysis.ProjectKey != projKey {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid project key: Got %s, want %s", analysis.ProjectKey, projKey)
			}
			if analysis.VcsName != vcs.Name {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid vcs name: Got %s, want %s", analysis.VcsName, vcs.Name)
			}
			if analysis.RepoName != repo.Name {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid repository name: Got %s, want %s", analysis.RepoName, repo.Name)
			}

			proj, err := project.Load(ctx, api.mustDB(), analysis.ProjectKey, project.LoadOptions.WithClearKeys)
			if err != nil {
				return err
			}

			if analysis.Commit == "" || analysis.Ref == "" {
				// retrieve commit for the given ref
				client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, analysis.ProjectKey, vcs.Name)
				if err != nil {
					return err
				}

				if analysis.Ref != "" && analysis.Commit == "" {
					switch {
					case strings.HasPrefix(analysis.Ref, sdk.GitRefTagPrefix):
						giventTag, err := client.Tag(ctx, repo.Name, strings.TrimPrefix(analysis.Ref, sdk.GitRefTagPrefix))
						if err != nil {
							return err
						}
						analysis.Commit = giventTag.Sha
					default:
						givenBranch, err := client.Branch(ctx, repo.Name, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(analysis.Ref, sdk.GitRefBranchPrefix), NoCache: true})
						if err != nil {
							return err
						}
						analysis.Commit = givenBranch.LatestCommit
					}
				} else if analysis.Ref == "" {
					defaultBranch, err := client.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true, NoCache: true})
					if err != nil {
						return err
					}
					analysis.Ref = defaultBranch.ID
					analysis.Commit = defaultBranch.LatestCommit
				}
			}

			isAdminMFA := false
			var u *sdk.AuthentifiedUser
			if !isHooks(ctx) {
				uc := getUserConsumer(ctx)
				if uc != nil {
					u = uc.AuthConsumerUser.AuthentifiedUser
				}
				isAdminMFA = isAdmin(ctx)
			} else if isHooks(ctx) && analysis.UserID != "" {
				u, err = user.LoadByID(ctx, api.mustDB(), analysis.UserID)
				if err != nil {
					return err
				}
				isAdminMFA = analysis.AdminMFA
				if isAdminMFA && u.Ring != sdk.UserRingAdmin {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "user %s is not admin", u.Username)
				}
			}

			createAnalysis := createAnalysisRequest{
				proj:          *proj,
				vcsProject:    *vcs,
				repo:          *repo,
				ref:           analysis.Ref,
				commit:        analysis.Commit,
				hookEventUUID: analysis.HookEventUUID,
				hookEventKey:  analysis.HookEventKey,
				user:          u,
				adminMFA:      isAdminMFA,
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			a, err := api.createAnalyze(ctx, tx, createAnalysis)
			if err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			response := sdk.AnalysisResponse{
				AnalysisID: a.ID,
				Status:     a.Status,
			}
			event_v2.PublishAnalysisStart(ctx, api.Cache, vcs.Name, repo.Name, a)
			return service.WriteJSON(w, response, http.StatusCreated)
		}
}

func (api *API) createAnalyze(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, analysisRequest createAnalysisRequest) (*sdk.ProjectRepositoryAnalysis, error) {
	ctx = context.WithValue(ctx, cdslog.VCSServer, analysisRequest.vcsProject.Name)
	ctx = context.WithValue(ctx, cdslog.Repository, analysisRequest.repo.Name)

	// Save analyze
	repoAnalysis := sdk.ProjectRepositoryAnalysis{
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		ProjectRepositoryID: analysisRequest.repo.ID,
		VCSProjectID:        analysisRequest.vcsProject.ID,
		ProjectKey:          analysisRequest.proj.Key,
		Ref:                 analysisRequest.ref,
		Commit:              analysisRequest.commit,
		Data: sdk.ProjectRepositoryData{
			HookEventUUID: analysisRequest.hookEventUUID,
			HookEventKey:  analysisRequest.hookEventKey,
		},
	}
	if analysisRequest.user != nil {
		repoAnalysis.Data.CDSUserID = analysisRequest.user.ID
		repoAnalysis.Data.CDSUserName = analysisRequest.user.Username
		repoAnalysis.Data.CDSAdminWithMFA = analysisRequest.adminMFA
	}

	if err := repository.InsertAnalysis(ctx, tx, &repoAnalysis); err != nil {
		return nil, err
	}

	return &repoAnalysis, nil
}

func (api *API) repositoryAnalysisPoller(ctx context.Context, tick time.Duration) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-ticker.C:
			analysis, err := repository.LoadRepositoryIDsAnalysisInProgress(ctx, api.mustDB())
			if err != nil {
				log.Error(ctx, "unable to load analysis in progress: %v", err)
				continue
			}
			for _, a := range analysis {
				api.GoRoutines.Exec(
					ctx,
					"repositoryAnalysisPoller-"+a.ID,
					func(ctx context.Context) {
						ctx = telemetry.New(ctx, api, "api.repositoryAnalysisPoller", nil, trace.SpanKindUnspecified)
						if err := api.analyzeRepository(ctx, a.ProjectRepositoryID, a.ID); err != nil {
							log.ErrorWithStackTrace(ctx, err)
						}
					},
				)
			}
		}
	}
}

func (api *API) analyzeRepository(ctx context.Context, projectRepoID string, analysisID string) error {
	ctx, next := telemetry.Span(ctx, "api.analyzeRepository.lock")
	defer next()

	ctx = context.WithValue(ctx, cdslog.AnalyzeID, analysisID)

	lockKey := cache.Key("api:analyzeRepository", analysisID)
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		return err
	}
	if !b {
		log.Debug(ctx, "api.analyzeRepository> analyze %s is locked in cache", analysisID)
		return nil
	}
	defer func() {
		_ = api.Cache.Unlock(lockKey)
	}()

	analysis, err := repository.LoadRepositoryAnalysisById(ctx, api.mustDB(), projectRepoID, analysisID)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load analyze %s", analysis.ID))
	}

	proj, err := project.Load(ctx, api.mustDB(), analysis.ProjectKey)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load project %s", analysis.ProjectKey))
	}

	if analysis.Status != sdk.RepositoryAnalysisStatusInProgress {
		return nil
	}

	vcsProjectWithSecret, err := vcs.LoadVCSByIDAndProjectKey(ctx, api.mustDB(), analysis.ProjectKey, analysis.VCSProjectID, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load vcs %s", analysis.VCSProjectID))
	}

	repo, err := repository.LoadRepositoryByID(ctx, api.mustDB(), analysis.ProjectRepositoryID)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load repository %s", analysis.ProjectRepositoryID))
	}

	// Check if there is an analysis on the current repository
	lockKeyRepo := cache.Key("api:repository:analyzeRepository", repo.ID)
	bRepoLock, err := api.Cache.Lock(lockKeyRepo, 5*time.Minute, 0, 1)
	if err != nil {
		return err
	}
	if !bRepoLock {
		log.Debug(ctx, "api.analyzeRepository> repository %s is locked in cache", repo.ID)
		return nil
	}
	defer func() {
		_ = api.Cache.Unlock(lockKeyRepo)
	}()

	var userDB sdk.AuthentifiedUser

	defer func() {
		event_v2.PublishAnalysisDone(ctx, api.Cache, vcsProjectWithSecret.Name, repo.Name, analysis, userDB)
	}()

	entitiesUpdated := make([]sdk.Entity, 0)
	defer func() {
		if err := sendAnalysisHookCallback(ctx, api.mustDB(), *analysis, entitiesUpdated, vcsProjectWithSecret.Name, repo.Name); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	}()

	ctx = context.WithValue(ctx, cdslog.VCSServer, vcsProjectWithSecret.Name)
	ctx = context.WithValue(ctx, cdslog.Repository, repo.Name)

	// If no user triggered the analysis, retrieve the signing key
	if analysis.Data.CDSUserID == "" {
		// Check Commit Signature
		var keyID, analysisError string
		switch vcsProjectWithSecret.Type {
		case sdk.VCSTypeBitbucketCloud, sdk.VCSTypeGitlab:
			keyID, analysisError, err = api.analyzeCommitSignatureThroughOperation(ctx, analysis, *vcsProjectWithSecret, *repo)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check the commit signature"))
			}
		case sdk.VCSTypeGitea, sdk.VCSTypeGithub, sdk.VCSTypeBitbucketServer:
			keyID, analysisError, err = api.analyzeCommitSignatureThroughVcsAPI(ctx, *analysis, *vcsProjectWithSecret, *repo)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check the commit signature"))
			}
		default:
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to analyze vcs of type: %s", vcsProjectWithSecret.Type))
		}
		analysis.Data.SignKeyID = keyID
		if analysisError != "" {
			analysis.Status = sdk.RepositoryAnalysisStatusSkipped
			analysis.Data.Error = analysisError
		} else {
			analysis.Data.CommitCheck = true

			// retrieve the committer
			u, analysisStatus, analysisError, err := findCommitter(ctx, api.Cache, api.mustDB(), analysis.Commit, analysis.Data.SignKeyID, analysis.ProjectKey, *vcsProjectWithSecret, repo.Name, api.Config.VCS.GPGKeys)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, err)
			}
			if u == nil {
				analysis.Status = analysisStatus
				analysis.Data.Error = analysisError
			} else {
				analysis.Data.CDSUserID = u.ID
				analysis.Data.CDSUserName = u.Username
			}
		}
	}

	// End analysis if needed
	if analysis.Status != sdk.RepositoryAnalysisStatusInProgress {
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
			_ = tx.Rollback()
			return sdk.WrapError(err, "unable to update analysis")
		}
		return sdk.WithStack(tx.Commit())
	}

	// Load committer / user that trigger the analysis
	u, err := user.LoadByID(ctx, api.mustDB(), analysis.Data.CDSUserID)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, err)
	}
	userDB = *u

	// Retrieve files content
	var filesContent map[string][]byte
	switch vcsProjectWithSecret.Type {
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud:
		// get archive
		filesContent, err = api.getCdsArchiveFileOnRepo(ctx, *repo, analysis, vcsProjectWithSecret.Name)
	case sdk.VCSTypeGitlab, sdk.VCSTypeGithub, sdk.VCSTypeGitea:
		analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
		filesContent, err = api.getCdsFilesOnVCSDirectory(ctx, analysis, vcsProjectWithSecret.Name, repo.Name, analysis.Commit, ".cds")
	case sdk.VCSTypeGerrit:
		return sdk.WithStack(sdk.ErrNotImplemented)
	}

	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve files"))
	}

	if len(filesContent) == 0 && analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
		analysis.Status = sdk.RepositoryAnalysisStatusSkipped
		analysis.Data.Error = "no cds files found"
	}

	if analysis.Status != sdk.RepositoryAnalysisStatusInProgress {
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
			_ = tx.Rollback()
			return sdk.WrapError(err, "unable to update analysis")
		}
		return sdk.WithStack(tx.Commit())
	}

	ef := NewEntityFinder(proj.Key, analysis.Ref, analysis.Commit, *repo, *vcsProjectWithSecret, *u, analysis.Data.CDSAdminWithMFA, api.Config.WorkflowV2.LibraryProjectKey)

	// Transform file content into entities
	entities, multiErr := api.handleEntitiesFiles(ctx, ef, filesContent, analysis)
	if multiErr != nil {
		return api.stopAnalysis(ctx, analysis, multiErr...)
	}

	userRoles := make(map[string]bool)
	skippedEntities := make([]sdk.EntityWithObject, 0)
	skippedFiles := make(sdk.StringSlice, 0)

	// Load existing entity on the current branch
	existingEntities, err := entity.LoadByRepositoryAndRefAndCommit(ctx, api.mustDB(), analysis.ProjectRepositoryID, analysis.Ref, "HEAD")
	if err != nil {
		return api.stopAnalysis(ctx, analysis, err)
	}

	// Build user role map
	for _, t := range sdk.EntityTypes {
		if _, has := userRoles[t]; !has {
			roleName, err := sdk.GetManageRoleByEntity(t)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, err)
			}
			b, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), roleName, analysis.Data.CDSUserID, analysis.ProjectKey)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check user permission"))
			}
			userRoles[t] = analysis.Data.CDSAdminWithMFA || b
			log.Info(ctx, "role: [%s] entity: [%s] has role: [%v]", roleName, t, b)
		}
	}

	// For each entities, check user role for each entities
	entitiesToUpdate := make([]sdk.EntityWithObject, 0)
skipEntity:
	for i := range entities {
		e := &entities[i]
		e.UserID = &userDB.ID

		for entityIndex := range analysis.Data.Entities {
			analysisEntity := &analysis.Data.Entities[entityIndex]
			if analysisEntity.Path+analysisEntity.FileName == e.FilePath {
				if userRoles[e.Type] {
					analysisEntity.Status = sdk.RepositoryAnalysisStatusSucceed
				} else {
					skippedFiles = append(skippedFiles, "User doesn't have the permission to manage "+e.Type)
					analysisEntity.Status = sdk.RepositoryAnalysisStatusSkipped
					skippedEntities = append(skippedEntities, *e)
					continue skipEntity
				}
				break
			}
		}
		entitiesToUpdate = append(entitiesToUpdate, *e)
	}

	// Insert / Update entities
	vcsAuthClient, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, analysis.ProjectKey, vcsProjectWithSecret.Name)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to retrieve vcs_server %s on project %s", vcsProjectWithSecret.Name, analysis.ProjectKey))
	}
	defaultBranch, err := vcsAuthClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true, NoCache: true})
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to retrieve default branch on repository %s", repo.Name))
	}

	var currentAnalysisBranch *sdk.VCSBranch
	var currentAnalysisTag sdk.VCSTag
	if analysis.Ref == defaultBranch.ID {
		currentAnalysisBranch = defaultBranch
	} else {
		if strings.HasPrefix(analysis.Ref, sdk.GitRefTagPrefix) {
			currentAnalysisTag, err = vcsAuthClient.Tag(ctx, repo.Name, strings.TrimPrefix(analysis.Ref, sdk.GitRefTagPrefix))
			if err != nil {
				return err
			}
		} else {
			currentAnalysisBranch, err = vcsAuthClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(analysis.Ref, sdk.GitRefBranchPrefix), NoCache: true})
			if err != nil {
				return err
			}
		}
	}

	eventInsertedEntities := make([]sdk.Entity, 0)
	eventUpdatedEntities := make([]sdk.Entity, 0)
	eventRemovedEntities := make([]sdk.Entity, 0)

	newHooks := make([]sdk.V2WorkflowHook, 0)

	srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeHooks)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to retrieve hook service"))
	}
	if len(srvs) < 1 {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to find hook service"))
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	for i := range entitiesToUpdate {
		e := &entitiesToUpdate[i]

		// Check if entity has changed from current HEAD
		var entityUpdated bool
		existingHeadEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, "HEAD")
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve latest entity with same name"))
		}
		if existingHeadEntity == nil || existingHeadEntity.Data != e.Entity.Data {
			entityUpdated = true
		}

		// If entity already exist for this, ignore it.
		existingEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, e.Commit)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check if %s of type %s already exist on git ref %s", e.Name, e.Type, e.Ref))
		}

		if existingEntity != nil {
			if existingEntity.Data == e.Entity.Data {
				continue
			} else {
				e.Entity.ID = existingEntity.ID
				if err := entity.Update(ctx, tx, &e.Entity); err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to update %s of type %s for ref %s and commit %s", e.Name, e.Type, e.Ref, e.Commit))
				}
			}
		} else {
			// Insert new entity for current branch and commit
			if err := entity.Insert(ctx, tx, &e.Entity); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save %s of type %s for ref %s and commit %s", e.Name, e.Type, e.Ref, e.Commit))
			}
			eventInsertedEntities = append(eventInsertedEntities, e.Entity)
		}

		// If it's an update, add it to the list of entity updated (for hooks)
		if entityUpdated {
			entitiesUpdated = append(entitiesUpdated, e.Entity)
		}
		// If current commit is HEAD, create/update HEAD entity
		if (currentAnalysisBranch != nil && currentAnalysisBranch.LatestCommit == e.Commit) || (currentAnalysisTag.Sha == e.Commit) || currentAnalysisTag.Sha == "" {
			if entityUpdated {
				newHead := e.Entity
				newHead.ID = ""
				newHead.Commit = "HEAD"
				if existingHeadEntity != nil {
					newHead.ID = existingHeadEntity.ID
					if err := entity.Update(ctx, tx, &newHead); err != nil {
						return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to update HEAD entity %s of type %s", e.Name, e.Type))
					}
					eventUpdatedEntities = append(eventUpdatedEntities, newHead)
				} else {
					if err := entity.Insert(ctx, tx, &newHead); err != nil {
						return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save HEAD entity %s of type %s", e.Name, e.Type))
					}
					eventInsertedEntities = append(eventInsertedEntities, newHead)
				}
			}
		}

		// Insert workflow hook
		if e.Type == sdk.EntityTypeWorkflow {
			hooks, err := manageWorkflowHooks(ctx, tx, api.Cache, ef, *e, vcsProjectWithSecret.Name, repo.Name, defaultBranch, srvs)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, fmt.Sprintf("unable to create workflow hooks for %s", e.Name)))
			}
			newHooks = append(newHooks, hooks...)
		}
	}

	// For skipped entities (user has no right to manage them), retrieve definition on head or default branch
	for _, e := range skippedEntities {
		// Get definition for the current commit
		existingEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, e.Commit)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check if %s of type %s already exist on git ref %s", e.Name, e.Type, e.Ref))
		}

		// Get definition for HEAD commit
		existingHeadEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, "HEAD")
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve HEAD entity with same name on same branch"))
		}

		// If both exist, skip
		if existingEntity != nil && existingHeadEntity != nil {
			continue
		}

		entityInserted := false

		// Create definition for commit HEAD from default branch
		if existingHeadEntity == nil {
			defaultBranchHeadEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, defaultBranch.ID, e.Type, e.Name, "HEAD")
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve latest entity on default branch with same name"))
			}
			if defaultBranchHeadEntity != nil {
				existingHeadEntity = &sdk.Entity{
					ProjectKey:          e.ProjectKey,
					ProjectRepositoryID: e.ProjectRepositoryID,
					Type:                e.Type,
					FilePath:            e.FilePath,
					Name:                e.Name,
					Commit:              "HEAD",
					Ref:                 e.Ref,
					Data:                defaultBranchHeadEntity.Data,
					UserID:              defaultBranchHeadEntity.UserID,
				}
				if err := entity.Insert(ctx, tx, existingHeadEntity); err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save skipped entity %s of type %s", e.Name, e.Type))
				}
			}
		}

		// Create definition for the current sha
		if existingEntity == nil && existingHeadEntity != nil {
			e.Data = existingHeadEntity.Data
			e.UserID = existingHeadEntity.UserID
			if err := entity.Insert(ctx, tx, &e.Entity); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save skipped entity from head %s of type %s", e.Name, e.Type))
			}
			entityInserted = true
		}

		// Insert workflow hook
		if entityInserted && e.Type == sdk.EntityTypeWorkflow {
			hooks, err := manageWorkflowHooks(ctx, tx, api.Cache, ef, e, vcsProjectWithSecret.Name, repo.Name, defaultBranch, srvs)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, fmt.Sprintf("unable to create workflow hooks for %s", e.Name)))
			}
			newHooks = append(newHooks, hooks...)
		}
	}

	// For head commit, check if entities are missing and delete them
	if (currentAnalysisBranch != nil && currentAnalysisBranch.LatestCommit == analysis.Commit) || (currentAnalysisTag.Sha == analysis.Commit) || currentAnalysisTag.Sha == "" {
		foundEntities := make(map[string]struct{})
		for _, e := range entities {
			foundEntities[e.Type+"-"+e.Name] = struct{}{}
		}
		delOpts := DeleteEntityOps{}
		if currentAnalysisBranch != nil && currentAnalysisBranch.Default && currentAnalysisBranch.LatestCommit == analysis.Commit {
			delOpts.WithHooks = true
		}
		for _, e := range existingEntities {
			// If an existing entities has not been found in the current head commit (deleted or renamed)
			// => remove the entity
			if _, has := foundEntities[e.Type+"-"+e.Name]; !has {
				// Check right
				if !userRoles[e.Type] {
					log.Warn(ctx, "user %s [%s] removed the entity %s [%s] but it has not the right to do it", u.Username, u.ID, e.Name, e.Type)
					continue
				}
				if err := DeleteEntity(ctx, tx, &e, srvs, delOpts); err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, fmt.Sprintf("unable to delete entity %s [%s] ", e.Name, e.Type)))
				}
				eventRemovedEntities = append(eventRemovedEntities, e)
			}
		}
	}

	schedulers := make([]sdk.V2WorkflowHook, 0)
	for _, h := range newHooks {
		if h.Type == sdk.WorkflowHookTypeScheduler {
			schedulers = append(schedulers, h)
		}
	}

	// Update analysis
	skippedFiles.Unique()
	analysis.Data.Error = strings.Join(skippedFiles, "\n")
	if len(skippedFiles) == len(analysis.Data.Entities) {
		analysis.Status = sdk.RepositoryAnalysisStatusSkipped
		if len(analysis.Data.Entities) == 0 {
			analysis.Data.Error = "no file found"
		}
	} else if len(schedulers) == 0 {
		analysis.Status = sdk.RepositoryAnalysisStatusSucceed
	}

	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		return sdk.WrapError(err, "unable to update analysis")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(tx.Commit())
	}

	for _, e := range eventInsertedEntities {
		event_v2.PublishEntityEvent(ctx, api.Cache, sdk.EventEntityCreated, repo.Name, repo.Name, e, &userDB)
	}
	for _, eUpdated := range eventUpdatedEntities {
		event_v2.PublishEntityEvent(ctx, api.Cache, sdk.EventEntityUpdated, vcsProjectWithSecret.Name, repo.Name, eUpdated, &userDB)
	}
	for _, eRemoved := range eventRemovedEntities {
		event_v2.PublishEntityEvent(ctx, api.Cache, sdk.EventEntityDeleted, vcsProjectWithSecret.Name, repo.Name, eRemoved, &userDB)
	}

	if len(schedulers) != 0 {
		// Instantiate Schedulers on hooks µservice
		if _, _, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/v2/workflow/scheduler", schedulers, nil); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to instantiate scheduler on hook service"))
		}
	}

	return nil
}

func manageWorkflowHooks(ctx context.Context, db gorpmapper.SqlExecutorWithTx, cache cache.Store, ef *EntityFinder, e sdk.EntityWithObject, workflowDefVCSName, workflowDefRepositoryName string, defaultBranch *sdk.VCSBranch, hookSrvs []sdk.Service) ([]sdk.V2WorkflowHook, error) {
	ctx, next := telemetry.Span(ctx, "manageWorkflowHooks")
	defer next()

	// If there is a workflow template and non hooks on workflow, check the workflow template
	if e.Workflow.From != "" && e.Workflow.On == nil {
		var wkfTmpl sdk.V2WorkflowTemplate
		if strings.HasPrefix(e.Workflow.From, ".cds/") {
			tmpl, err := entity.LoadEntityByPathAndRefAndCommit(ctx, db, e.ProjectRepositoryID, e.Workflow.From, e.Ref, e.Commit)
			if err != nil {
				return nil, err
			}
			if err := yaml.Unmarshal([]byte(tmpl.Data), &wkfTmpl); err != nil {
				return nil, err
			}
		} else {
			completePath, errMsg, err := ef.searchEntity(ctx, db, cache, e.Workflow.From, sdk.EntityTypeWorkflowTemplate)
			if err != nil {
				return nil, err
			}
			if errMsg != "" {
				return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, errMsg)
			}
			workflowTemplate := ef.templatesCache[completePath]
			wkfTmpl = workflowTemplate.Template
		}
		if _, err := wkfTmpl.Resolve(ctx, &e.Workflow); err != nil {
			return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to compute workflow from template: %v", err)
		}
	}

	hooks := make([]sdk.V2WorkflowHook, 0)

	// Remove existing hook for the current branch
	if e.Ref == defaultBranch.ID && e.Commit == defaultBranch.LatestCommit {
		// Search old scheduler definition and remove them from hooks uservice
		whs, err := workflow_v2.LoadHookSchedulerByWorkflow(ctx, db, e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, e.Name)
		if err != nil {
			return nil, err
		}
		if len(whs) > 0 {
			if err := DeleteAllEntitySchedulerHook(ctx, db, whs[0].VCSName, whs[0].RepositoryName, whs[0].WorkflowName, hookSrvs); err != nil {
				return nil, err
			}
		}
	}
	if err := workflow_v2.DeleteWorkflowHooks(ctx, db, e.ID); err != nil {
		return nil, err
	}

	targetVCS := workflowDefVCSName
	targetRepository := workflowDefRepositoryName
	if e.Workflow.Repository != nil {
		targetVCS = e.Workflow.Repository.VCSServer
		targetRepository = strings.ToLower(e.Workflow.Repository.Name)
	}

	workflowSameRepo := true
	if targetVCS != workflowDefVCSName || targetRepository != workflowDefRepositoryName {
		workflowSameRepo = false
	}

	// Only save hook push if
	// * workflow repo declaration == workflow.repository || default branch
	if e.Workflow.On != nil && e.Workflow.On.Push != nil {
		if workflowSameRepo || e.Ref == defaultBranch.ID {
			wh := sdk.V2WorkflowHook{
				EntityID:       e.ID,
				ProjectKey:     e.ProjectKey,
				Type:           sdk.WorkflowHookTypeRepository,
				Ref:            e.Ref,
				Commit:         e.Commit,
				WorkflowName:   e.Name,
				VCSName:        workflowDefVCSName,
				RepositoryName: workflowDefRepositoryName,
				Data: sdk.V2WorkflowHookData{
					RepositoryEvent: sdk.WorkflowHookEventNamePush,
					VCSServer:       targetVCS,
					RepositoryName:  targetRepository,
					CommitFilter:    e.Workflow.On.Push.Commit,
					BranchFilter:    e.Workflow.On.Push.Branches,
					TagFilter:       e.Workflow.On.Push.Tags,
					PathFilter:      e.Workflow.On.Push.Paths,
				},
			}
			if e.Workflow.Repository != nil {
				wh.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return nil, err
			}
			hooks = append(hooks, wh)

			if e.Ref == defaultBranch.ID && e.Commit == defaultBranch.LatestCommit {
				// Load existing head hook
				existingHook, err := workflow_v2.LoadHookHeadRepositoryWebHookByWorkflowAndEvent(ctx, db, e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, e.Name, sdk.WorkflowHookEventNamePush, defaultBranch.ID)
				if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return nil, err
				}
				if existingHook != nil {
					// Update data and ref
					existingHook.Data = sdk.V2WorkflowHookData{
						RepositoryEvent: sdk.WorkflowHookEventNamePush,
						VCSServer:       targetVCS,
						RepositoryName:  targetRepository,
						CommitFilter:    e.Workflow.On.Push.Commit,
						BranchFilter:    e.Workflow.On.Push.Branches,
						TagFilter:       e.Workflow.On.Push.Tags,
						PathFilter:      e.Workflow.On.Push.Paths,
					}
					if e.Workflow.Repository != nil {
						existingHook.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
					}
					existingHook.Ref = e.Ref
					if err := workflow_v2.UpdateWorkflowHook(ctx, db, existingHook); err != nil {
						return nil, err
					}
				} else {
					// Create head hook
					newHeadHook := sdk.V2WorkflowHook{
						EntityID:       e.ID,
						ProjectKey:     e.ProjectKey,
						Type:           sdk.WorkflowHookTypeRepository,
						Ref:            e.Ref,
						Commit:         "HEAD",
						WorkflowName:   e.Name,
						VCSName:        workflowDefVCSName,
						RepositoryName: workflowDefRepositoryName,
						Data: sdk.V2WorkflowHookData{
							RepositoryEvent: sdk.WorkflowHookEventNamePush,
							VCSServer:       targetVCS,
							RepositoryName:  targetRepository,
							CommitFilter:    e.Workflow.On.Push.Commit,
							BranchFilter:    e.Workflow.On.Push.Branches,
							TagFilter:       e.Workflow.On.Push.Tags,
							PathFilter:      e.Workflow.On.Push.Paths,
						},
					}
					if e.Workflow.Repository != nil {
						newHeadHook.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
					}
					if err := workflow_v2.InsertWorkflowHook(ctx, db, &newHeadHook); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	if e.Workflow.On != nil && e.Workflow.On.PullRequest != nil {
		if workflowSameRepo || e.Ref == defaultBranch.ID {
			wh := sdk.V2WorkflowHook{
				EntityID:       e.ID,
				ProjectKey:     e.ProjectKey,
				Type:           sdk.WorkflowHookTypeRepository,
				Ref:            e.Ref,
				Commit:         e.Commit,
				WorkflowName:   e.Name,
				VCSName:        workflowDefVCSName,
				RepositoryName: workflowDefRepositoryName,
				Data: sdk.V2WorkflowHookData{
					RepositoryEvent: sdk.WorkflowHookEventNamePullRequest,
					VCSServer:       targetVCS,
					RepositoryName:  targetRepository,
					BranchFilter:    e.Workflow.On.PullRequest.Branches,
					PathFilter:      e.Workflow.On.PullRequest.Paths,
					TypesFilter:     e.Workflow.On.PullRequest.Types,
				},
			}
			if e.Workflow.Repository != nil {
				wh.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return nil, err
			}
			hooks = append(hooks, wh)

			if e.Ref == defaultBranch.ID && e.Commit == defaultBranch.LatestCommit {
				// Load existing head hook
				existingHook, err := workflow_v2.LoadHookHeadRepositoryWebHookByWorkflowAndEvent(ctx, db, e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, e.Name, sdk.WorkflowHookEventNamePullRequest, defaultBranch.ID)
				if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return nil, err
				}
				if existingHook != nil {
					// Update data and ref
					existingHook.Data = sdk.V2WorkflowHookData{
						RepositoryEvent: sdk.WorkflowHookEventNamePullRequest,
						VCSServer:       targetVCS,
						RepositoryName:  targetRepository,
						BranchFilter:    e.Workflow.On.PullRequest.Branches,
						PathFilter:      e.Workflow.On.PullRequest.Paths,
						TypesFilter:     e.Workflow.On.PullRequest.Types,
					}
					if e.Workflow.Repository != nil {
						existingHook.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
					}
					existingHook.Ref = e.Ref
					if err := workflow_v2.UpdateWorkflowHook(ctx, db, existingHook); err != nil {
						return nil, err
					}
				} else {
					// Create head hook
					newHeadHook := sdk.V2WorkflowHook{
						EntityID:       e.ID,
						ProjectKey:     e.ProjectKey,
						Type:           sdk.WorkflowHookTypeRepository,
						Ref:            e.Ref,
						Commit:         "HEAD",
						WorkflowName:   e.Name,
						VCSName:        workflowDefVCSName,
						RepositoryName: workflowDefRepositoryName,
						Data: sdk.V2WorkflowHookData{
							RepositoryEvent: sdk.WorkflowHookEventNamePullRequest,
							VCSServer:       targetVCS,
							RepositoryName:  targetRepository,
							BranchFilter:    e.Workflow.On.PullRequest.Branches,
							PathFilter:      e.Workflow.On.PullRequest.Paths,
							TypesFilter:     e.Workflow.On.PullRequest.Types,
						},
					}
					if e.Workflow.Repository != nil {
						newHeadHook.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
					}
					if err := workflow_v2.InsertWorkflowHook(ctx, db, &newHeadHook); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	if e.Workflow.On != nil && e.Workflow.On.PullRequestComment != nil {
		if workflowSameRepo || e.Ref == defaultBranch.ID {
			wh := sdk.V2WorkflowHook{
				EntityID:       e.ID,
				ProjectKey:     e.ProjectKey,
				Type:           sdk.WorkflowHookTypeRepository,
				Ref:            e.Ref,
				Commit:         e.Commit,
				WorkflowName:   e.Name,
				VCSName:        workflowDefVCSName,
				RepositoryName: workflowDefRepositoryName,
				Data: sdk.V2WorkflowHookData{
					RepositoryEvent: sdk.WorkflowHookEventNamePullRequestComment,
					VCSServer:       targetVCS,
					RepositoryName:  targetRepository,
					BranchFilter:    e.Workflow.On.PullRequest.Branches,
					PathFilter:      e.Workflow.On.PullRequest.Paths,
					TypesFilter:     e.Workflow.On.PullRequest.Types,
				},
			}
			if e.Workflow.Repository != nil {
				wh.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return nil, err
			}
			hooks = append(hooks, wh)
		}
	}

	// Only save workflow_update hook if :
	// * workflow.repository declaration != workflow.repository && default branch
	if e.Workflow.On != nil && e.Workflow.On.WorkflowUpdate != nil {
		if !workflowSameRepo && e.Ref == defaultBranch.ID {
			wh := sdk.V2WorkflowHook{
				VCSName:        workflowDefVCSName,
				EntityID:       e.ID,
				ProjectKey:     e.ProjectKey,
				Type:           sdk.WorkflowHookTypeWorkflow,
				Ref:            e.Ref,
				Commit:         e.Commit,
				WorkflowName:   e.Name,
				RepositoryName: workflowDefRepositoryName,
				Data: sdk.V2WorkflowHookData{
					VCSServer:      e.Workflow.Repository.VCSServer,
					RepositoryName: e.Workflow.Repository.Name,
					TargetBranch:   e.Workflow.On.WorkflowUpdate.TargetBranch,
				},
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return nil, err
			}
			hooks = append(hooks, wh)
		}
	}

	// Only save model_update hook if :
	// * workflow and model on the same repo
	// * workflow repo declaration != workflow.repository && default branch
	if e.Workflow.On != nil && e.Workflow.On.ModelUpdate != nil {
		if e.Ref == defaultBranch.ID {
			for _, m := range e.Workflow.On.ModelUpdate.Models {
				mSplit := strings.Split(m, "/")
				var modelVCSName, modelRepoName, modelFullName string
				switch len(mSplit) {
				case 1:
					modelVCSName = workflowDefVCSName
					modelRepoName = workflowDefRepositoryName
					modelFullName = fmt.Sprintf("%s/%s/%s/%s", e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, m)
				case 3:
					modelVCSName = workflowDefVCSName
					modelRepoName = mSplit[0] + "/" + mSplit[1]
					modelFullName = fmt.Sprintf("%s/%s/%s", e.ProjectKey, workflowDefVCSName, m)
				case 4:
					modelVCSName = mSplit[0]
					modelRepoName = mSplit[1] + "/" + mSplit[2]
					modelFullName = fmt.Sprintf("%s/%s", e.ProjectKey, m)
				case 5:
					modelVCSName = mSplit[1]
					modelRepoName = mSplit[2] + "/" + mSplit[3]
					modelFullName = m
				}
				// Default branch && workflow and model on the same repo && distant workflow
				if modelVCSName == workflowDefVCSName && modelRepoName == workflowDefRepositoryName &&
					(workflowDefVCSName != e.Workflow.Repository.VCSServer || workflowDefRepositoryName != e.Workflow.Repository.Name) {

					wh := sdk.V2WorkflowHook{
						VCSName:        workflowDefVCSName,
						EntityID:       e.ID,
						ProjectKey:     e.ProjectKey,
						Type:           sdk.WorkflowHookTypeWorkerModel,
						Ref:            e.Ref,
						Commit:         e.Commit,
						WorkflowName:   e.Name,
						RepositoryName: workflowDefRepositoryName,
						Data: sdk.V2WorkflowHookData{
							VCSServer:      e.Workflow.Repository.VCSServer,
							RepositoryName: e.Workflow.Repository.Name,
							TargetBranch:   e.Workflow.On.ModelUpdate.TargetBranch,
							Model:          modelFullName,
						},
					}
					if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
						return nil, err
					}
					hooks = append(hooks, wh)
				}
			}
		}
	}

	// Manage scheduler
	// * On default branch, latest commit:
	//   1. remove old definition of the scheduler in DB + in hooks
	//   2. insert new definition
	if e.Ref == defaultBranch.ID && e.Commit == defaultBranch.LatestCommit && e.Workflow.On != nil {
		// Insert new scheduler definition
		destVCS := workflowDefVCSName
		destRepo := workflowDefRepositoryName
		if e.Workflow.Repository != nil {
			destVCS = e.Workflow.Repository.VCSServer
			destRepo = e.Workflow.Repository.Name
		}

		for _, s := range e.Workflow.On.Schedule {
			// Add in data desitnation vcs / repo
			wh := sdk.V2WorkflowHook{
				VCSName:        workflowDefVCSName,
				EntityID:       e.ID,
				ProjectKey:     e.ProjectKey,
				Type:           sdk.WorkflowHookTypeScheduler,
				Ref:            e.Ref,
				Commit:         e.Commit,
				WorkflowName:   e.Name,
				RepositoryName: workflowDefRepositoryName,
				Data: sdk.V2WorkflowHookData{
					Cron:           s.Cron,
					CronTimeZone:   s.Timezone,
					VCSServer:      destVCS,
					RepositoryName: destRepo,
				},
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return nil, err
			}
			hooks = append(hooks, wh)
		}

	}

	// Manage workflow-run
	// * On default branch, latest commit:
	//   1. remove old definition of the scheduler in DB
	//   2. insert new definition
	if e.Ref == defaultBranch.ID && e.Commit == defaultBranch.LatestCommit && e.Workflow.On != nil {

		destVCS := workflowDefVCSName
		destRepo := workflowDefRepositoryName
		if e.Workflow.Repository != nil {
			destVCS = e.Workflow.Repository.VCSServer
			destRepo = e.Workflow.Repository.Name
		}

		for _, s := range e.Workflow.On.WorkflowRun {
			mSplit := strings.Split(s.Workflow, "/")
			var workflowFullName string
			switch len(mSplit) {
			case 1:
				workflowFullName = fmt.Sprintf("%s/%s/%s/%s", e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, s.Workflow)
			case 3:
				workflowFullName = fmt.Sprintf("%s/%s/%s", e.ProjectKey, workflowDefVCSName, s.Workflow)
			case 4:
				workflowFullName = fmt.Sprintf("%s/%s", e.ProjectKey, s.Workflow)
			case 5:
				workflowFullName = s.Workflow
			}

			// Add in data desitnation vcs / repo
			wh := sdk.V2WorkflowHook{
				VCSName:        workflowDefVCSName,
				EntityID:       e.ID,
				ProjectKey:     e.ProjectKey,
				Type:           sdk.WorkflowHookTypeWorkflowRun,
				Ref:            e.Ref,
				Commit:         e.Commit,
				WorkflowName:   e.Name,
				RepositoryName: workflowDefRepositoryName,
				Data: sdk.V2WorkflowHookData{
					// Destination repository
					VCSServer:      destVCS,
					RepositoryName: destRepo,

					// Workflow run to react
					WorkflowRunName:   workflowFullName, // searchEntity return proj/vcs/repo/name@ref, we must remove @ref,
					BranchFilter:      s.Branches,
					TagFilter:         s.Tags,
					WorkflowRunStatus: s.Status,
				},
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return nil, err
			}
			hooks = append(hooks, wh)
		}

	}
	return hooks, nil
}

func sendAnalysisHookCallback(ctx context.Context, db *gorp.DbMap, analysis sdk.ProjectRepositoryAnalysis, entities []sdk.Entity, vcsServerName, repoName string) error {
	// Remove hooks
	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeHooks)
	if err != nil {
		return err
	}
	if len(srvs) < 1 {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find hook uservice")
	}
	callback := sdk.HookEventCallback{
		RepositoryName: repoName,
		VCSServerName:  vcsServerName,
		HookEventUUID:  analysis.Data.HookEventUUID,
		HookEventKey:   analysis.Data.HookEventKey,
		AnalysisCallback: &sdk.HookAnalysisCallback{
			AnalysisStatus: analysis.Status,
			AnalysisID:     analysis.ID,
			Error:          analysis.Data.Error,
			Models:         make([]sdk.EntityFullName, 0),
			Workflows:      make([]sdk.EntityFullName, 0),
			Username:       analysis.Data.CDSUserName,
			UserID:         analysis.Data.CDSUserID,
		},
	}
	for _, e := range entities {
		ent := sdk.EntityFullName{
			ProjectKey: e.ProjectKey,
			VCSName:    vcsServerName,
			RepoName:   repoName,
			Name:       e.Name,
			Ref:        e.Ref,
		}
		switch e.Type {
		case sdk.EntityTypeWorkerModel:
			callback.AnalysisCallback.Models = append(callback.AnalysisCallback.Models, ent)
		case sdk.EntityTypeWorkflow:
			callback.AnalysisCallback.Workflows = append(callback.AnalysisCallback.Workflows, ent)
		}
	}

	if _, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/v2/repository/event/callback", callback, nil); err != nil {
		return sdk.WrapError(err, "unable to send analysis call to  hook [HTTP: %d]", code)
	}
	return nil
}

// findCommitter
func findCommitter(ctx context.Context, cache cache.Store, db *gorp.DbMap, sha, signKeyID, projKey string, vcsProjectWithSecret sdk.VCSProject, repoName string, vcsPublicKeys map[string][]GPGKey) (*sdk.AuthentifiedUser, string, string, error) {
	ctx, next := telemetry.Span(ctx, "findCommitter", trace.StringAttribute(telemetry.TagProjectKey, projKey), trace.StringAttribute(telemetry.TagVCSServer, vcsProjectWithSecret.Name), trace.StringAttribute(telemetry.TagRepository, repoName))
	defer next()

	publicKeyFound := false
	publicKeys, has := vcsPublicKeys[vcsProjectWithSecret.Name]
	if has {
		for _, k := range publicKeys {
			if signKeyID == k.ID {
				publicKeyFound = true
				break
			}
		}
	}

	if !publicKeyFound {
		gpgKey, err := user.LoadGPGKeyByKeyID(ctx, db, signKeyID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", sdk.NewErrorFrom(err, "unable get gpg key: %s", signKeyID)
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("gpgkey %s not found", signKeyID), nil
		}

		cdsUser, err := user.LoadByID(ctx, db, gpgKey.AuthentifiedUserID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", sdk.WithStack(sdk.NewErrorFrom(err, "unable to load user %s", gpgKey.AuthentifiedUserID))
			}
			return nil, sdk.RepositoryAnalysisStatusError, fmt.Sprintf("user %s not found for gpg key %s", gpgKey.AuthentifiedUserID, gpgKey.KeyID), nil
		}
		return cdsUser, "", "", nil

	}

	client, err := repositoriesmanager.AuthorizedClient(ctx, db, cache, projKey, vcsProjectWithSecret.Name)
	if err != nil {
		return nil, "", "", sdk.WithStack(err)
	}

	var commitUser *sdk.AuthentifiedUser

	// Get commit
	tx, err := db.Begin()
	if err != nil {
		return nil, "", "", sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	switch vcsProjectWithSecret.Type {
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeGitlab:
		commit, err := client.Commit(ctx, repoName, sha)
		if err != nil {
			return nil, "", "", err
		}
		commitUser, err = user.LoadByUsername(ctx, db, commit.Committer.Slug)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", sdk.WithStack(sdk.NewErrorFrom(err, "unable to get user %s", commit.Committer.Slug))
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("committer %s not found in CDS", commit.Committer.Slug), nil
		}
	case sdk.VCSTypeGithub:
		commit, err := client.Commit(ctx, repoName, sha)
		if err != nil {
			return nil, "", "", err
		}

		if commit.Committer.ID == "" {
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("unable to find commiter for commit %s", sha), nil
		}

		// Retrieve user link by external ID
		userLink, err := link.LoadUserLinkByTypeAndExternalID(ctx, db, vcsProjectWithSecret.Type, commit.Committer.ID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", err
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("%s user %s not found in CDS", vcsProjectWithSecret.Type, commit.Committer.Name), nil
		}

		// Check if username changed
		if userLink.Username != commit.Committer.Name {
			// Update user link
			userLink.Username = commit.Committer.Name
			if err := link.Update(ctx, tx, userLink); err != nil {
				return nil, "", "", err
			}
		}

		// Load user
		commitUser, err = user.LoadByID(ctx, tx, userLink.AuthentifiedUserID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrUserNotFound) {
				return nil, "", "", err
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("committer %s not found in CDS", commit.Committer.Name), nil
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, "", "", sdk.WithStack(err)
	}

	return commitUser, "", "", nil
}

func sortEntitiesFiles(filesContent map[string][]byte) []string {
	keys := make([]string, 0, len(filesContent))
	for k := range filesContent {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		switch {
		case strings.HasPrefix(keys[i], ".cds/worker-models/"):
			if strings.HasPrefix(keys[j], ".cds/worker-models/") {
				return keys[i] < keys[j]
			}
			return true
		case strings.HasPrefix(keys[i], ".cds/workflows"):
			if strings.HasPrefix(keys[j], ".cds/workflows/") {
				return keys[i] < keys[j]
			}
			return false
		case strings.HasPrefix(keys[i], ".cds/actions"):
			if strings.HasPrefix(keys[j], ".cds/worker-models/") {
				return false
			}
			if strings.HasPrefix(keys[j], ".cds/workflow-templates/") {
				return true
			}
			if strings.HasPrefix(keys[j], ".cds/workflows/") {
				return true
			}
			return keys[i] < keys[j]
		case strings.HasPrefix(keys[i], ".cds/workflow-templates"):
			if strings.HasPrefix(keys[j], ".cds/worker-models/") {
				return false
			}
			if strings.HasPrefix(keys[j], ".cds/actions/") {
				return false
			}
			if strings.HasPrefix(keys[j], ".cds/workflows/") {
				return true
			}
			return keys[i] < keys[j]
		}
		return keys[i] < keys[j]
	})
	return keys
}

func (api *API) handleEntitiesFiles(ctx context.Context, ef *EntityFinder, filesContent map[string][]byte, analysis *sdk.ProjectRepositoryAnalysis) ([]sdk.EntityWithObject, []error) {
	sortedKeys := sortEntitiesFiles(filesContent)

	entities := make([]sdk.EntityWithObject, 0)
	analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
	for _, filePath := range sortedKeys {
		content := filesContent[filePath]
		dir, fileName := filepath.Split(filePath)
		var es []sdk.EntityWithObject
		var err sdk.MultiError
		switch {
		case strings.HasPrefix(filePath, ".cds/worker-models/"):
			var wms []sdk.V2WorkerModel
			es, err = ReadEntityFile(ctx, api, dir, fileName, content, &wms, sdk.EntityTypeWorkerModel, *analysis, ef)
		case strings.HasPrefix(filePath, ".cds/actions/"):
			var actions []sdk.V2Action
			es, err = ReadEntityFile(ctx, api, dir, fileName, content, &actions, sdk.EntityTypeAction, *analysis, ef)
		case strings.HasPrefix(filePath, ".cds/workflows/"):
			var w []sdk.V2Workflow
			es, err = ReadEntityFile(ctx, api, dir, fileName, content, &w, sdk.EntityTypeWorkflow, *analysis, ef)
		case strings.HasPrefix(filePath, ".cds/workflow-templates/"):
			var wt []sdk.V2WorkflowTemplate
			es, err = ReadEntityFile(ctx, api, dir, fileName, content, &wt, sdk.EntityTypeWorkflowTemplate, *analysis, ef)
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		entities = append(entities, es...)
		analysis.Data.Entities = append(analysis.Data.Entities, sdk.ProjectRepositoryDataEntity{
			FileName: fileName,
			Path:     dir,
		})
	}
	return entities, nil

}

func Lint[T sdk.Lintable](ctx context.Context, api *API, o T, ef *EntityFinder) []error {
	// 1. Static lint
	if err := o.Lint(); err != nil {
		return err
	}

	// 2. Lint againt some API specific rules

	var err []error
	switch x := any(o).(type) {
	case sdk.V2WorkerModel:
		// 2.1 Validate docker image against the whitelist from API configuration
		var dockerSpec sdk.V2WorkerModelDockerSpec
		if err := json.Unmarshal(x.Spec, &dockerSpec); err != nil {
			// Check only docker spec, so we skipp other errors
			break
		}
		// Verify the image if any whitelist is setup
		if dockerSpec.Image != "" && len(api.WorkerModelDockerImageWhiteList) > 0 {
			var allowedImage = false
			for _, r := range api.WorkerModelDockerImageWhiteList { // At least one regexp must match
				if r.MatchString(dockerSpec.Image) {
					allowedImage = true
					break
				}
			}
			if !allowedImage {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "image %q is not allowed", dockerSpec.Image))
			}
		}
	case sdk.V2Workflow:
		switch {
		case x.From != "":
			var tmpl *sdk.V2WorkflowTemplate
			if strings.HasPrefix(x.From, ".cds/workflow-templates/") {
				// Retrieve tmpl from current analysis
				for _, v := range ef.templatesCache {
					if v.FilePath == x.From {
						tmpl = &v.Template
						break
					}
				}
			} else {
				// Retrieve existing template in DB
				path, msg, errSearch := ef.searchEntity(ctx, api.mustDB(), api.Cache, x.From, sdk.EntityTypeWorkflowTemplate)
				if errSearch != nil {
					err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to retrieve entity %s of type %s: %v", x.From, sdk.EntityTypeWorkflowTemplate, errSearch))
					break
				}
				if msg != "" {
					err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, msg))
					break
				}
				t := ef.templatesCache[path].Template
				tmpl = &t
			}
			if tmpl == nil || tmpl.Name == "" {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unknown workflow template %s", x.From))
			} else {
				// Check required parameters
				for _, v := range tmpl.Parameters {
					if wkfP, has := x.Parameters[v.Key]; (!has || len(wkfP) == 0) && v.Required {
						err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "required template parameter %s is required by template %s", v.Key, x.From))
					}
				}
			}
		default:
			sameVCS := x.Repository == nil || x.Repository.VCSServer == ef.currentVCS.Name || x.Repository.VCSServer == ""
			sameRepo := x.Repository == nil || x.Repository.Name == ef.currentRepo.Name || x.Repository.Name == ""
			if sameVCS && sameRepo && x.Repository != nil && x.Repository.InsecureSkipSignatureVerify {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "parameter `insecure-skip-signature-verify`is not allowed if the workflow is defined on the same repository as `workfow.repository.name`. "))
			}
		}
	}

	if len(err) > 0 {
		return err
	}

	return nil
}

func ReadEntityFile[T sdk.Lintable](ctx context.Context, api *API, directory, fileName string, content []byte, out *[]T, t string, analysis sdk.ProjectRepositoryAnalysis, ef *EntityFinder) ([]sdk.EntityWithObject, []error) {
	namePattern, err := regexp.Compile(sdk.EntityNamePattern)
	if err != nil {
		return nil, []error{sdk.WrapError(err, "unable to compile regexp %s", namePattern)}
	}

	if err := yaml.UnmarshalMultipleDocuments(content, out); err != nil {
		return nil, []error{sdk.NewErrorFrom(sdk.ErrInvalidData, "%s%s: %s", directory, fileName, err)}
	}
	var entities []sdk.EntityWithObject
	for _, o := range *out {
		if err := Lint(ctx, api, o, ef); err != nil {
			return nil, err
		}
		eo := sdk.EntityWithObject{
			Entity: sdk.Entity{
				Data:                string(content),
				Name:                o.GetName(),
				Ref:                 analysis.Ref,
				Commit:              analysis.Commit,
				ProjectKey:          analysis.ProjectKey,
				ProjectRepositoryID: analysis.ProjectRepositoryID,
				Type:                t,
				FilePath:            directory + fileName,
			},
		}
		if !namePattern.MatchString(o.GetName()) {
			return nil, []error{sdk.NewErrorFrom(sdk.ErrInvalidData, "name %s doesn't match %s", o.GetName(), sdk.EntityNamePattern)}
		}
		switch t {
		case sdk.EntityTypeWorkerModel:
			eo.Model = any(o).(sdk.V2WorkerModel)
			ef.workerModelCache[fmt.Sprintf("%s/%s/%s/%s@%s", analysis.ProjectKey, ef.currentVCS.Name, ef.currentRepo.Name, eo.Model.Name, analysis.Ref)] = eo
		case sdk.EntityTypeAction:
			eo.Action = any(o).(sdk.V2Action)
			ef.actionsCache[fmt.Sprintf("%s/%s/%s/%s@%s", analysis.ProjectKey, ef.currentVCS.Name, ef.currentRepo.Name, eo.Action.Name, analysis.Ref)] = eo.Action
		case sdk.EntityTypeWorkflow:
			eo.Workflow = any(o).(sdk.V2Workflow)
		case sdk.EntityTypeWorkflowTemplate:
			eo.Template = any(o).(sdk.V2WorkflowTemplate)
			ef.templatesCache[fmt.Sprintf("%s/%s/%s/%s@%s", analysis.ProjectKey, ef.currentVCS.Name, ef.currentRepo.Name, eo.Template.Name, analysis.Ref)] = eo
		}

		entities = append(entities, eo)
	}
	return entities, nil
}

// analyzeCommitSignatureThroughVcsAPI analyzes commit.
func (api *API) analyzeCommitSignatureThroughVcsAPI(ctx context.Context, analysis sdk.ProjectRepositoryAnalysis, vcsProject sdk.VCSProject, repoWithSecret sdk.ProjectRepository) (string, string, error) {
	var keyID, analyzesError string

	ctx, next := telemetry.Span(ctx, "api.analyzeCommitSignatureThroughVcsAPI")
	defer next()

	// Check commit signature
	client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, analysis.ProjectKey, vcsProject.Name)
	if err != nil {
		return keyID, analyzesError, err
	}
	vcsCommit, err := client.Commit(ctx, repoWithSecret.Name, analysis.Commit)
	if err != nil {
		return keyID, analyzesError, err
	}

	if vcsCommit.Hash == "" {
		return keyID, analyzesError, fmt.Errorf("commit %s not found", analysis.Commit)
	}
	keyID = vcsCommit.KeyID
	if keyID == "" {
		if vcsCommit.Signature != "" {
			keyID, err = gpg.GetKeyIdFromSignature(vcsCommit.Signature)
			if err != nil {
				return keyID, analyzesError, fmt.Errorf("unable to extract keyID from signature: %v", err)
			}
		} else {
			analyzesError = fmt.Sprintf("commit %s is not signed", vcsCommit.Hash)
		}
	}
	return keyID, analyzesError, nil
}

func (api *API) analyzeCommitSignatureThroughOperation(ctx context.Context, analysis *sdk.ProjectRepositoryAnalysis, vcsProject sdk.VCSProject, repoWithSecret sdk.ProjectRepository) (string, string, error) {
	var keyId, analyzeError string
	ctx, next := telemetry.Span(ctx, "api.analyzeCommitSignatureThroughOperation")
	defer next()
	if analysis.Data.OperationUUID == "" {
		proj, err := project.Load(ctx, api.mustDB(), analysis.ProjectKey, project.LoadOptions.WithClearKeys)
		if err != nil {
			return keyId, analyzeError, err
		}

		opts := sdk.OperationCheckout{
			Commit:         analysis.Commit,
			CheckSignature: true,
			ProcessSemver:  false,
			GetChangeSet:   false,
		}
		ope, err := operation.CheckoutAndAnalyzeOperation(ctx, api.mustDB(), *proj, vcsProject, repoWithSecret.Name, repoWithSecret.CloneURL, analysis.Ref, opts)
		if err != nil {
			return keyId, analyzeError, err
		}
		analysis.Data.OperationUUID = ope.UUID

		tx, err := api.mustDB().Begin()
		if err != nil {
			return keyId, analyzeError, sdk.WithStack(err)
		}

		if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
			return keyId, analyzeError, err
		}
		if err := tx.Commit(); err != nil {
			return keyId, analyzeError, sdk.WithStack(err)
		}
	}
	ope, err := operation.Poll(ctx, api.mustDB(), analysis.Data.OperationUUID)
	if err != nil {
		return keyId, analyzeError, err
	}

	if ope.Status == sdk.OperationStatusDone && ope.Setup.Checkout.Result.CommitVerified {
		keyId = ope.Setup.Checkout.Result.SignKeyID
	}
	if ope.Status == sdk.OperationStatusDone && !ope.Setup.Checkout.Result.CommitVerified {
		keyId = ope.Setup.Checkout.Result.SignKeyID
		analyzeError = ope.Setup.Checkout.Result.Msg
	}
	if ope.Status == sdk.OperationStatusError {
		return "", "", sdk.WithStack(fmt.Errorf("%s", ope.Error.Message))
	}
	return keyId, analyzeError, nil
}

func (api *API) getCdsFilesOnVCSDirectory(ctx context.Context, analysis *sdk.ProjectRepositoryAnalysis, vcsName, repoName, commit, directory string) (map[string][]byte, error) {
	ctx, next := telemetry.Span(ctx, "api.getCdsFilesOnVCSDirectory")
	defer next()

	client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, analysis.ProjectKey, vcsName)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	filesContent := make(map[string][]byte)
	contents, err := client.ListContent(ctx, repoName, commit, directory, "0", "100")
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list content on commit [%s] in directory %s: %v", commit, directory, err)
	}
	for _, c := range contents {
		if c.IsFile && (strings.HasSuffix(c.Name, ".yml") || strings.HasSuffix(c.Name, ".yaml")) {
			filePath := directory + "/" + c.Name
			vcsContent, err := client.GetContent(ctx, repoName, commit, filePath)
			if err != nil {
				return nil, err
			}
			contentBts, err := base64.StdEncoding.DecodeString(vcsContent.Content)
			if err != nil {
				return nil, sdk.WithStack(err)
			}
			filesContent[filePath] = contentBts
		}
		if c.IsDirectory {
			contents, err := api.getCdsFilesOnVCSDirectory(ctx, analysis, vcsName, repoName, commit, directory+"/"+c.Name)
			if err != nil {
				return nil, err
			}
			for k, v := range contents {
				filesContent[k] = v
			}
		}
	}
	return filesContent, nil
}

func (api *API) getCdsArchiveFileOnRepo(ctx context.Context, repo sdk.ProjectRepository, analysis *sdk.ProjectRepositoryAnalysis, vcsName string) (map[string][]byte, error) {
	ctx, next := telemetry.Span(ctx, "api.getCdsArchiveFileOnRepo")
	defer next()

	client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, analysis.ProjectKey, vcsName)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	filesContent := make(map[string][]byte)

	if _, err := client.ListContent(ctx, repo.Name, analysis.Commit, ".cds", "0", "1"); err != nil {
		if strings.Contains(err.Error(), "resource not found") {
			return filesContent, nil
		}
		return nil, sdk.WrapError(err, "unable to list content on commit [%s] in directory %s: %v", analysis.Commit, ".cds", err)
	}

	reader, _, err := client.GetArchive(ctx, repo.Name, ".cds", "tar.gz", analysis.Commit)
	if err != nil {
		return nil, err
	}

	gzf, err := gzip.NewReader(reader)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to read gzip file")
	}
	tarReader := tar.NewReader(gzf)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read tar file"))
		}

		dir, fileName := filepath.Split(hdr.Name)
		if !strings.HasSuffix(fileName, ".yml") && !strings.HasSuffix(fileName, ".yaml") {
			continue
		}
		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tarReader); err != nil {
			return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read tar file"))
		}
		filesContent[dir+fileName] = buff.Bytes()
	}
	return filesContent, nil
}

func (api *API) stopAnalysis(ctx context.Context, analysis *sdk.ProjectRepositoryAnalysis, originalErrors ...error) error {
	for _, e := range originalErrors {
		log.ErrorWithStackTrace(ctx, e)
	}
	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	analysis.Status = sdk.RepositoryAnalysisStatusError

	analysisErrors := make([]string, 0, len(originalErrors))
	for _, e := range originalErrors {
		analysisErrors = append(analysisErrors, sdk.ExtractHTTPError(e).From)
	}
	analysis.Data.Error = strings.Join(analysisErrors, ".\n")
	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
