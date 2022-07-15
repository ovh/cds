package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
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

			var repositoryID string
			var operationUUID string
			switch vcsProject.Type {
			case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud, sdk.VCSTypeGitlab, sdk.VCSTypeGerrit:
				clearRepo, err := repository.LoadRepositoryByName(ctx, api.mustDB(), vcsProject.ID, analyze.RepoName, gorpmapping.GetOptions.WithDecryption)
				if err != nil {
					return err
				}
				repositoryID = clearRepo.ID

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

				if err := operation.PostRepositoryOperation(ctx, api.mustDB(), *proj, ope, nil); err != nil {
					return err
				}
				operationUUID = ope.UUID
			case sdk.VCSTypeGitea, sdk.VCSTypeGithub:
			default:
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to analyze vcs type: %s", vcsProject.Type)
			}

			// Save analyze
			repoAnalyze := sdk.ProjectRepositoryAnalyze{
				Status:              sdk.RepositoryAnalyzeStatusInProgress,
				ProjectRepositoryID: repositoryID,
				VCSProjectID:        vcsProject.ID,
				ProjectKey:          proj.Key,
				Branch:              analyze.Branch,
				Commit:              analyze.Commit,
				Data: sdk.ProjectRepositoryData{
					OperationUUID: operationUUID,
				},
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WrapError(err, "unable to start db transaction")
			}
			defer tx.Rollback() // nolint

			if err := repository.InsertAnalyze(ctx, tx, &repoAnalyze); err != nil {
				return err
			}

			response := sdk.AnalyzeResponse{
				AnalyzeID:   repoAnalyze.ID,
				OperationID: operationUUID,
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
							log.Error(ctx, "WorkflowRunCraft> error on workflow run %d: %v", a.ID, err)
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
		return sdk.WrapError(err, "unable to load analyze %d", analyze.ID)
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
		if ope.Status == sdk.OperationStatusDone {
			analyze.Data.CommitCheck = true
			analyze.Data.SignKeyID = ope.Setup.Checkout.Result.SignKeyID
		}
		if ope.Status == sdk.OperationStatusError {
			analyze.Data.Error = ope.Error.Message
			analyze.Status = sdk.RepositoryAnalyzeStatusError

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

	client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, analyze.ProjectKey, vcsProject.Name)
	if err != nil {
		return err
	}

	switch vcsProject.Type {
	case sdk.VCSTypeBitbucketServer, sdk.VCSTypeBitbucketCloud:
		// get archive
		err = api.getCdsArchiveFileOnRepo(ctx, client, *repo, analyze)

	case sdk.VCSTypeGitlab, sdk.VCSTypeGithub, sdk.VCSTypeGitea:
		return sdk.WithStack(sdk.ErrNotImplemented)
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

	if err := repository.UpdateAnalyze(ctx, tx, analyze); err != nil {
		return err
	}

	return sdk.WrapError(tx.Commit(), "unable to commit")
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

		log.Debug(ctx, "extract cds files> Reading %s", hdr.Name)
		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tarReader); err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read tar file"))
		}

		b := buff.Bytes()
		if len(b) == 0 {
			continue
		}
		hash := md5.New()
		if _, err := io.Copy(hash, buff); err != nil {
			return sdk.WrapError(err, "unable to compute md5 for %s", hdr.Name)
		}

		dir, fileName := filepath.Split(hdr.Name)
		entity := sdk.ProjectRepositoryDataEntity{
			FileName: fileName,
			Path:     dir,
			Content:  string(b),
			Md5Sum:   hex.EncodeToString(hash.Sum(nil)),
		}
		analyze.Data.Entities = append(analyze.Data.Entities, entity)
	}
	return nil
}
