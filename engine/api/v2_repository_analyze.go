package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) cleanRepositoryAnalysis(ctx context.Context, delay time.Duration) error {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			var analysis sdk.AnalysisRequest
			if err := service.UnmarshalBody(req, &analysis); err != nil {
				return err
			}

			proj, err := project.Load(ctx, api.mustDB(), analysis.ProjectKey, project.LoadOptions.WithClearKeys)
			if err != nil {
				return err
			}

			vcsProject, err := vcs.LoadVCSByProject(ctx, api.mustDB(), analysis.ProjectKey, analysis.VcsName)
			if err != nil {
				return err
			}

			repo, err := repository.LoadRepositoryByName(ctx, api.mustDB(), vcsProject.ID, analysis.RepoName)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			analyzeReponse, err := api.createAnalyze(ctx, tx, *proj, *vcsProject, *repo, analysis.Branch, analysis.Commit)
			if err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event.PublishProjectRepositoryAnalyze(ctx, proj.Key, vcsProject.ID, repo.ID, analyzeReponse.AnalysisID, analyzeReponse.Status)
			return service.WriteJSON(w, analyzeReponse, http.StatusCreated)
		}
}

func (api *API) createAnalyze(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, proj sdk.Project, vcsProject sdk.VCSProject, repo sdk.ProjectRepository, branch, commit string) (*sdk.AnalysisResponse, error) {
	ctx = context.WithValue(ctx, cdslog.VCSServer, vcsProject.Name)
	ctx = context.WithValue(ctx, cdslog.Repository, repo.Name)

	// Save analyze
	repoAnalysis := sdk.ProjectRepositoryAnalysis{
		Status:              sdk.RepositoryAnalysisStatusInProgress,
		ProjectRepositoryID: repo.ID,
		VCSProjectID:        vcsProject.ID,
		ProjectKey:          proj.Key,
		Branch:              branch,
		Commit:              commit,
		Data:                sdk.ProjectRepositoryData{},
	}

	if err := repository.InsertAnalysis(ctx, tx, &repoAnalysis); err != nil {
		return nil, err
	}

	response := sdk.AnalysisResponse{
		AnalysisID: repoAnalysis.ID,
		Status:     repoAnalysis.Status,
	}

	return &response, nil
}

func (api *API) repositoryAnalysisPoller(ctx context.Context, tick time.Duration) error {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
	defer func() {
		event.PublishProjectRepositoryAnalyze(ctx, analysis.ProjectKey, analysis.VCSProjectID, analysis.ProjectRepositoryID, analysis.ID, analysis.Status)
	}()

	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load analyze %s", analysis.ID))
	}

	if analysis.Status != sdk.RepositoryAnalysisStatusInProgress {
		return nil
	}

	vcsProjectWithSecret, err := vcs.LoadVCSByID(ctx, api.mustDB(), analysis.ProjectKey, analysis.VCSProjectID, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load vcs %s", analysis.VCSProjectID))
	}

	repo, err := repository.LoadRepositoryByID(ctx, api.mustDB(), analysis.ProjectRepositoryID)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to load repository %s", analysis.ProjectRepositoryID))
	}

	var keyID, analysisError string
	switch vcsProjectWithSecret.Type {
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud, sdk.VCSTypeGitlab:
		keyID, analysisError, err = api.analyzeCommitSignatureThroughOperation(ctx, analysis, *vcsProjectWithSecret, *repo)
		if err != nil {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check the commit signature"))
		}
	case sdk.VCSTypeGitea, sdk.VCSTypeGithub:
		keyID, analysisError, err = api.analyzeCommitSignatureThroughVcsAPI(ctx, analysis, *vcsProjectWithSecret, *repo)
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
	}

	var filesContent map[string][]byte
	var userID string
	if analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
		user, analysisStatus, analysisError, err := findCommitter(ctx, api.Cache, api.mustDB(), *analysis, *vcsProjectWithSecret, repo.Name, api.Config.VCS.GPGKeys)
		if err != nil {
			return api.stopAnalysis(ctx, analysis, err)
		}
		if user == nil {
			analysis.Status = analysisStatus
			analysis.Data.Error = analysisError
		}
		if user != nil {
			userID = user.ID
			analysis.Data.CDSUserID = user.ID
			analysis.Data.CDSUserName = user.Username

			if analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
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
			}
		}
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if len(filesContent) == 0 && analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
		analysis.Status = sdk.RepositoryAnalysisStatusSkipped
		analysis.Data.Error = "no cds files found"
	}

	if analysis.Status != sdk.RepositoryAnalysisStatusInProgress {
		if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
			return sdk.WrapError(err, "unable to update analysis")
		}
		return sdk.WithStack(tx.Commit())
	}

	entities, multiErr := api.handleEntitiesFiles(ctx, filesContent, analysis)
	if multiErr != nil {
		return api.stopAnalysis(ctx, analysis, multiErr...)
	}

	userRoles := make(map[string]bool)
	skippedFiles := make(sdk.StringSlice, 0)
skipEntity:
	for i := range entities {
		e := &entities[i]

		// Check user role
		if _, has := userRoles[e.Type]; !has {
			roleName, err := sdk.GetManageRoleByEntity(e.Type)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, err)
			}
			b, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), roleName, userID, analysis.ProjectKey)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check user permission"))
			}
			userRoles[e.Type] = b
		}

		for i := range analysis.Data.Entities {
			analysisEntity := &analysis.Data.Entities[i]
			if analysisEntity.Path+analysisEntity.FileName == e.FilePath {
				if userRoles[e.Type] {
					analysisEntity.Status = sdk.RepositoryAnalysisStatusSucceed
				} else {
					skippedFiles = append(skippedFiles, "User doesn't have the permission to manage "+e.Type)
					analysisEntity.Status = sdk.RepositoryAnalysisStatusSkipped
					continue skipEntity
				}
				break
			}
		}

		existingEntity, err := entity.LoadByBranchTypeName(ctx, tx, e.ProjectRepositoryID, e.Branch, e.Type, e.Name)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to check if %s of type %s already exist on branch %s", e.Name, e.Type, e.Branch))
		}
		if existingEntity == nil {
			if err := entity.Insert(ctx, tx, e); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, "unable to save %s of type %s", e.Name, e.Type))
			}
		} else {
			e.ID = existingEntity.ID
			if err := entity.Update(ctx, tx, e); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(err, fmt.Sprintf("unable to update %s of type %s", e.Name, e.Type)))
			}
		}
	}
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
	return sdk.WithStack(tx.Commit())
}

func findCommitter(ctx context.Context, cache cache.Store, db *gorp.DbMap, analysis sdk.ProjectRepositoryAnalysis, vcsProjectWithSecret sdk.VCSProject, repoName string, vcsPublicKeys map[string][]GPGKey) (*sdk.AuthentifiedUser, string, string, error) {
	publicKeyFound := false
	publicKeys, has := vcsPublicKeys[vcsProjectWithSecret.Name]
	if has {
		for _, k := range publicKeys {
			if analysis.Data.SignKeyID == k.ID {
				publicKeyFound = true
				break
			}
		}
	}

	if !publicKeyFound {
		gpgKey, err := user.LoadGPGKeyByKeyID(ctx, db, analysis.Data.SignKeyID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", sdk.NewErrorFrom(err, "unable get gpg key: %s", analysis.Data.SignKeyID)
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("gpgkey %s not found", analysis.Data.SignKeyID), nil
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

	// Get commit
	tx, err := db.Begin()
	if err != nil {
		return nil, "", "", sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	client, err := repositoriesmanager.AuthorizedClient(ctx, tx, cache, analysis.ProjectKey, vcsProjectWithSecret.Name)
	if err != nil {
		return nil, "", "", sdk.WithStack(err)
	}

	var commitUser *sdk.AuthentifiedUser

	switch vcsProjectWithSecret.Type {
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeGitlab:
		commit, err := client.Commit(ctx, repoName, analysis.Commit)
		if err != nil {
			return nil, "", "", err
		}
		commitUser, err = user.LoadByUsername(ctx, tx, commit.Committer.Slug)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", sdk.WithStack(sdk.NewErrorFrom(err, "unable to get user %s", commit.Committer.Slug))
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("committer %s not found in CDS", commit.Committer.Slug), nil
		}
	case sdk.VCSTypeGithub:
		pr, err := client.SearchPullRequest(ctx, repoName, analysis.Commit, "closed")
		if err != nil {
			return nil, "", "", sdk.WithStack(sdk.NewErrorFrom(err, "unable to retrieve pull request with commit %s", analysis.Commit))
		}

		userLink, err := link.LoadUserLinkByTypeAndExternalID(ctx, tx, vcsProjectWithSecret.Type, pr.MergeBy.ID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, "", "", err
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("%s user %s not found in CDS", vcsProjectWithSecret.Type, pr.MergeBy.Slug), nil
		}

		//
		if userLink.Username != pr.MergeBy.Slug {
			// Update user link
			userLink.Username = pr.MergeBy.Slug
			if err := link.Update(ctx, tx, userLink); err != nil {
				return nil, "", "", err
			}
		}

		commitUser, err = user.LoadByID(ctx, tx, userLink.AuthentifiedUserID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrUserNotFound) {
				return nil, "", "", err
			}
			return nil, sdk.RepositoryAnalysisStatusSkipped, fmt.Sprintf("committer %s not found in CDS", pr.MergeBy.Slug), nil
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, "", "", sdk.WithStack(err)
	}

	return commitUser, "", "", nil
}

func (api *API) handleEntitiesFiles(_ context.Context, filesContent map[string][]byte, analysis *sdk.ProjectRepositoryAnalysis) ([]sdk.Entity, []error) {
	entities := make([]sdk.Entity, 0)
	analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
	for filePath, content := range filesContent {
		dir, fileName := filepath.Split(filePath)
		var es []sdk.Entity
		var err sdk.MultiError
		switch {
		case strings.HasPrefix(filePath, ".cds/worker-models/"):
			var wms []sdk.V2WorkerModel
			es, err = sdk.ReadEntityFile(dir, fileName, content, &wms, sdk.EntityTypeWorkerModel, *analysis)
		case strings.HasPrefix(filePath, ".cds/actions/"):
			var actions []sdk.V2Action
			es, err = sdk.ReadEntityFile(dir, fileName, content, &actions, sdk.EntityTypeAction, *analysis)
		case strings.HasPrefix(filePath, ".cds/workflows/"):
			var w []sdk.V2Workflow
			es, err = sdk.ReadEntityFile(dir, fileName, content, &w, sdk.EntityTypeWorkflow, *analysis)
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

// analyzeCommitSignatureThroughVcsAPI analyzes commit.
func (api *API) analyzeCommitSignatureThroughVcsAPI(ctx context.Context, analysis *sdk.ProjectRepositoryAnalysis, vcsProject sdk.VCSProject, repoWithSecret sdk.ProjectRepository) (string, string, error) {
	var keyID, analyzesError string

	ctx, next := telemetry.Span(ctx, "api.analyzeCommitSignatureThroughVcsAPI")
	defer next()
	tx, err := api.mustDB().Begin()
	if err != nil {
		return keyID, analyzesError, sdk.WithStack(err)
	}

	// Check commit signature
	client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analysis.ProjectKey, vcsProject.Name)
	if err != nil {
		_ = tx.Rollback() // nolint
		return keyID, analyzesError, err
	}
	vcsCommit, err := client.Commit(ctx, repoWithSecret.Name, analysis.Commit)
	if err != nil {
		_ = tx.Rollback() // nolint
		return keyID, analyzesError, err
	}
	if err := tx.Commit(); err != nil {
		return keyID, analyzesError, sdk.WithStack(err)
	}

	if vcsCommit.Hash == "" {
		return keyID, analyzesError, fmt.Errorf("commit %s not found", analysis.Commit)
	} else {
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

		ope := &sdk.Operation{
			VCSServer:    vcsProject.Name,
			RepoFullName: repoWithSecret.Name,
			URL:          repoWithSecret.CloneURL,
			RepositoryStrategy: sdk.RepositoryStrategy{
				SSHKey:   vcsProject.Auth.SSHKeyName,
				User:     vcsProject.Auth.Username,
				Password: vcsProject.Auth.Token,
			},
			Setup: sdk.OperationSetup{
				Checkout: sdk.OperationCheckout{
					Commit:         analysis.Commit,
					Branch:         analysis.Branch,
					CheckSignature: true,
				},
			},
		}
		if vcsProject.Auth.SSHKeyName != "" {
			ope.RepositoryStrategy.ConnectionType = "ssh"
		} else {
			ope.RepositoryStrategy.ConnectionType = "https"
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return keyId, analyzeError, sdk.WithStack(err)
		}

		if err := operation.PostRepositoryOperation(ctx, tx, *proj, ope, nil); err != nil {
			return keyId, analyzeError, err
		}
		analysis.Data.OperationUUID = ope.UUID

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

	tx, err := api.mustDB().Begin()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analysis.ProjectKey, vcsName)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	filesContent := make(map[string][]byte)
	contents, err := client.ListContent(ctx, repoName, commit, directory)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list content on commit [%s] in directory %s: %v", commit, directory, err)
	}
	for _, c := range contents {
		if c.IsFile && strings.HasSuffix(c.Name, ".yml") {
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
	if err := tx.Commit(); err != nil {
		return nil, sdk.WithStack(err)
	}
	return filesContent, nil
}

func (api *API) getCdsArchiveFileOnRepo(ctx context.Context, repo sdk.ProjectRepository, analysis *sdk.ProjectRepositoryAnalysis, vcsName string) (map[string][]byte, error) {
	ctx, next := telemetry.Span(ctx, "api.getCdsArchiveFileOnRepo")
	defer next()

	tx, err := api.mustDB().Begin()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analysis.ProjectKey, vcsName)
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
		if !strings.HasSuffix(fileName, ".yml") {
			continue
		}
		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tarReader); err != nil {
			return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read tar file"))
		}
		filesContent[dir+fileName] = buff.Bytes()

	}
	if err := tx.Commit(); err != nil {
		return nil, sdk.WithStack(err)
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
