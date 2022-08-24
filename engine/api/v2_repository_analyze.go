package api

import (
	"archive/tar"
	"compress/gzip"
	"context"
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
			log.Debug(ctx, "found %d analysis in progress", len(analysis))
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
	_, next := telemetry.Span(ctx, "api.analyzeRepository.lock")
	lockKey := cache.Key("api:analyzeRepository", analysisID)
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		log.Debug(ctx, "api.analyzeRepository> analyze %s is locked in cache", analysisID)
		next()
		return nil
	}
	next()
	defer func() {
		_ = api.Cache.Unlock(lockKey)
	}()

	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadRepositoryAnalysisById")
	analysis, err := repository.LoadRepositoryAnalysisById(ctx, api.mustDB(), projectRepoID, analysisID)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		next()
		return nil
	}
	if err != nil {
		next()
		return sdk.WrapError(err, "unable to load analyze %s", analysis.ID)
	}
	next()

	if analysis.Status != sdk.RepositoryAnalysisStatusInProgress {
		return nil
	}

	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadVCSByID")
	vcsProject, err := vcs.LoadVCSByID(ctx, api.mustDB(), analysis.ProjectKey, analysis.VCSProjectID)
	if err != nil {
		next()
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, err)
		}
		return err
	}
	next()

	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadRepositoryByID")
	repoWithSecret, err := repository.LoadRepositoryByID(ctx, api.mustDB(), analysis.ProjectRepositoryID, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		next()
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return api.stopAnalysis(ctx, analysis, err)
		}
		return err
	}
	next()

	switch vcsProject.Type {
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud, sdk.VCSTypeGitlab, sdk.VCSTypeGerrit:
		_, next = telemetry.Span(ctx, "api.analyzeRepository.analyzeCommitSignatureThroughOperation")
		if err := api.analyzeCommitSignatureThroughOperation(ctx, analysis, *vcsProject, repoWithSecret); err != nil {
			next()
			return err
		}
		next()
	case sdk.VCSTypeGitea, sdk.VCSTypeGithub:
		_, next = telemetry.Span(ctx, "api.analyzeRepository.analyzeCommitSignatureThroughVcsAPI")
		if err := api.analyzeCommitSignatureThroughVcsAPI(ctx, analysis, *vcsProject, repoWithSecret); err != nil {
			next()
			return err
		}
		next()
	default:
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to analyze vcs type: %s", vcsProject.Type)
	}

	// remove secret from repo
	repoWithSecret.Auth = sdk.ProjectRepositoryAuth{}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	if analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
		var cdsUser *sdk.AuthentifiedUser
		_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadGPGKeyByKeyID")
		gpgKey, err := user.LoadGPGKeyByKeyID(ctx, tx, analysis.Data.SignKeyID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				next()
				return sdk.WrapError(err, "unable to find gpg key: %s", analysis.Data.SignKeyID)
			}
			analysis.Status = sdk.RepositoryAnalysisStatusSkipped
			analysis.Data.Error = fmt.Sprintf("gpgkey %s not found", analysis.Data.SignKeyID)
		}
		next()

		if gpgKey != nil {
			_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadByID")
			cdsUser, err = user.LoadByID(ctx, tx, gpgKey.AuthentifiedUserID)
			if err != nil {
				if !sdk.ErrorIs(err, sdk.ErrNotFound) {
					next()
					return sdk.WrapError(err, "unable to find user %s", gpgKey.AuthentifiedUserID)
				}
				analysis.Status = sdk.RepositoryAnalysisStatusError
				analysis.Data.Error = fmt.Sprintf("user %s not found", gpgKey.AuthentifiedUserID)
			}
			next()
		}

		if cdsUser != nil {
			analysis.Data.CDSUserID = cdsUser.ID
			analysis.Data.CDSUserName = cdsUser.Username

			// Check user right
			_, next = telemetry.Span(ctx, "api.analyzeRepository.HasRoleOnProjectAndUserID")
			b, err := rbac.HasRoleOnProjectAndUserID(ctx, tx, sdk.RoleManage, cdsUser.ID, analysis.ProjectKey)
			if err != nil {
				next()
				return err
			}
			next()
			if !b {
				analysis.Status = sdk.RepositoryAnalysisStatusSkipped
				analysis.Data.Error = fmt.Sprintf("user %s doesn't have enough right on project %s", cdsUser.ID, analysis.ProjectKey)
			}

			if analysis.Status == sdk.RepositoryAnalysisStatusInProgress {
				client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analysis.ProjectKey, vcsProject.Name)
				if err != nil {
					return err
				}

				switch vcsProject.Type {
				case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud:
					// get archive
					_, next = telemetry.Span(ctx, "api.analyzeRepository.getCdsArchiveFileOnRepo")
					err = api.getCdsArchiveFileOnRepo(ctx, client, repoWithSecret, analysis)
					next()
				case sdk.VCSTypeGitlab, sdk.VCSTypeGithub, sdk.VCSTypeGitea:
					analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
					_, next = telemetry.Span(ctx, "api.analyzeRepository.getCdsFilesOnVCSDirectory")
					err = api.getCdsFilesOnVCSDirectory(ctx, client, analysis, repoWithSecret.Name, analysis.Commit, ".cds")
					next()
				case sdk.VCSTypeGerrit:
					return sdk.WithStack(sdk.ErrNotImplemented)
				}
				if err != nil {
					analysis.Status = sdk.RepositoryAnalysisStatusError
					analysis.Data.Error = err.Error()
				} else {
					analysis.Status = sdk.RepositoryAnalysisStatusSucceed
				}
			}
		}
	}

	_, next = telemetry.Span(ctx, "api.analyzeRepository.getCdsFilesOnVCSDirectory")
	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		next()
		return sdk.WrapError(err, "unable to failed analyze")
	}
	next()
	return sdk.WithStack(tx.Commit())
}

func (api *API) analyzeCommitSignatureThroughVcsAPI(ctx context.Context, analysis *sdk.ProjectRepositoryAnalysis, vcsProject sdk.VCSProject, repoWithSecret sdk.ProjectRepository) error {
	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	// Check commit signature
	client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analysis.ProjectKey, vcsProject.Name)
	if err != nil {
		_ = tx.Rollback() // nolint
		return err
	}
	vcsCommit, err := client.Commit(ctx, repoWithSecret.Name, analysis.Commit)
	if err != nil {
		_ = tx.Rollback() // nolint
		return err
	}
	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	if vcsCommit.Hash == "" {
		analysis.Status = sdk.RepositoryAnalysisStatusError
		analysis.Data.Error = fmt.Sprintf("commit %s not found", analysis.Commit)
	} else {
		if vcsCommit.Signature != "" {
			keyId, err := gpg.GetKeyIdFromSignature(vcsCommit.Signature)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				analysis.Status = sdk.RepositoryAnalysisStatusError
				analysis.Data.Error = fmt.Sprintf("unable to extract keyID from signature: %v", err)
			} else {
				analysis.Data.SignKeyID = keyId
				analysis.Data.CommitCheck = true
			}
		} else {
			analysis.Data.CommitCheck = false
			analysis.Status = sdk.RepositoryAnalysisStatusSkipped
			analysis.Data.Error = fmt.Sprintf("commit %s is not signed", vcsCommit.Hash)
		}
	}
	return nil
}

func (api *API) analyzeCommitSignatureThroughOperation(ctx context.Context, analysis *sdk.ProjectRepositoryAnalysis, vcsProject sdk.VCSProject, repoWithSecret sdk.ProjectRepository) error {
	if analysis.Data.OperationUUID == "" {
		proj, err := project.Load(ctx, api.mustDB(), analysis.ProjectKey)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return api.stopAnalysis(ctx, analysis, err)
			}
			return err
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
			return sdk.WithStack(err)
		}

		_, next := telemetry.Span(ctx, "api.analyzeCheckCommitThroughOperation.PostRepositoryOperation")
		if err := operation.PostRepositoryOperation(ctx, tx, *proj, ope, nil); err != nil {
			return err
		}
		next()
		analysis.Data.OperationUUID = ope.UUID

		_, next = telemetry.Span(ctx, "api.analyzeCheckCommitThroughOperation.UpdateAnalysis")
		if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
			return err
		}
		next()
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
	}
	_, next := telemetry.Span(ctx, "api.analyzeRepository.Poll")
	ope, err := operation.Poll(ctx, api.mustDB(), analysis.Data.OperationUUID)
	if err != nil {
		next()
		return err
	}
	next()

	if ope.Status == sdk.OperationStatusDone && ope.Setup.Checkout.Result.CommitVerified {
		analysis.Data.CommitCheck = true
		analysis.Data.SignKeyID = ope.Setup.Checkout.Result.SignKeyID
	}
	if ope.Status == sdk.OperationStatusDone && !ope.Setup.Checkout.Result.CommitVerified {
		analysis.Data.CommitCheck = false
		analysis.Data.SignKeyID = ope.Setup.Checkout.Result.SignKeyID
		analysis.Data.Error = ope.Setup.Checkout.Result.Msg
		analysis.Status = sdk.RepositoryAnalysisStatusSkipped
	}
	if ope.Status == sdk.OperationStatusError {
		analysis.Data.Error = ope.Error.Message
		analysis.Status = sdk.RepositoryAnalysisStatusError
	}
	return nil
}

func (api *API) getCdsFilesOnVCSDirectory(ctx context.Context, client sdk.VCSAuthorizedClientService, analysis *sdk.ProjectRepositoryAnalysis, repoName, commit, directory string) error {
	contents, err := client.ListContent(ctx, repoName, commit, directory)
	if err != nil {
		return sdk.WrapError(err, "unable to list content on commit [%s] in directory %s: %v", commit, directory, err)
	}
	for _, c := range contents {
		if c.IsFile && strings.HasSuffix(c.Name, ".yml") {
			analysis.Data.Entities = append(analysis.Data.Entities, sdk.ProjectRepositoryDataEntity{
				FileName: c.Name,
				Path:     directory + "/",
			})
		}
		if c.IsDirectory {
			if err := api.getCdsFilesOnVCSDirectory(ctx, client, analysis, repoName, commit, directory+"/"+c.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (api *API) getCdsArchiveFileOnRepo(ctx context.Context, client sdk.VCSAuthorizedClientService, repo sdk.ProjectRepository, analysis *sdk.ProjectRepositoryAnalysis) error {
	analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
	reader, _, err := client.GetArchive(ctx, repo.Name, ".cds", "tar.gz", analysis.Commit)
	if err != nil {
		return err
	}

	gzf, err := gzip.NewReader(reader)
	if err != nil {
		return sdk.WrapError(err, "unable to read gzip file")
	}
	tarReader := tar.NewReader(gzf)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read tar file"))
		}

		dir, fileName := filepath.Split(hdr.Name)
		if strings.HasSuffix(fileName, ".yml") {
			entity := sdk.ProjectRepositoryDataEntity{
				FileName: fileName,
				Path:     dir,
			}
			analysis.Data.Entities = append(analysis.Data.Entities, entity)
		}

	}
	return nil
}

func (api *API) stopAnalysis(ctx context.Context, analysis *sdk.ProjectRepositoryAnalysis, originalError error) error {
	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint
	analysis.Status = sdk.RepositoryAnalysisStatusError
	analysis.Data.Error = fmt.Sprintf("%v", originalError)
	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return originalError
}
