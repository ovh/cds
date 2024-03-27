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
	user          *sdk.AuthentifiedUser
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
	return service.RBAC(api.projectRead),
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
						givenBranch, err := client.Branch(ctx, repo.Name, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(analysis.Ref, sdk.GitRefBranchPrefix)})
						if err != nil {
							return err
						}
						analysis.Commit = givenBranch.LatestCommit
					}
				} else if analysis.Ref == "" {
					defaultBranch, err := client.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true})
					if err != nil {
						return err
					}
					analysis.Ref = defaultBranch.ID
					analysis.Commit = defaultBranch.LatestCommit
				}
			}

			var u *sdk.AuthentifiedUser
			if !isHooks(ctx) {
				uc := getUserConsumer(ctx)
				if uc != nil {
					u = uc.AuthConsumerUser.AuthentifiedUser
				}
			} else if isHooks(ctx) && analysis.UserID != "" {
				u, err = user.LoadByID(ctx, api.mustDB(), analysis.UserID)
				if err != nil {
					return err
				}
			}

			createAnalysis := createAnalysisRequest{
				proj:          *proj,
				vcsProject:    *vcs,
				repo:          *repo,
				ref:           analysis.Ref,
				commit:        analysis.Commit,
				hookEventUUID: analysis.HookEventUUID,
				user:          u,
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
		},
	}
	if analysisRequest.user != nil {
		repoAnalysis.Data.CDSUserID = analysisRequest.user.ID
		repoAnalysis.Data.CDSUserName = analysisRequest.user.Username
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
		if err := sendAnalysisHookCallback(ctx, api.mustDB(), *analysis, entitiesUpdated, vcsProjectWithSecret.Type, vcsProjectWithSecret.Name, repo.Name); err != nil {
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
		case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud, sdk.VCSTypeGitlab:
			keyID, analysisError, err = api.analyzeCommitSignatureThroughOperation(ctx, analysis, *vcsProjectWithSecret, *repo)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check the commit signature"))
			}
		case sdk.VCSTypeGitea, sdk.VCSTypeGithub:
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

	// Transform file content into entities
	entities, multiErr := api.handleEntitiesFiles(ctx, filesContent, analysis)
	if multiErr != nil {
		return api.stopAnalysis(ctx, analysis, multiErr...)
	}

	userRoles := make(map[string]bool)
	skippedEntities := make([]sdk.EntityWithObject, 0)
	skippedFiles := make(sdk.StringSlice, 0)

	// For each entities, check user role for each entities
	entitiesToUpdate := make([]sdk.EntityWithObject, 0)
skipEntity:
	for i := range entities {
		e := &entities[i]

		// Check user role
		if _, has := userRoles[e.Type]; !has {
			roleName, err := sdk.GetManageRoleByEntity(e.Type)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, err)
			}
			b, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), roleName, analysis.Data.CDSUserID, analysis.ProjectKey)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check user permission"))
			}
			userRoles[e.Type] = b
			log.Info(ctx, "role: [%s] entity: [%s] has role: [%v]", roleName, e.Type, b)
		}

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
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to retrieve vcs_server %s on project %s", vcsProjectWithSecret.Name, analysis.ProjectKey))
	}
	defaultBranch, err := vcsAuthClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true})
	if err != nil {
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
			currentAnalysisBranch, err = vcsAuthClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(analysis.Ref, sdk.GitRefBranchPrefix)})
			if err != nil {
				return err
			}
		}
	}

	eventInsertedEntities := make([]sdk.Entity, 0)
	eventUpdatedEntities := make([]sdk.Entity, 0)

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint
	for i := range entitiesToUpdate {
		e := &entities[i]

		// Check if entity has changed from current HEAD
		var entityUpdated bool
		existingHeadEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, "HEAD")
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve latest entity with same name"))
		}
		if existingHeadEntity == nil || existingHeadEntity.Data != e.Entity.Data {
			entityUpdated = true
		}

		// If entity already exist, ignore it.
		existingEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, e.Commit)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check if %s of type %s already exist on git ref %s", e.Name, e.Type, e.Ref))
		}
		if existingEntity != nil {
			continue
		}

		// Insert new entity for current branch and commit
		if err := entity.Insert(ctx, tx, &e.Entity); err != nil {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save %s of type %s", e.Name, e.Type))
		}
		eventInsertedEntities = append(eventInsertedEntities, e.Entity)

		// If it's a new entity or an update, add it to the list of entity updated (for hooks)
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
			if err := manageWorkflowHooks(ctx, tx, *e, vcsProjectWithSecret.Name, repo.Name, defaultBranch); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, fmt.Sprintf("unable to create workflow hooks for %s", e.Name)))
			}
		}
	}

	// For skipped entities, retrieve definition on head or default branch
	for _, e := range skippedEntities {
		// Check if head entity exist
		existingHeadEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, "HEAD")
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve HEAD entity with same name on same branch"))
		}
		entityInserted := false
		if existingHeadEntity != nil {
			// Copy it for the current commit
			e.Data = existingHeadEntity.Data
			if err := entity.Insert(ctx, tx, &e.Entity); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save %s of type %s", e.Name, e.Type))
			}
			entityInserted = true
		} else {
			// If head entity do not exist, check default branch
			defaultBranchHeadEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, defaultBranch.ID, e.Type, e.Name, "HEAD")
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve latest entity on default branch with same name"))
			}
			if defaultBranchHeadEntity != nil {
				// Copy it for the current commit
				e.Data = defaultBranchHeadEntity.Data
				if err := entity.Insert(ctx, tx, &e.Entity); err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save %s of type %s", e.Name, e.Type))
				}
				entityInserted = true
			}
		}

		// Insert workflow hook
		if entityInserted && e.Type == sdk.EntityTypeWorkflow {
			if err := manageWorkflowHooks(ctx, tx, e, vcsProjectWithSecret.Name, repo.Name, defaultBranch); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, fmt.Sprintf("unable to create workflow hooks for %s", e.Name)))
			}
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
	} else {
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

	return nil
}

func manageWorkflowHooks(ctx context.Context, db gorpmapper.SqlExecutorWithTx, e sdk.EntityWithObject, workflowDefVCSName, workflowDefRepositoryName string, defaultBranch *sdk.VCSBranch) error {
	ctx, next := telemetry.Span(ctx, "manageWorkflowHooks")
	defer next()

	// Remove existing hook for the current branch
	if err := workflow_v2.DeleteWorkflowHooks(ctx, db, e.ID); err != nil {
		return err
	}

	if e.Workflow.On == nil {
		return nil
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
	if e.Workflow.On.Push != nil {
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
					RepositoryEvent: sdk.WorkflowHookEventPush,
					VCSServer:       targetVCS,
					RepositoryName:  targetRepository,
					BranchFilter:    e.Workflow.On.Push.Branches,
					TagFilter:       e.Workflow.On.Push.Tags,
					PathFilter:      e.Workflow.On.Push.Paths,
				},
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return err
			}

			if e.Commit == defaultBranch.LatestCommit {
				// Load existing head hook
				existingHook, err := workflow_v2.LoadHookHeadRepositoryWebHookByWorkflowAndEvent(ctx, db, e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, e.Name, sdk.WorkflowHookEventPush)
				if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return err
				}
				if existingHook != nil {
					// Update data and ref
					existingHook.Data = sdk.V2WorkflowHookData{
						RepositoryEvent: sdk.WorkflowHookEventPush,
						VCSServer:       targetVCS,
						RepositoryName:  targetRepository,
						BranchFilter:    e.Workflow.On.Push.Branches,
						TagFilter:       e.Workflow.On.Push.Tags,
						PathFilter:      e.Workflow.On.Push.Paths,
					}
					existingHook.Ref = e.Ref
					if err := workflow_v2.UpdateWorkflowHook(ctx, db, existingHook); err != nil {
						return err
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
							RepositoryEvent: sdk.WorkflowHookEventPush,
							VCSServer:       targetVCS,
							RepositoryName:  targetRepository,
							BranchFilter:    e.Workflow.On.Push.Branches,
							TagFilter:       e.Workflow.On.Push.Tags,
							PathFilter:      e.Workflow.On.Push.Paths,
						},
					}
					if err := workflow_v2.InsertWorkflowHook(ctx, db, &newHeadHook); err != nil {
						return err
					}
				}
			}
		}
	}

	if e.Workflow.On.PullRequest != nil {
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
					RepositoryEvent: sdk.WorkflowHookEventPullRequest,
					VCSServer:       targetVCS,
					RepositoryName:  targetRepository,
					BranchFilter:    e.Workflow.On.PullRequest.Branches,
					PathFilter:      e.Workflow.On.PullRequest.Paths,
					TypesFilter:     e.Workflow.On.PullRequest.Types,
				},
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return err
			}

			if e.Commit == defaultBranch.LatestCommit {
				// Load existing head hook
				existingHook, err := workflow_v2.LoadHookHeadRepositoryWebHookByWorkflowAndEvent(ctx, db, e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, e.Name, sdk.WorkflowHookEventPullRequest)
				if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return err
				}
				if existingHook != nil {
					// Update data and ref
					existingHook.Data = sdk.V2WorkflowHookData{
						RepositoryEvent: sdk.WorkflowHookEventPullRequest,
						VCSServer:       targetVCS,
						RepositoryName:  targetRepository,
						BranchFilter:    e.Workflow.On.PullRequest.Branches,
						PathFilter:      e.Workflow.On.PullRequest.Paths,
						TypesFilter:     e.Workflow.On.PullRequest.Types,
					}
					existingHook.Ref = e.Ref
					if err := workflow_v2.UpdateWorkflowHook(ctx, db, existingHook); err != nil {
						return err
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
							RepositoryEvent: sdk.WorkflowHookEventPullRequest,
							VCSServer:       targetVCS,
							RepositoryName:  targetRepository,
							BranchFilter:    e.Workflow.On.PullRequest.Branches,
							PathFilter:      e.Workflow.On.PullRequest.Paths,
							TypesFilter:     e.Workflow.On.PullRequest.Types,
						},
					}
					if err := workflow_v2.InsertWorkflowHook(ctx, db, &newHeadHook); err != nil {
						return err
					}
				}
			}
		}
	}

	if e.Workflow.On.PullRequestComment != nil {
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
					RepositoryEvent: sdk.WorkflowHookEventPullRequestComment,
					VCSServer:       targetVCS,
					RepositoryName:  targetRepository,
					BranchFilter:    e.Workflow.On.PullRequest.Branches,
					PathFilter:      e.Workflow.On.PullRequest.Paths,
					TypesFilter:     e.Workflow.On.PullRequest.Types,
				},
			}
			if err := workflow_v2.InsertWorkflowHook(ctx, db, &wh); err != nil {
				return err
			}
		}
	}

	// Only save workflow_update hook if :
	// * workflow.repository declaration != workflow.repository && default branch
	if e.Workflow.On.WorkflowUpdate != nil {
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
				return err
			}
		}
	}

	// Only save model_update hook if :
	// * workflow and model on the same repo
	// * workflow repo declaration != workflow.repository && default branch
	if e.Workflow.On.ModelUpdate != nil {
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
						return err
					}
				}
			}
		}
	}

	return nil
}

func sendAnalysisHookCallback(ctx context.Context, db *gorp.DbMap, analysis sdk.ProjectRepositoryAnalysis, entities []sdk.Entity, vcsServerType, vcsServerName, repoName string) error {
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
		VCSServerType:  vcsServerType,
		VCSServerName:  vcsServerName,
		HookEventUUID:  analysis.Data.HookEventUUID,
		AnalysisCallback: &sdk.HookAnalysisCallback{
			AnalysisStatus: analysis.Status,
			AnalysisID:     analysis.ID,
			Error:          analysis.Data.Error,
			Models:         make([]sdk.EntityFullName, 0),
			Workflows:      make([]sdk.EntityFullName, 0),
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
			if strings.HasPrefix(keys[j], ".cds/workflows/") {
				return true
			}
			return keys[i] < keys[j]
		}
		return keys[i] < keys[j]
	})
	return keys
}

func (api *API) handleEntitiesFiles(_ context.Context, filesContent map[string][]byte, analysis *sdk.ProjectRepositoryAnalysis) ([]sdk.EntityWithObject, []error) {
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
			es, err = ReadEntityFile(api, dir, fileName, content, &wms, sdk.EntityTypeWorkerModel, *analysis)
		case strings.HasPrefix(filePath, ".cds/actions/"):
			var actions []sdk.V2Action
			es, err = ReadEntityFile(api, dir, fileName, content, &actions, sdk.EntityTypeAction, *analysis)
		case strings.HasPrefix(filePath, ".cds/workflows/"):
			var w []sdk.V2Workflow
			es, err = ReadEntityFile(api, dir, fileName, content, &w, sdk.EntityTypeWorkflow, *analysis)
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

func Lint[T sdk.Lintable](api *API, o T) []error {
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
	}

	if len(err) > 0 {
		return err
	}

	return nil
}

func ReadEntityFile[T sdk.Lintable](api *API, directory, fileName string, content []byte, out *[]T, t string, analysis sdk.ProjectRepositoryAnalysis) ([]sdk.EntityWithObject, []error) {
	namePattern, err := regexp.Compile(sdk.EntityNamePattern)
	if err != nil {
		return nil, []error{sdk.WrapError(err, "unable to compile regexp %s", namePattern)}
	}

	if err := yaml.UnmarshalMultipleDocuments(content, out); err != nil {
		return nil, []error{sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to read %s%s: %v", directory, fileName, err)}
	}
	var entities []sdk.EntityWithObject
	for _, o := range *out {
		if err := o.Lint(); err != nil {
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
			return nil, []error{sdk.WrapError(sdk.ErrInvalidData, "name %s doesn't match %s", o.GetName(), sdk.EntityNamePattern)}
		}
		switch t {
		case sdk.EntityTypeWorkerModel:
			eo.Model = any(o).(sdk.V2WorkerModel)
		case sdk.EntityTypeAction:
			eo.Action = any(o).(sdk.V2Action)
		case sdk.EntityTypeWorkflow:
			eo.Workflow = any(o).(sdk.V2Workflow)
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
	if vcsCommit.Signature != "" {
		keyId, err := gpg.GetKeyIdFromSignature(vcsCommit.Signature)
		if err != nil {
			return keyID, analyzesError, fmt.Errorf("unable to extract keyID from signature: %v", err)
		} else {
			keyID = keyId
		}
	} else {
		analyzesError = fmt.Sprintf("commit %s is not signed", vcsCommit.Hash)
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
	contents, err := client.ListContent(ctx, repoName, commit, directory)
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
	analysis.Data.Error = strings.Join(analysisErrors, "\n")
	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
