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

func (api *API) getProjectRepositoryAnalyzesHandler() ([]service.RbacChecker, service.Handler) {
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

			analyzes, err := repository.LoadAllAnalyzesByRepo(ctx, api.mustDB(), repo.ID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, analyzes, http.StatusOK)
		}
}

func (api *API) getProjectRepositoryAnalyzeHandler() ([]service.RbacChecker, service.Handler) {
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
			analyseID := vars["analyzeID"]

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			analyze, err := repository.LoadRepositoryAnalyzeById(ctx, api.mustDB(), repo.ID, analyseID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, analyze, http.StatusOK)
		}
}

// postRepositoryAnalyzeHandler Trigger repository analysis
func (api *API) postRepositoryAnalyzeHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.IsHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			var analyze sdk.AnalyzeRequest
			if err := service.UnmarshalBody(req, &analyze); err != nil {
				return err
			}

			ctx = context.WithValue(ctx, cdslog.VCSServer, analyze.VcsName)
			ctx = context.WithValue(ctx, cdslog.Repository, analyze.RepoName)

			proj, err := project.Load(ctx, api.mustDB(), analyze.ProjectKey, project.LoadOptions.WithClearKeys)
			if err != nil {
				return err
			}

			vcsProject, err := vcs.LoadVCSByProject(ctx, api.mustDB(), analyze.ProjectKey, analyze.VcsName)
			if err != nil {
				return err
			}

			clearRepo, err := repository.LoadRepositoryByName(ctx, api.mustDB(), vcsProject.ID, analyze.RepoName, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WrapError(err, "unable to start db transaction")
			}
			defer tx.Rollback() // nolint

			// Save analyze
			repoAnalyze := sdk.ProjectRepositoryAnalyze{
				Status:              sdk.RepositoryAnalyzeStatusInProgress,
				ProjectRepositoryID: clearRepo.ID,
				VCSProjectID:        vcsProject.ID,
				ProjectKey:          proj.Key,
				Branch:              analyze.Branch,
				Commit:              analyze.Commit,
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
							Commit:         analyze.Commit,
							Branch:         analyze.Branch,
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
				repoAnalyze.Data.OperationUUID = ope.UUID
			case sdk.VCSTypeGitea, sdk.VCSTypeGithub:
				// Check commit signature
				client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analyze.ProjectKey, vcsProject.Name)
				if err != nil {
					return err
				}
				vcsCommit, err := client.Commit(ctx, analyze.RepoName, analyze.Commit)
				if err != nil {
					return err
				}
				if vcsCommit.Hash == "" {
					repoAnalyze.Status = sdk.RepositoryAnalyzeStatusError
					repoAnalyze.Data.Error = fmt.Sprintf("commit %s not found", analyze.Commit)
				} else {
					if vcsCommit.Verified && vcsCommit.KeySignID != "" {
						repoAnalyze.Data.SignKeyID = vcsCommit.KeySignID
						repoAnalyze.Data.CommitCheck = true
					} else {
						repoAnalyze.Data.SignKeyID = vcsCommit.KeySignID
						repoAnalyze.Data.CommitCheck = false
						repoAnalyze.Status = sdk.RepositoryAnalyzeStatusSkipped
						repoAnalyze.Data.Error = fmt.Sprintf("commit %s is not signed", vcsCommit.Hash)
					}
				}

			default:
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to analyze vcs type: %s", vcsProject.Type)
			}

			if err := repository.InsertAnalyze(ctx, tx, &repoAnalyze); err != nil {
				return err
			}

			response := sdk.AnalyzeResponse{
				AnalyzeID:   repoAnalyze.ID,
				OperationID: repoAnalyze.Data.OperationUUID,
			}

			if err := tx.Commit(); err != nil {
				return sdk.WrapError(err, "unable to commit transaction")
			}
			return service.WriteJSON(w, &response, http.StatusCreated)
		}
}

func (api *API) repositoryAnalyzePoller(ctx context.Context, tick time.Duration) error {
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
					"repositoryAnalyzePoller-"+a.ID,
					func(ctx context.Context) {
						ctx = telemetry.New(ctx, api, "api.repositoryAnalyzePoller", nil, trace.SpanKindUnspecified)
						if err := api.analyzeRepository(ctx, a.ProjectRepositoryID, a.ID); err != nil {
							log.ErrorWithStackTrace(ctx, err)
						}
					},
				)
			}
		}
	}
}

func (api *API) analyzeRepository(ctx context.Context, projectRepoID string, analyzeID string) error {
	_, next := telemetry.Span(ctx, "api.analyzeRepository.lock")
	lockKey := cache.Key("api:analyzeRepository", analyzeID)
	b, err := api.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		log.Debug(ctx, "api.analyzeRepository> analyze %s is locked in cache", analyzeID)
		next()
		return nil
	}
	next()
	defer func() {
		_ = api.Cache.Unlock(lockKey)
	}()

	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadRepositoryAnalyzeById")
	analyze, err := repository.LoadRepositoryAnalyzeById(ctx, api.mustDB(), projectRepoID, analyzeID)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		next()
		return nil
	}
	if err != nil {
		next()
		return sdk.WrapError(err, "unable to load analyze %s", analyze.ID)
	}
	next()

	if analyze.Status != sdk.RepositoryAnalyzeStatusInProgress {
		return nil
	}

	if analyze.Data.OperationUUID != "" {
		_, next = telemetry.Span(ctx, "api.analyzeRepository.Poll")
		ope, err := operation.Poll(ctx, api.mustDB(), analyze.Data.OperationUUID)
		if err != nil {
			next()
			return err
		}
		next()

		stopAnalyze := false
		if ope.Status == sdk.OperationStatusDone && ope.Setup.Checkout.Result.CommitVerified {
			analyze.Data.CommitCheck = true
			analyze.Data.SignKeyID = ope.Setup.Checkout.Result.SignKeyID
		}
		if ope.Status == sdk.OperationStatusDone && !ope.Setup.Checkout.Result.CommitVerified {
			analyze.Data.CommitCheck = false
			analyze.Data.SignKeyID = ope.Setup.Checkout.Result.SignKeyID
			analyze.Data.Error = ope.Setup.Checkout.Result.Msg
			analyze.Status = sdk.RepositoryAnalyzeStatusSkipped
			stopAnalyze = true
		}

		if ope.Status == sdk.OperationStatusError {
			analyze.Data.Error = ope.Error.Message
			analyze.Status = sdk.RepositoryAnalyzeStatusError
			stopAnalyze = true
		}

		if stopAnalyze {
			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WrapError(err, "unable to start transaction")
			}
			defer tx.Rollback()
			if err := repository.UpdateAnalyze(ctx, tx, analyze); err != nil {
				return sdk.WrapError(err, "unable to failed analyze")
			}
			return sdk.WithStack(tx.Commit())
		}
	}

	// Retrieve cds files
	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadRepositoryByVCSAndID")
	repo, err := repository.LoadRepositoryByVCSAndID(ctx, api.mustDB(), analyze.VCSProjectID, analyze.ProjectRepositoryID)
	if err != nil {
		next()
		return err
	}
	next()
	_, next = telemetry.Span(ctx, "api.analyzeRepository.LoadVCSByID")
	vcsProject, err := vcs.LoadVCSByID(ctx, api.mustDB(), analyze.ProjectKey, analyze.VCSProjectID, gorpmapping.GetOptions.WithDecryption)
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
	gpgKey, err := user.LoadGPGKeyByKeyID(ctx, tx, analyze.Data.SignKeyID)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WrapError(err, "unable to find gpg key: %s", analyze.Data.SignKeyID)
		}
		analyze.Status = sdk.RepositoryAnalyzeStatusError
		analyze.Data.Error = fmt.Sprintf("gpgkey %s not found", analyze.Data.SignKeyID)
	}

	if gpgKey != nil {
		cdsUser, err = user.LoadByID(ctx, tx, gpgKey.AuthentifiedUserID)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.WrapError(err, "unable to find user %s", gpgKey.AuthentifiedUserID)
			}
			analyze.Status = sdk.RepositoryAnalyzeStatusError
			analyze.Data.Error = fmt.Sprintf("user %s not found", gpgKey.AuthentifiedUserID)
		}
	}

	if cdsUser != nil {
		analyze.Data.CDSUserID = cdsUser.ID
		analyze.Data.CDSUserName = cdsUser.Username

		// Check user right
		b, err := rbac.HasRoleOnProjectAndUserID(ctx, tx, sdk.RoleManage, cdsUser.ID, analyze.ProjectKey)
		if err != nil {
			return err
		}
		if !b {
			analyze.Status = sdk.RepositoryAnalyzeStatusSkipped
			analyze.Data.Error = fmt.Sprintf("user %s doesn't have enough right on project %s", cdsUser.ID, analyze.ProjectKey)
		}

		if analyze.Status != sdk.RepositoryAnalyzeStatusSkipped {
			client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analyze.ProjectKey, vcsProject.Name)
			if err != nil {
				return err
			}

			switch vcsProject.Type {
			case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud:
				// get archive
				err = api.getCdsArchiveFileOnRepo(ctx, client, *repo, analyze)
			case sdk.VCSTypeGitlab, sdk.VCSTypeGithub, sdk.VCSTypeGitea:
				analyze.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
				err = api.getCdsFilesOnVCSDirectory(ctx, client, analyze, repo.Name, analyze.Commit, ".cds")
			case sdk.VCSTypeGerrit:
				return sdk.WithStack(sdk.ErrNotImplemented)
			}

			// Update analyze
			if err != nil {
				analyze.Status = sdk.RepositoryAnalyzeStatusError
				analyze.Data.Error = err.Error()
			} else {
				analyze.Status = sdk.RepositoryAnalyzeStatusSucceed
			}
		}
	}

	if err := repository.UpdateAnalyze(ctx, tx, analyze); err != nil {
		return err
	}
	return sdk.WrapError(tx.Commit(), "unable to commit")
}

func (api *API) getCdsFilesOnVCSDirectory(ctx context.Context, client sdk.VCSAuthorizedClientService, analyze *sdk.ProjectRepositoryAnalyze, repoName, commit, directory string) error {
	contents, err := client.ListContent(ctx, repoName, commit, directory)
	if err != nil {
		return sdk.WrapError(err, "unable to list content on commit [%s] in directory %s: %v", commit, directory, err)
	}
	for _, c := range contents {
		if c.IsFile && strings.HasSuffix(c.Name, ".yml") {
			analyze.Data.Entities = append(analyze.Data.Entities, sdk.ProjectRepositoryDataEntity{
				FileName: c.Name,
				Path:     directory + "/",
			})
		}
		if c.IsDirectory {
			if err := api.getCdsFilesOnVCSDirectory(ctx, client, analyze, repoName, commit, directory+"/"+c.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (api *API) getCdsArchiveFileOnRepo(ctx context.Context, client sdk.VCSAuthorizedClientService, repo sdk.ProjectRepository, analyze *sdk.ProjectRepositoryAnalyze) error {
	analyze.Data.Entities = make([]sdk.ProjectRepositoryDataEntity, 0)
	reader, _, err := client.GetArchive(ctx, repo.Name, ".cds", "tar.gz", analyze.Commit)
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
			analyze.Data.Entities = append(analyze.Data.Entities, entity)
		}

	}
	return nil
}
