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
	initiator     *sdk.V2Initiator
}

func (api *API) cleanRepositoryAnalysis(ctx context.Context, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	maxAnalysis := api.Config.Entity.AnalysisRetention
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
				if nb > maxAnalysis {
					toDelete := int(nb - maxAnalysis)
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

			// analysis.Initiator and analysis.DeprecatedUserID can be nil
			if analysis.Initiator == nil && analysis.DeprecatedUserID != "" {
				analysis.Initiator = &sdk.V2Initiator{}
				analysis.Initiator.UserID = analysis.DeprecatedUserID
			}

			if analysis.Initiator != nil && analysis.Initiator.User == nil && analysis.Initiator.UserID != "" {
				u, err := user.LoadByID(ctx, api.mustDB(), analysis.Initiator.UserID, user.LoadOptions.WithContacts)
				if err != nil {
					return err
				}
				analysis.Initiator.User = u.Initiator()
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
			if !isHooks(ctx) {
				uc := getUserConsumer(ctx)
				u := uc.AuthConsumerUser.AuthentifiedUser
				isAdminMFA = isAdmin(ctx)
				analysis.Initiator = &sdk.V2Initiator{}
				analysis.Initiator.User = u.Initiator()
				analysis.Initiator.UserID = u.ID
				analysis.Initiator.IsAdminWithMFA = isAdminMFA
			} else if isHooks(ctx) && analysis.Initiator != nil && analysis.Initiator.UserID != "" {
				u, err := user.LoadByID(ctx, api.mustDB(), analysis.Initiator.UserID)
				if err != nil {
					return err
				}
				analysis.Initiator.User = u.Initiator()
				isAdminMFA = analysis.Initiator.IsAdminWithMFA
				if isAdminMFA && u.Ring != sdk.UserRingAdmin {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "user %s is not admin", u.Username)
				}
			}

			// At this point analysis.Initiator can still be nil if the event is from a vcs webhook
			createAnalysis := createAnalysisRequest{
				proj:          *proj,
				vcsProject:    *vcs,
				repo:          *repo,
				ref:           analysis.Ref,
				commit:        analysis.Commit,
				hookEventUUID: analysis.HookEventUUID,
				hookEventKey:  analysis.HookEventKey,
				initiator:     analysis.Initiator,
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
			Initiator:     analysisRequest.initiator,
		},
	}

	if analysisRequest.initiator != nil {
		repoAnalysis.Data.DeprecatedCDSUserID = analysisRequest.initiator.UserID
		repoAnalysis.Data.DeprecatedCDSUserName = analysisRequest.initiator.Username()
		repoAnalysis.Data.DeprecatedCDSAdminWithMFA = analysisRequest.initiator.IsAdminWithMFA
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

	log.Debug(ctx, "analysis initiator: %+v", analysis.Data.Initiator)

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

	defer func() {
		event_v2.PublishAnalysisDone(ctx, api.Cache, vcsProjectWithSecret.Name, repo.Name, analysis, analysis.Data.Initiator)
	}()

	entitiesUpdated := make([]sdk.Entity, 0)
	skippedEntities := make([]sdk.EntityWithObject, 0)
	skippedHooks := make([]sdk.V2WorkflowHook, 0)
	defer func() {
		if err := sendAnalysisHookCallback(ctx, api.mustDB(), *analysis, entitiesUpdated, skippedEntities, skippedHooks, vcsProjectWithSecret.Name, repo.Name); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	}()

	ctx = context.WithValue(ctx, cdslog.VCSServer, vcsProjectWithSecret.Name)
	ctx = context.WithValue(ctx, cdslog.Repository, repo.Name)

	// If no user triggered the analysis, retrieve the signing key
	if analysis.Data.Initiator.UserID == "" {
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
			commitInitiator, analysisStatus, analysisError, err := findCommitter(ctx, api.Cache, api.mustDB(), analysis.Commit, analysis.Data.SignKeyID, analysis.ProjectKey, *vcsProjectWithSecret, repo.Name, api.Config.VCS.GPGKeys)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, err)
			}
			if commitInitiator == nil {
				analysis.Status = analysisStatus
				analysis.Data.Error = analysisError
			} else {
				analysis.Data.Initiator = commitInitiator
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
	if analysis.Data.Initiator.UserID != "" && analysis.Data.Initiator.User == nil { // Should not happen
		usr, err := user.LoadByID(ctx, api.mustDB(), analysis.Data.Initiator.UserID, user.LoadOptions.WithContacts)
		if err != nil {
			return api.stopAnalysis(ctx, analysis, err)
		}
		analysis.Data.Initiator.User = usr.Initiator()
		// Compatibility code
		analysis.Data.DeprecatedCDSUserID = analysis.Data.Initiator.UserID
		analysis.Data.DeprecatedCDSUserName = analysis.Data.Initiator.Username()
		analysis.Data.DeprecatedCDSAdminWithMFA = analysis.Data.Initiator.IsAdminWithMFA
	}

	log.Debug(ctx, "analyzeRepository - analysis.Data.Initiator = %+v", analysis.Data.Initiator)

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

	ef, err := NewEntityFinder(ctx, api.mustDB(), proj.Key, analysis.Ref, analysis.Commit, *repo, *vcsProjectWithSecret, *analysis.Data.Initiator, api.Config.WorkflowV2.LibraryProjectKey)
	if err != nil {
		return err
	}

	// Transform file content into entities
	entities, multiErr := api.handleEntitiesFiles(ctx, ef, filesContent, analysis)
	if multiErr != nil {
		return api.stopAnalysis(ctx, analysis, multiErr...)
	}

	userRoles := make(map[string]bool)
	skippedFiles := make(sdk.StringSlice, 0)

	// Build user role map
	for _, t := range sdk.EntityTypes {
		if _, has := userRoles[t]; !has {
			roleName, err := sdk.GetManageRoleByEntity(t)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, err)
			}

			if analysis.Data.Initiator.IsUser() {
				hasRole, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), roleName, analysis.Data.Initiator.UserID, analysis.ProjectKey)
				if err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check user permission"))
				}
				userRoles[t] = analysis.Data.Initiator.IsAdminWithMFA || hasRole
				log.Info(ctx, "role: [%s] entity: [%s] has role: [%v]", roleName, t, b)
			} else {
				hasRole, err := rbac.HasRoleOnProjectAndVCSUser(ctx, api.mustDB(), roleName, sdk.RBACVCSUser{VCSServer: analysis.Data.Initiator.VCS, VCSUsername: analysis.Data.Initiator.VCSUsername}, analysis.ProjectKey)
				if err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check VCS User permission"))
				}
				userRoles[t] = hasRole
				log.Info(ctx, "role: [%s] entity: [%s] has role: [%v]", roleName, t, b)
			}
		}
	}

	// For each entities, check user role for each entities
	entitiesToUpdate := make([]sdk.EntityWithObject, 0)
skipEntity:
	for i := range entities {
		e := &entities[i]
		e.UserID = &analysis.Data.Initiator.UserID

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
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve tag %v on repository %s", strings.TrimPrefix(analysis.Ref, sdk.GitRefTagPrefix), repo.Name))
			}
		} else {
			currentAnalysisBranch, err = vcsAuthClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(analysis.Ref, sdk.GitRefBranchPrefix), NoCache: true})
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve branch %v on repository %s", strings.TrimPrefix(analysis.Ref, sdk.GitRefBranchPrefix), repo.Name))
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

		// Check if entity has changed from current HEAD. We need this check to be able to trigger workflow_update or model_update hook
		var entityUpdated bool
		existingHeadEntity, err := entity.LoadHeadEntityByRefTypeName(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to retrieve latest entity with same name"))
		}
		if existingHeadEntity == nil || existingHeadEntity.Data != e.Entity.Data {
			entityUpdated = true
		}

		// If entity already exist for this commit, ignore it.
		existingEntity, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, e.Commit)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check if %s of type %s already exist on git ref %s", e.Name, e.Type, e.Ref))
		}
		if existingEntity != nil {
			continue
		}

		// If current analysis is the latest commit of the branch
		if (currentAnalysisBranch != nil && currentAnalysisBranch.LatestCommit == e.Commit) || currentAnalysisTag.Sha == e.Commit {
			e.Entity.Head = true

			if existingHeadEntity != nil {
				existingHeadEntity.Head = false
				if err := entity.Update(ctx, tx, existingHeadEntity); err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to change head entity for %s [%s] on ref %s for commit %s", e.Name, e.Type, e.Ref, e.Commit))
				}

				// Remove head on hooks
				if e.Type == sdk.EntityTypeWorkflow {
					hooks, err := workflow_v2.LoadHooksByEntityID(ctx, tx, existingHeadEntity.ID)
					if err != nil {
						return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load hooks on previous head definition %s [%s] on ref %s for commit %s", e.Name, e.Type, e.Ref, e.Commit))
					}
					for _, h := range hooks {
						h.Head = false
						if err := workflow_v2.UpdateWorkflowHook(ctx, tx, &h); err != nil {
							return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to update hooks on previous head definition %s [%s] on ref %s for commit %s", e.Name, e.Type, e.Ref, e.Commit))
						}
					}
				}
			}

			// //////// Remove old CDS resource with commit=HEAD
			if e.Type == sdk.EntityTypeWorkflow {
				// Remove old entity
				entityWithCommitHEAD, err := entity.LoadByRefTypeNameCommit(ctx, tx, e.ProjectRepositoryID, e.Ref, e.Type, e.Name, "HEAD")
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
				if entityWithCommitHEAD != nil {
					if err := entity.Delete(ctx, tx, entityWithCommitHEAD); err != nil {
						log.ErrorWithStackTrace(ctx, err)
					}
				}

				// Remove old head hooks
				headHooks, err := workflow_v2.LoadOldHeadHooksByVCSAndRepoAndRefAndWorkflow(ctx, tx, repo.ProjectKey, vcsProjectWithSecret.Name, repo.Name, e.Ref, e.Name)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
				for _, h := range headHooks {
					if err := workflow_v2.DeleteWorkflowHookByID(ctx, tx, h.ID); err != nil {
						log.ErrorWithStackTrace(ctx, err)
					}
				}

			}

			//////////

		}

		// Insert new entity for current branch and commit
		if err := entity.Insert(ctx, tx, &e.Entity); err != nil {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save %s of type %s for ref %s and commit %s", e.Name, e.Type, e.Ref, e.Commit))
		}
		eventInsertedEntities = append(eventInsertedEntities, e.Entity)

		// If it's an update, add it to the list of entity updated (for hooks)
		if entityUpdated {
			entitiesUpdated = append(entitiesUpdated, e.Entity)
		}

		// Insert workflow hook
		if e.Type == sdk.EntityTypeWorkflow {
			hooks, err := manageWorkflowHooks(ctx, tx, api.Cache, ef, *e, vcsProjectWithSecret.Name, repo.Name, defaultBranch, srvs)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to create workflow hooks for %s", e.Name))
			}
			newHooks = append(newHooks, hooks...)
		}
	}

	for i := range skippedEntities {
		e := skippedEntities[i]
		if e.Entity.Type != sdk.EntityTypeWorkflow {
			continue
		}
		hooks, err := prepareWorkflowHooks(ctx, tx, api.Cache, ef, e, vcsProjectWithSecret.Name, repo.Name, defaultBranch)
		if err != nil {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to create workflow hooks for %s", e.Name))
		}
		skippedHooks = append(skippedHooks, hooks...)
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

		// Load existing entity on the current branch
		existingEntities, err := entity.LoadHeadEntitiesByRepositoryAndRef(ctx, api.mustDB(), analysis.ProjectRepositoryID, analysis.Ref)
		if err != nil {
			return api.stopAnalysis(ctx, analysis, err)
		}

		for _, e := range existingEntities {
			// If an existing entities has not been found in the current head commit (deleted or renamed)
			// => remove the entity
			if _, has := foundEntities[e.Type+"-"+e.Name]; !has {
				// Check right
				if !userRoles[e.Type] {
					log.Warn(ctx, "user %s removed the entity %s [%s] but has not the right to do it", analysis.Data.Initiator.Username(), e.Name, e.Type)
					continue
				}
				log.Info(ctx, "deleting entity %s of type %s: file doesn't exist anymore", e.Name, e.Type)
				if err := DeleteEntity(ctx, tx, &e, srvs, delOpts); err != nil {
					return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to delete entity %s [%s] ", e.Name, e.Type))
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
		event_v2.PublishEntityEvent(ctx, api.Cache, sdk.EventEntityCreated, repo.Name, repo.Name, e, analysis.Data.Initiator)
	}
	for _, eUpdated := range eventUpdatedEntities {
		event_v2.PublishEntityEvent(ctx, api.Cache, sdk.EventEntityUpdated, vcsProjectWithSecret.Name, repo.Name, eUpdated, analysis.Data.Initiator)
	}
	for _, eRemoved := range eventRemovedEntities {
		event_v2.PublishEntityEvent(ctx, api.Cache, sdk.EventEntityDeleted, vcsProjectWithSecret.Name, repo.Name, eRemoved, analysis.Data.Initiator)
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

func prepareWorkflowHooks(ctx context.Context, db gorpmapper.SqlExecutorWithTx, cache cache.Store, ef *EntityFinder, e sdk.EntityWithObject, workflowDefVCSName, workflowDefRepositoryName string, defaultBranch *sdk.VCSBranch) ([]sdk.V2WorkflowHook, error) {
	ctx, next := telemetry.Span(ctx, "prepareWorkflowHooks")
	defer next()

	whs := make([]sdk.V2WorkflowHook, 0)

	// If there is a workflow template and non hooks on workflow, check the workflow template
	if e.Workflow.From != "" && e.Workflow.On == nil {
		entTemplate, _, msg, err := ef.searchWorkflowTemplate(ctx, db, cache, e.Workflow.From)
		if err != nil {
			return nil, err
		}
		if msg != "" {
			return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "%s", msg)
		}
		if _, err := entTemplate.Template.Resolve(ctx, &e.Workflow); err != nil {
			return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to compute workflow from template: %v", err)
		}
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

	// Prepare repository push event hook
	// 1. workflow + code on same repo: create hook
	// 2. workflow distant: create only on default branch
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
				Head: e.Head,
			}
			if e.Workflow.Repository != nil {
				wh.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
			}
			whs = append(whs, wh)
		}
	}

	// Prepare repository pr hook
	// 1. workflow + code on same repo: create hook
	// 2. workflow distant: create only on default branch
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
				Head: e.Head,
			}
			if e.Workflow.Repository != nil {
				wh.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
			}
			whs = append(whs, wh)
		}
	}

	// Prepare repository pr_comment hook
	// 1. workflow + code on same repo: create hook
	// 2. workflow distant: create only on default branch
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
				Head: e.Head,
			}
			if e.Workflow.Repository != nil {
				wh.Data.InsecureSkipSignatureVerify = e.Workflow.Repository.InsecureSkipSignatureVerify
			}
			whs = append(whs, wh)
		}
	}

	// Prepare workflow_update hook:
	// * workflow distant && default branch
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
				Head: e.Head,
			}
			whs = append(whs, wh)
		}
	}

	// Prepare model_update hook:
	// * distant workflow && model on the same repo && default branch
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
						Head: e.Head,
					}
					whs = append(whs, wh)
				}
			}
		}
	}

	// Prepare scheduler
	// default branch && latest commit
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
				Head: e.Head,
			}
			whs = append(whs, wh)
		}
	}

	// Prepare workflow_run hook
	// default branch && latest commit
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

			// Add in data destination vcs / repo
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
				Head: e.Head,
			}
			whs = append(whs, wh)
		}

	}

	return whs, nil
}

func manageWorkflowHooks(ctx context.Context, db gorpmapper.SqlExecutorWithTx, cache cache.Store, ef *EntityFinder, e sdk.EntityWithObject, workflowDefVCSName, workflowDefRepositoryName string, defaultBranch *sdk.VCSBranch, hookSrvs []sdk.Service) ([]sdk.V2WorkflowHook, error) {
	ctx, next := telemetry.Span(ctx, "manageWorkflowHooks")
	defer next()

	// Remove existing scheduler hook for the current branch
	if e.Ref == defaultBranch.ID && e.Commit == defaultBranch.LatestCommit {
		// Search old scheduler definition and remove them from hooks uservice
		whs, err := workflow_v2.LoadHookByWorkflowAndType(ctx, db, e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, e.Name, sdk.WorkflowHookTypeScheduler)
		if err != nil {
			return nil, err
		}
		if len(whs) > 0 {
			if err := DeleteAllEntitySchedulerHook(ctx, db, whs[0].VCSName, whs[0].RepositoryName, whs[0].WorkflowName, hookSrvs); err != nil {
				return nil, err
			}
		}

		// Remove previous hooks workflow_run
		whs, err = workflow_v2.LoadHookByWorkflowAndType(ctx, db, e.ProjectKey, workflowDefVCSName, workflowDefRepositoryName, e.Name, sdk.WorkflowHookTypeWorkflowRun)
		if err != nil {
			return nil, err
		}
		for _, wh := range whs {
			if err := workflow_v2.DeleteWorkflowHookByID(ctx, db, wh.ID); err != nil {
				return nil, err
			}
		}
	}

	hooks, err := prepareWorkflowHooks(ctx, db, cache, ef, e, workflowDefVCSName, workflowDefRepositoryName, defaultBranch)
	if err != nil {
		return nil, err
	}
	for i := range hooks {
		wh := &hooks[i]
		if wh.ID == "" {
			if err := workflow_v2.InsertWorkflowHook(ctx, db, wh); err != nil {
				return nil, err
			}
		} else {
			if err := workflow_v2.UpdateWorkflowHook(ctx, db, wh); err != nil {
				return nil, err
			}
		}
	}
	return hooks, nil
}

func sendAnalysisHookCallback(ctx context.Context, db *gorp.DbMap, analysis sdk.ProjectRepositoryAnalysis, entities []sdk.Entity, skippedEntity []sdk.EntityWithObject, skippedHooks []sdk.V2WorkflowHook, vcsServerName, repoName string) error {
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
		},
	}

	if analysis.Data.Initiator != nil {
		callback.AnalysisCallback.Initiator = analysis.Data.Initiator
		callback.AnalysisCallback.DeprecatedUsername = analysis.Data.Initiator.Username()
		callback.AnalysisCallback.DeprecatedUserID = analysis.Data.Initiator.UserID
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
	for _, e := range skippedEntity {
		ent := sdk.EntityFullName{
			ProjectKey: e.ProjectKey,
			VCSName:    vcsServerName,
			RepoName:   repoName,
			Name:       e.Name,
			Ref:        e.Ref,
		}
		switch e.Type {
		case sdk.EntityTypeWorkflow:
			callback.AnalysisCallback.SkippedWorkflows = append(callback.AnalysisCallback.SkippedWorkflows, ent)
		}
	}
	callback.AnalysisCallback.SkippedHooks = append(callback.AnalysisCallback.SkippedHooks, skippedHooks...)

	if _, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/v2/repository/event/callback", callback, nil); err != nil {
		return sdk.WrapError(err, "unable to send analysis call to  hook [HTTP: %d]", code)
	}
	return nil
}

func findCommitter(ctx context.Context, cache cache.Store, db *gorp.DbMap, sha, signKeyID, projKey string, vcsProjectWithSecret sdk.VCSProject, repoName string, vcsPublicKeys map[string][]GPGKey) (*sdk.V2Initiator, string, string, error) {
	ctx, next := telemetry.Span(ctx, "findCommitter", trace.StringAttribute(telemetry.TagProjectKey, projKey), trace.StringAttribute(telemetry.TagVCSServer, vcsProjectWithSecret.Name), trace.StringAttribute(telemetry.TagRepository, repoName))
	defer next()

	publicKeyFound := false

	// Checking known public gpg keys from vcs server (from configuration)
	publicKeys, has := vcsPublicKeys[vcsProjectWithSecret.Name]
	if has {
		for _, k := range publicKeys {
			if signKeyID == k.ID {
				publicKeyFound = true
				break
			}
		}
	}

	// Checking CDS users public gpg keys
	var userGPGKey *sdk.UserGPGKey
	if !publicKeyFound {
		var err error
		userGPGKey, err = user.LoadGPGKeyByKeyID(ctx, db, signKeyID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, sdk.RepositoryAnalysisStatusError, "", sdk.NewErrorFrom(err, "unable get gpg key: %s", signKeyID)
			}
		}
	}

	// Is the GPG Key matches a CDS User, load the User
	var cdsUser *sdk.AuthentifiedUser
	if userGPGKey != nil {
		var err error
		cdsUser, err = user.LoadByID(ctx, db, userGPGKey.AuthentifiedUserID, user.LoadOptions.WithContacts)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", sdk.WithStack(sdk.NewErrorFrom(err, "unable to load user %s", userGPGKey.AuthentifiedUserID))
			}
			return nil, sdk.RepositoryAnalysisStatusError, fmt.Sprintf("user %s not found for gpg key %s", userGPGKey.AuthentifiedUserID, userGPGKey.KeyID), nil
		}

		return &sdk.V2Initiator{
			UserID: cdsUser.ID,
			User:   cdsUser.Initiator(),
		}, "", "", nil
	}

	client, err := repositoriesmanager.AuthorizedClient(ctx, db, cache, projKey, vcsProjectWithSecret.Name)
	if err != nil {
		return nil, sdk.RepositoryAnalysisStatusError, "", sdk.WithStack(err)
	}

	var possibleVCSGPGUSers []sdk.VCSUserGPGKey
	if !publicKeyFound {
		k, _ := project.LoadKeyByLongKeyID(ctx, db, signKeyID)
		if k == nil {
			// The key is not known from CDS. Let's raise an error
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("gpg key %s not found in CDS", signKeyID), nil
		}

		projWhoOwnTheKey, err := project.LoadByID(db, k.ProjectID)
		if err != nil {
			return nil, sdk.RepositoryAnalysisStatusError, "", err
		}

		allvcs, err := vcs.LoadAllVCSByProject(ctx, db, projWhoOwnTheKey.Key, gorpmapper.GetAllOptions.WithDecryption)
		if err != nil {
			return nil, sdk.RepositoryAnalysisStatusError, "", err
		}

		var selectedVCS []sdk.VCSProject
		for _, v := range allvcs {
			v.Auth.Token = "" // we are sure we don't need this
			v.Auth.SSHPrivateKey = ""

			log.Debug(ctx, "%s = %s", v.Auth.GPGKeyName, k.Name)
			if v.Auth.GPGKeyName == k.Name {
				selectedVCS = append(selectedVCS, v)
			}
		}

		for _, vcs := range selectedVCS {
			possibleVCSGPGUSers = append(possibleVCSGPGUSers, sdk.VCSUserGPGKey{
				ProjectKey:     projWhoOwnTheKey.Key,
				VCSProjectName: vcs.Name,
				Username:       vcs.Auth.Username,
				KeyName:        vcs.Auth.GPGKeyName,
				KeyID:          k.LongKeyID,
				PublicKey:      k.Public,
			})
		}
	}

	// Get commit
	tx, err := db.Begin()
	if err != nil {
		return nil, sdk.RepositoryAnalysisStatusError, "", sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	var committer string

	switch vcsProjectWithSecret.Type {
	case sdk.VCSTypeGitea:
		commit, err := client.Commit(ctx, repoName, sha)
		if err != nil {
			return nil, sdk.RepositoryAnalysisStatusError, "", err
		}
		committer = commit.Committer.DisplayName
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeGitlab:
		commit, err := client.Commit(ctx, repoName, sha)
		if err != nil {
			return nil, sdk.RepositoryAnalysisStatusError, "", err
		}
		committer = commit.Committer.Slug
	case sdk.VCSTypeGithub:
		commit, err := client.Commit(ctx, repoName, sha)
		if err != nil {
			return nil, sdk.RepositoryAnalysisStatusError, "", err
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
		committer = commit.Committer.Name

		// Load user
		cdsUser, err = user.LoadByID(ctx, tx, userLink.AuthentifiedUserID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrUserNotFound) {
				return nil, sdk.RepositoryAnalysisStatusError, "", err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.RepositoryAnalysisStatusError, "", sdk.WithStack(err)
	}

	// First search among possibleVCSGPGUSers
	for _, VCSGPGUser := range possibleVCSGPGUSers {
		if VCSGPGUser.Username == committer {
			return &sdk.V2Initiator{
				VCS:         vcsProjectWithSecret.Name,
				VCSUsername: VCSGPGUser.Username,
			}, "", "", nil
		}
	}

	// Then try to load a CDS user from the committer name
	if cdsUser == nil { // Committer can be not nil in GitHub usescases
		cdsUser, err = user.LoadByUsername(ctx, db, committer, user.LoadOptions.WithContacts)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrUserNotFound) {
				return nil, sdk.RepositoryAnalysisStatusError, "", sdk.WithStack(sdk.NewErrorFrom(err, "unable to get user %s", committer))
			}
		}
	}

	if cdsUser != nil {
		return &sdk.V2Initiator{
			UserID: cdsUser.ID,
			User:   cdsUser.Initiator(),
		}, "", "", nil
	}

	// This error should not happen
	return nil, sdk.RepositoryAnalysisStatusSkipped, "Unknown committer", nil
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
		for _, newEntity := range es {
			for _, alreadyExistEntity := range entities {
				if newEntity.Name == alreadyExistEntity.Name && newEntity.Type == alreadyExistEntity.Type {
					return nil, []error{sdk.NewErrorFrom(sdk.ErrInvalidData, "there is at least 2 %s with the name %s", newEntity.Type, newEntity.Name)}
				}
			}
		}

		entities = append(entities, es...)
		analysis.Data.Entities = append(analysis.Data.Entities, sdk.ProjectRepositoryDataEntity{
			FileName: fileName,
			Path:     dir,
		})
	}
	return entities, nil

}

func Lint[T sdk.Lintable](ctx context.Context, db *gorp.DbMap, store cache.Store, o T, ef *EntityFinder, wmDockerImageWhiteList []regexp.Regexp) []error {
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
		if dockerSpec.Image != "" && len(wmDockerImageWhiteList) > 0 {
			var allowedImage = false
			for _, r := range wmDockerImageWhiteList { // At least one regexp must match
				if r.MatchString(dockerSpec.Image) {
					allowedImage = true
					break
				}
			}
			if !allowedImage {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "worker model %s: image %q is not allowed", x.Name, dockerSpec.Image))
			}
		}
	case sdk.V2Workflow:
		switch {
		case x.From != "":
			entTmpl, _, msg, errSearch := ef.searchWorkflowTemplate(ctx, db, store, x.From)
			if errSearch != nil {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "workflow %s: unable to retrieve template %s of type %s: %v", x.Name, x.From, sdk.EntityTypeWorkflowTemplate, errSearch))
				break
			}
			if msg != "" {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "workflow %s: %s", x.Name, msg))
				break
			}
			if entTmpl == nil || entTmpl.Template.Name == "" {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "workflow %s: unknown workflow template %s", x.Name, x.From))
			}
			// Check required parameters
			for _, v := range entTmpl.Template.Parameters {
				if wkfP, has := x.Parameters[v.Key]; (!has || len(wkfP) == 0) && v.Required {
					err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "workflow %s: required template parameter %s is missing or empty", x.Name, x.From))
				}
			}

		default:
			sameVCS := x.Repository == nil || x.Repository.VCSServer == ef.currentVCS.Name || x.Repository.VCSServer == ""
			sameRepo := x.Repository == nil || x.Repository.Name == ef.currentRepo.Name || x.Repository.Name == ""
			if sameVCS && sameRepo && x.Repository != nil && x.Repository.InsecureSkipSignatureVerify {
				err = append(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "workflow %s: parameter `insecure-skip-signature-verify`is not allowed if the workflow is defined on the same repository as `workflow.repository.name`. ", x.Name))
			}
			for jobID, j := range x.Jobs {
				// Check if worker model exists
				if !strings.Contains(j.RunsOn.Model, "${{") && len(j.Steps) > 0 {
					_, _, msg, errSearch := ef.searchWorkerModel(ctx, db, store, j.RunsOn.Model)
					if errSearch != nil {
						err = append(err, errSearch)
					}
					if msg != "" {
						err = append(err, sdk.NewErrorFrom(sdk.ErrInvalidData, "workflow %s: %s", x.Name, msg))
					}
				}

				// Check if actions/plugins exists
				for _, s := range j.Steps {
					if s.Uses == "" {
						continue
					}
					_, _, msg, errSearch := ef.searchAction(ctx, db, store, s.Uses)
					if errSearch != nil {
						err = append(err, errSearch)
					}
					if msg != "" {
						err = append(err, sdk.NewErrorFrom(sdk.ErrInvalidData, "%s", msg))
					}
				}

				// Check concurrency
				if j.Concurrency != "" {
					found := false
					for _, c := range x.Concurrencies {
						if j.Concurrency == c.Name {
							found = true
							break
						}
					}
					if !found {
						// Check if there is interpolation
						if !strings.Contains(j.Concurrency, "${{") {
							if _, errC := project.LoadConcurrencyByNameAndProjectKey(ctx, db, ef.currentProject, j.Concurrency); errC != nil {
								if sdk.ErrorIs(errC, sdk.ErrNotFound) {
									err = append(err, sdk.NewErrorFrom(sdk.ErrInvalidData, "workflow %s job %s: concurrency %s doesn't exist", x.Name, jobID, j.Concurrency))
								} else {
									log.ErrorWithStackTrace(ctx, errC)
									err = append(err, sdk.NewErrorFrom(sdk.ErrUnknownError, "workflow %s job %s: unable to check if concurrency %s exists", x.Name, jobID, j.Concurrency))
								}
							}
						}
					}
				}

				if x.Concurrency != "" {
					found := false
					for _, c := range x.Concurrencies {
						if x.Concurrency == c.Name {
							found = true
							break
						}
					}
					if !found {
						// Check if there is interpolation
						if !strings.Contains(x.Concurrency, "${{") {
							if _, errC := project.LoadConcurrencyByNameAndProjectKey(ctx, db, ef.currentProject, x.Concurrency); errC != nil {
								if sdk.ErrorIs(errC, sdk.ErrNotFound) {
									err = append(err, sdk.NewErrorFrom(sdk.ErrInvalidData, "workflow %s: concurrency %s doesn't exist", x.Name, x.Concurrency))
								} else {
									log.ErrorWithStackTrace(ctx, errC)
									err = append(err, sdk.NewErrorFrom(sdk.ErrUnknownError, "workflow %s: unable to check if concurrency %s exists", x.Name, x.Concurrency))
								}
							}
						}
					}
				}
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
		if err := Lint(ctx, api.mustDB(), api.Cache, o, ef, api.WorkerModelDockerImageWhiteList); err != nil {
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
			ef.localWorkerModelCache[eo.Entity.FilePath] = eo
			ef.workerModelCache[fmt.Sprintf("%s/%s/%s/%s@%s", analysis.ProjectKey, ef.currentVCS.Name, ef.currentRepo.Name, eo.Model.Name, analysis.Ref)] = eo
		case sdk.EntityTypeAction:
			eo.Action = any(o).(sdk.V2Action)
			ef.localActionsCache[eo.Entity.FilePath] = eo
			ef.actionsCache[fmt.Sprintf("%s/%s/%s/%s@%s", analysis.ProjectKey, ef.currentVCS.Name, ef.currentRepo.Name, eo.Action.Name, analysis.Ref)] = eo
		case sdk.EntityTypeWorkflow:
			eo.Workflow = any(o).(sdk.V2Workflow)
		case sdk.EntityTypeWorkflowTemplate:
			eo.Template = any(o).(sdk.V2WorkflowTemplate)
			ef.localTemplatesCache[eo.Entity.FilePath] = eo
			ef.templatesCache[fmt.Sprintf("%s/%s/%s/%s@%s", analysis.ProjectKey, ef.currentVCS.Name, ef.currentRepo.Name, eo.Template.Name, analysis.Ref)] = eo
		}

		entities = append(entities, eo)
	}
	return entities, nil
}

// analyzeCommitSignatureThroughVcsAPI analyzes commit.
func (api *API) analyzeCommitSignatureThroughVcsAPI(ctx context.Context, analysis sdk.ProjectRepositoryAnalysis, vcsProject sdk.VCSProject, repoWithSecret sdk.ProjectRepository) (string, string, error) {
	var keyID, signature, analyzesError string
	ctx, next := telemetry.Span(ctx, "api.analyzeCommitSignatureThroughVcsAPI")
	defer next()

	// Check commit signature
	client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, analysis.ProjectKey, vcsProject.Name)
	if err != nil {
		return keyID, analyzesError, err
	}

	switch {
	case strings.HasPrefix(analysis.Ref, sdk.GitRefTagPrefix) && (vcsProject.Type == sdk.VCSTypeGithub):
		tag, err := client.Tag(ctx, repoWithSecret.Name, strings.TrimPrefix(analysis.Ref, sdk.GitRefTagPrefix))
		if err != nil {
			return keyID, analyzesError, err
		}
		signature = tag.Signature
	default:
		vcsCommit, err := client.Commit(ctx, repoWithSecret.Name, analysis.Commit)
		if err != nil {
			return keyID, analyzesError, err
		}
		if vcsCommit.Hash == "" {
			return keyID, analyzesError, sdk.WithStack(fmt.Errorf("commit %s not found", analysis.Commit))
		}
		keyID = vcsCommit.KeyID
		signature = vcsCommit.Signature
	}

	if keyID == "" {
		if signature != "" {
			keyID, err = gpg.GetKeyIdFromSignature(signature)
			if err != nil {
				return keyID, analyzesError, fmt.Errorf("unable to extract keyID from signature: %v", err)
			}
		} else {
			analyzesError = fmt.Sprintf("commit %s is not signed", analysis.Commit)
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
