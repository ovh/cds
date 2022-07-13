package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

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
							Commit: analyze.Commit,
							Branch: analyze.Branch,
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
			return service.WriteJSON(w, &response, http.StatusCreated)
		}
}
