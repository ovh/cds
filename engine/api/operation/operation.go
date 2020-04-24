package operation

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var CacheOperationKey = cache.Key("repositories", "operation", "push")

func pushOperation(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, data exportentities.WorkflowComponents, ope sdk.Operation) (*sdk.Operation, error) {
	if ope.RepositoryStrategy.SSHKey != "" {
		key := proj.GetSSHKey(ope.RepositoryStrategy.SSHKey)
		if key == nil {
			return nil, sdk.WithStack(fmt.Errorf("unable to find key %s on project %s", ope.RepositoryStrategy.SSHKey, proj.Key))
		}
		ope.RepositoryStrategy.SSHKeyContent = key.Private
	}

	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, ope.VCSServer)
	if vcsServer == nil {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "no vcs server found on project %s for given name %s", proj.Key, ope.VCSServer)
	}
	client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, vcsServer)
	if err != nil {
		return nil, err
	}
	repo, err := client.RepoByFullname(ctx, ope.RepoFullName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get repo %s", ope.RepoFullName)
	}
	if ope.RepositoryStrategy.ConnectionType == "ssh" {
		ope.URL = repo.SSHCloneURL
	} else {
		ope.URL = repo.HTTPCloneURL
	}

	buf := new(bytes.Buffer)
	if err := exportentities.TarWorkflowComponents(ctx, data, buf); err != nil {
		return nil, sdk.WrapError(err, "cannot tar pulled workflow")
	}

	multipartData := &services.MultiPartData{
		Reader:      buf,
		ContentType: "application/tar",
	}
	if err := PostRepositoryOperation(ctx, db, proj, &ope, multipartData); err != nil {
		return nil, sdk.WrapError(err, "unable to post repository operation")
	}

	ope.RepositoryStrategy.SSHKeyContent = ""
	_ = store.SetWithTTL(cache.Key(CacheOperationKey, ope.UUID), ope, 300)
	return &ope, nil
}

func PushOperation(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, data exportentities.WorkflowComponents, vcsServerName, repoFullname, branch, message string, vcsStrategy sdk.RepositoryStrategy, u sdk.Identifiable) (*sdk.Operation, error) {
	ope := sdk.Operation{
		VCSServer:          vcsServerName,
		RepoFullName:       repoFullname,
		RepositoryStrategy: vcsStrategy,
		Setup: sdk.OperationSetup{
			Push: sdk.OperationPush{
				FromBranch: branch,
				Message:    message,
			},
		},
	}
	ope.User.Email = u.GetEmail()
	ope.User.Username = u.GetFullname()
	ope.User.Username = u.GetUsername()

	return pushOperation(ctx, db, store, proj, data, ope)
}

func PushOperationUpdate(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, data exportentities.WorkflowComponents, vcsServerName, repoFullname, branch, message string, vcsStrategy sdk.RepositoryStrategy, u sdk.Identifiable) (*sdk.Operation, error) {
	ope := sdk.Operation{
		VCSServer:          vcsServerName,
		RepoFullName:       repoFullname,
		RepositoryStrategy: vcsStrategy,
		Setup: sdk.OperationSetup{
			Push: sdk.OperationPush{
				FromBranch: branch,
				Message:    message,
				Update:     true,
			},
		},
	}
	ope.User.Email = u.GetEmail()
	ope.User.Username = u.GetFullname()
	ope.User.Username = u.GetUsername()

	return pushOperation(ctx, db, store, proj, data, ope)
}

// PostRepositoryOperation creates a new repository operation
func PostRepositoryOperation(ctx context.Context, db gorp.SqlExecutor, prj sdk.Project, ope *sdk.Operation, multipartData *services.MultiPartData) error {
	srvs, err := services.LoadAllByType(ctx, db, services.TypeRepositories)
	if err != nil {
		return sdk.WrapError(err, "Unable to found repositories service")
	}

	if ope.RepositoryStrategy.ConnectionType == "ssh" {
		found := false
		for _, k := range prj.Keys {
			if k.Name == ope.RepositoryStrategy.SSHKey {
				ope.RepositoryStrategy.SSHKeyContent = k.Private
				found = true
				break
			}
		}
		if !found {
			return sdk.WithStack(fmt.Errorf("unable to find key %s on project %s", ope.RepositoryStrategy.SSHKey, prj.Key))
		}
		ope.RepositoryStrategy.User = ""
		ope.RepositoryStrategy.Password = ""
	} else {
		ope.RepositoryStrategy.SSHKey = ""
		ope.RepositoryStrategy.SSHKeyContent = ""
	}

	if multipartData == nil {
		if _, _, err := services.NewClient(db, srvs).DoJSONRequest(ctx, http.MethodPost, "/operations", ope, ope); err != nil {
			return sdk.WrapError(err, "Unable to perform operation")
		}
		return nil
	}
	if _, err := services.NewClient(db, srvs).DoMultiPartRequest(ctx, http.MethodPost, "/operations", multipartData, ope, ope); err != nil {
		return sdk.WrapError(err, "Unable to perform multipart operation")
	}
	return nil
}

// GetRepositoryOperation get repository operation status
func GetRepositoryOperation(ctx context.Context, db gorp.SqlExecutor, ope *sdk.Operation) error {
	srvs, err := services.LoadAllByType(ctx, db, services.TypeRepositories)
	if err != nil {
		return sdk.WrapError(err, "Unable to found repositories service")
	}

	if _, _, err := services.NewClient(db, srvs).DoJSONRequest(ctx, http.MethodGet, "/operations/"+ope.UUID, nil, ope); err != nil {
		return sdk.WrapError(err, "Unable to get operation")
	}
	return nil
}
