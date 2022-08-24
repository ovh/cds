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
				Data: sdk.ProjectRepositoryData{
					OperationUUID: "",
				},
			}

			switch vcsProject.Type {
			case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud, sdk.VCSTypeGitlab, sdk.VCSTypeGerrit:
				ope := &sdk.Operation{
					VCSServer:    vcsProject.Name,
					RepoFullName: clearRepo.Name,
					URL:          clearRepo.CloneURL,
					RepositoryStrategy: sdk.RepositoryStrategy{
						SSHKey:   clearRepo.Auth.SSHKeyName,
						User:     clearRepo.Auth.Username,
						Password: clearRepo.Auth.Token,
					},
					Setup: sdk.OperationSetup{
						Checkout: sdk.OperationCheckout{
							Commit:         analysis.Commit,
							Branch:         analysis.Branch,
							CheckSignature: true,
						},
					},
				}

				if clearRepo.Auth.SSHKeyName != "" {
					ope.RepositoryStrategy.ConnectionType = "ssh"
				} else {
					ope.RepositoryStrategy.ConnectionType = "https"
				}

				if err := operation.PostRepositoryOperation(ctx, tx, *proj, ope, nil); err != nil {
					return err
				}
				repoAnalysis.Data.OperationUUID = ope.UUID
			case sdk.VCSTypeGitea, sdk.VCSTypeGithub:
				// Check commit signature
				client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analysis.ProjectKey, vcsProject.Name)
				if err != nil {
					return err
				}
				vcsCommit, err := client.Commit(ctx, analysis.RepoName, analysis.Commit)
				if err != nil {
					return err
				}
				if vcsCommit.Hash == "" {
					repoAnalysis.Status = sdk.RepositoryAnalysisStatusError
					repoAnalysis.Data.Error = fmt.Sprintf("commit %s not found", analysis.Commit)
				} else {
					if vcsCommit.Verified && vcsCommit.KeySignID != "" {
						repoAnalysis.Data.SignKeyID = vcsCommit.KeySignID
						repoAnalysis.Data.CommitCheck = true
					} else {
						repoAnalysis.Data.SignKeyID = vcsCommit.KeySignID
						repoAnalysis.Data.CommitCheck = false
						repoAnalysis.Status = sdk.RepositoryAnalysisStatusSkipped
						repoAnalysis.Data.Error = fmt.Sprintf("commit %s is not signed", vcsCommit.Hash)
					}
				}

			default:
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to analyze vcs type: %s", vcsProject.Type)
			}

			if err := repository.InsertAnalysis(ctx, tx, &repoAnalysis); err != nil {
				return err
			}

			response := sdk.AnalysisResponse{
				AnalysisID:  repoAnalysis.ID,
				OperationID: repoAnalysis.Data.OperationUUID,
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

	if analysis.Data.OperationUUID != "" {
		_, next = telemetry.Span(ctx, "api.analyzeRepository.Poll")
		ope, err := operation.Poll(ctx, api.mustDB(), analysis.Data.OperationUUID)
		if err != nil {
			next()
			return err
		}
		next()

		stopAnalyze := false
		if ope.Status == sdk.OperationStatusDone && ope.Setup.Checkout.Result.CommitVerified {
			analysis.Data.CommitCheck = true
			analysis.Data.SignKeyID = ope.Setup.Checkout.Result.SignKeyID
		}
		if ope.Status == sdk.OperationStatusDone && !ope.Setup.Checkout.Result.CommitVerified {
			analysis.Data.CommitCheck = false
			analysis.Data.SignKeyID = ope.Setup.Checkout.Result.SignKeyID
			analysis.Data.Error = ope.Setup.Checkout.Result.Msg
			analysis.Status = sdk.RepositoryAnalysisStatusSkipped
			stopAnalyze = true
		}

		if ope.Status == sdk.OperationStatusError {
			analysis.Data.Error = ope.Error.Message
			analysis.Status = sdk.RepositoryAnalysisStatusError
			stopAnalyze = true
		}

		if stopAnalyze {
			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WrapError(err, "unable to start transaction")
			}
			defer tx.Rollback()
			if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
				return sdk.WrapError(err, "unable to failed analyze")
			}
			return sdk.WithStack(tx.Commit())
		}
	}

	// Retrieve cds files
	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadRepositoryByVCSAndID")
	repo, err := repository.LoadRepositoryByVCSAndID(ctx, api.mustDB(), analysis.VCSProjectID, analysis.ProjectRepositoryID)
	if err != nil {
		next()
		return err
	}
	next()
	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadVCSByID")
	vcsProject, err := vcs.LoadVCSByID(ctx, api.mustDB(), analysis.ProjectKey, analysis.VCSProjectID, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		next()
		return err
	}
	next()

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WrapError(err, "unable to start transaction")
	}
	defer tx.Rollback() // nolint

	// Search User by gpgkey
	var cdsUser *sdk.AuthentifiedUser
	gpgKey, err := user.LoadGPGKeyByKeyID(ctx, tx, analysis.Data.SignKeyID)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WrapError(err, "unable to find gpg key: %s", analysis.Data.SignKeyID)
		}
		analysis.Status = sdk.RepositoryAnalysisStatusError
		analysis.Data.Error = fmt.Sprintf("gpgkey %s not found", analysis.Data.SignKeyID)
	}

	if gpgKey != nil {
		cdsUser, err = user.LoadByID(ctx, tx, gpgKey.AuthentifiedUserID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.WrapError(err, "unable to find user %s", gpgKey.AuthentifiedUserID)
			}
			analysis.Status = sdk.RepositoryAnalysisStatusError
			analysis.Data.Error = fmt.Sprintf("user %s not found", gpgKey.AuthentifiedUserID)
		}
	}

	if cdsUser != nil {
		analysis.Data.CDSUserID = cdsUser.ID
		analysis.Data.CDSUserName = cdsUser.Username

		// Check user right
		b, err := rbac.HasRoleOnProjectAndUserID(ctx, tx, sdk.RoleManage, cdsUser.ID, analysis.ProjectKey)
		if err != nil {
			return err
		}
		if !b {
			analysis.Status = sdk.RepositoryAnalysisStatusSkipped
			analysis.Data.Error = fmt.Sprintf("user %s doesn't have enough right on project %s", cdsUser.ID, analysis.ProjectKey)
		}

		if analysis.Status != sdk.RepositoryAnalysisStatusSkipped {
			client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analysis.ProjectKey, vcsProject.Name)
			if err != nil {
				return err
			}

			switch vcsProject.Type {
			case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud:
				// get archive
				err = api.getCdsArchiveFileOnRepo(ctx, client, *repo, analysis)
			case sdk.VCSTypeGitlab, sdk.VCSTypeGithub, sdk.VCSTypeGitea:
				analysis.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
				err = api.getCdsFilesOnVCSDirectory(ctx, client, analysis, repo.Name, analysis.Commit, ".cds")
			case sdk.VCSTypeGerrit:
				return sdk.WithStack(sdk.ErrNotImplemented)
			}

			// Update analyze
			if err != nil {
				analysis.Status = sdk.RepositoryAnalysisStatusError
				analysis.Data.Error = err.Error()
			} else {
				analysis.Status = sdk.RepositoryAnalysisStatusSucceed
			}
		}
	}

	if err := repository.UpdateAnalysis(ctx, tx, analysis); err != nil {
		return err
	}
	return sdk.WrapError(tx.Commit(), "unable to commit")
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
