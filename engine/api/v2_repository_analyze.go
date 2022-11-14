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

	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/cache"
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
	return service.RBAC(rbac.ProjectRead),
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
	return service.RBAC(rbac.ProjectRead),
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
	return service.RBAC(rbac.IsHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			var analysis sdk.AnalysisRequest
			if err := service.UnmarshalBody(req, &analysis); err != nil {
				return err
			}

			ctx = context.WithValue(ctx, cdslog.VCSServer, analysis.VcsName)
			ctx = context.WithValue(ctx, cdslog.Repository, analysis.RepoName)

			proj, err := project.Load(ctx, api.mustDB(), analysis.ProjectKey, project.LoadOptions.WithClearKeys)
			if err != nil {
				return err
			}

			vcsProject, err := vcs.LoadVCSByProject(ctx, api.mustDB(), analysis.ProjectKey, analysis.VcsName)
			if err != nil {
				return err
			}

			clearRepo, err := repository.LoadRepositoryByName(ctx, api.mustDB(), vcsProject.ID, analysis.RepoName, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WrapError(err, "unable to start db transaction")
			}
			defer tx.Rollback() // nolint

			// Save analyze
			repoAnalysis := sdk.ProjectRepositoryAnalysis{
				Status:              sdk.RepositoryAnalysisStatusInProgress,
				ProjectRepositoryID: clearRepo.ID,
				VCSProjectID:        vcsProject.ID,
				ProjectKey:          proj.Key,
				Branch:              analysis.Branch,
				Commit:              analysis.Commit,
				Data:                sdk.ProjectRepositoryData{},
			}

			if err := repository.InsertAnalysis(ctx, tx, &repoAnalysis); err != nil {
				return err
			}

			response := sdk.AnalysisResponse{
				AnalysisID: repoAnalysis.ID,
			}

			if err := tx.Commit(); err != nil {
				return sdk.WrapError(err, "unable to commit transaction")
			}
			return service.WriteJSON(w, &response, http.StatusCreated)
		}
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
		return api.stopAnalysis(ctx, analysis, sdk.WrapError(err, "unable to load analyze %s", analysis.ID))
	}

	if analysis.Status != sdk.RepositoryAnalysisStatusInProgress {
		return nil
	}

	vcsProject, err := vcs.LoadVCSByID(ctx, api.mustDB(), analysis.ProjectKey, analysis.VCSProjectID)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, err)
	}

	repoWithSecret, err := repository.LoadRepositoryByID(ctx, api.mustDB(), analysis.ProjectRepositoryID, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return api.stopAnalysis(ctx, analysis, err)
	}

	var keyID, analysisError string
	switch vcsProject.Type {
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud, sdk.VCSTypeGitlab, sdk.VCSTypeGerrit:
		keyID, analysisError, err = api.analyzeCommitSignatureThroughOperation(ctx, analysis, *vcsProject, *repoWithSecret)
		if err != nil {
			return api.stopAnalysis(ctx, analysis, err)
		}
	case sdk.VCSTypeGitea, sdk.VCSTypeGithub:
		keyID, analysisError, err = api.analyzeCommitSignatureThroughVcsAPI(ctx, analysis, *vcsProject, *repoWithSecret)
		if err != nil {
			return api.stopAnalysis(ctx, analysis, err)
		}
	default:
		return api.stopAnalysis(ctx, analysis, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to analyze vcs type: %s", vcsProject.Type))
	}
	analysis.Data.SignKeyID = keyID
	if analysisError != "" {
		analysis.Status = sdk.RepositoryAnalysisStatusSkipped
		analysis.Data.Error = analysisError
	}

	// remove secret from repo
	repoWithSecret.Auth = sdk.ProjectRepositoryAuth{}

	var filesContent map[string][]byte
	if analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
		var cdsUser *sdk.AuthentifiedUser
		gpgKey, err := user.LoadGPGKeyByKeyID(ctx, api.mustDB(), analysis.Data.SignKeyID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return api.stopAnalysis(ctx, analysis, sdk.WrapError(err, "unable to find gpg key: %s", analysis.Data.SignKeyID))
			}
			analysis.Status = sdk.RepositoryAnalysisStatusSkipped
			analysis.Data.Error = fmt.Sprintf("gpgkey %s not found", analysis.Data.SignKeyID)
		}

		if gpgKey != nil {
			cdsUser, err = user.LoadByID(ctx, api.mustDB(), gpgKey.AuthentifiedUserID)
			if err != nil {
				if !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return api.stopAnalysis(ctx, analysis, sdk.WrapError(err, "unable to find user %s", gpgKey.AuthentifiedUserID))
				}
				analysis.Status = sdk.RepositoryAnalysisStatusError
				analysis.Data.Error = fmt.Sprintf("user %s not found", gpgKey.AuthentifiedUserID)
			}
		}

		if cdsUser != nil {
			analysis.Data.CDSUserID = cdsUser.ID
			analysis.Data.CDSUserName = cdsUser.Username

			// Check user right
			b, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), sdk.ProjectRoleManage, cdsUser.ID, analysis.ProjectKey)
			if err != nil {
				return api.stopAnalysis(ctx, analysis, err)
			}
			if !b {
				analysis.Status = sdk.RepositoryAnalysisStatusSkipped
				analysis.Data.Error = fmt.Sprintf("user %s doesn't have enough right on project %s", cdsUser.ID, analysis.ProjectKey)
			}

			if analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
				switch vcsProject.Type {
				case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud:
					// get archive
					filesContent, err = api.getCdsArchiveFileOnRepo(ctx, *repoWithSecret, analysis, vcsProject.Name)
				case sdk.VCSTypeGitlab, sdk.VCSTypeGithub, sdk.VCSTypeGitea:
					analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
					filesContent, err = api.getCdsFilesOnVCSDirectory(ctx, analysis, vcsProject.Name, repoWithSecret.Name, analysis.Commit, ".cds")
				case sdk.VCSTypeGerrit:
					return sdk.WithStack(sdk.ErrNotImplemented)
				}

				if err != nil {
					return api.stopAnalysis(ctx, analysis, err)
				}
			}
		}
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return api.stopAnalysis(ctx, analysis, err)
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

	entities, multiErr := api.handleEntitiesFiles(ctx, filesContent, *analysis)
	if multiErr != nil {
		return api.stopAnalysis(ctx, analysis, multiErr...)
	}

	for i := range entities {
		e := &entities[i]
		existingEntity, err := entity.LoadByBranchTypeName(ctx, tx, e.ProjectRepositoryID, e.Branch, e.Type, e.Name)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, err)
		}
		if existingEntity == nil {
			if err := entity.Insert(ctx, tx, e); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.WrapError(err, "unable to insert entity %s", e.Name))
			}
		} else {
			e.ID = existingEntity.ID
			if err := entity.Update(ctx, tx, e); err != nil {
				return api.stopAnalysis(ctx, analysis, sdk.WrapError(err, "unable to save entity %s/%s", e.ID, e.Name))
			}
		}
	}
	analysis.Status = sdk.RepositoryAnalysisStatusSucceed

	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		return sdk.WrapError(err, "unable to update analysis")
	}
	return sdk.WithStack(tx.Commit())
}

func (api *API) handleEntitiesFiles(_ context.Context, filesContent map[string][]byte, analysis sdk.ProjectRepositoryAnalysis) ([]sdk.Entity, []error) {
	entities := make([]sdk.Entity, 0)
	for filePath, content := range filesContent {
		dir, fileName := filepath.Split(filePath)
		fileName = strings.TrimSuffix(fileName, ".yml")
		var es []sdk.Entity
		var err sdk.MultiError
		switch {
		case strings.HasPrefix(filePath, ".cds/worker-model-templates/"):
			var tmpls []sdk.WorkerModelTemplate
			es, err = sdk.ReadEntityFile(dir, fileName, content, &tmpls, sdk.EntityTypeWorkerModelTemplate, analysis)
		case strings.HasPrefix(filePath, ".cds/worker-models/"):
			var wms []sdk.V2WorkerModel
			es, err = sdk.ReadEntityFile(dir, fileName, content, &wms, sdk.EntityTypeWorkerModel, analysis)
		}
		if err != nil {
			return nil, err
		}
		entities = append(entities, es...)
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
				SSHKey:   repoWithSecret.Auth.SSHKeyName,
				User:     repoWithSecret.Auth.Username,
				Password: repoWithSecret.Auth.Token,
			},
			Setup: sdk.OperationSetup{
				Checkout: sdk.OperationCheckout{
					Commit:         analysis.Commit,
					Branch:         analysis.Branch,
					CheckSignature: true,
				},
			},
		}
		if repoWithSecret.Auth.SSHKeyName != "" {
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
		analysis.Data.CommitCheck = false
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
			analysis.Data.Entities = append(analysis.Data.Entities, sdk.ProjectRepositoryDataEntity{
				FileName: c.Name,
				Path:     directory + "/",
			})
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
	analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
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
		if strings.HasSuffix(fileName, ".yml") {
			e := sdk.ProjectRepositoryDataEntity{
				FileName: fileName,
				Path:     dir,
			}
			analysis.Data.Entities = append(analysis.Data.Entities, e)
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
		analysisErrors = append(analysisErrors, e.Error())
	}
	analysis.Data.Error = fmt.Sprintf("%s", strings.Join(analysisErrors, ", "))
	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
