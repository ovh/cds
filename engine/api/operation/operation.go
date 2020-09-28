package operation

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func pushOperation(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, data exportentities.WorkflowComponents, ope sdk.Operation) (*sdk.Operation, error) {
	if ope.RepositoryStrategy.SSHKey != "" {
		key := proj.GetSSHKey(ope.RepositoryStrategy.SSHKey)
		if key == nil {
			return nil, sdk.WithStack(fmt.Errorf("unable to find key %s on project %s", ope.RepositoryStrategy.SSHKey, proj.Key))
		}
		ope.RepositoryStrategy.SSHKeyContent = key.Private
	}

	vcsServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, proj.Key, ope.VCSServer)
	if err != nil {
		return nil, err
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

	ope.RepositoryStrategy.SSHKeyContent = sdk.PasswordPlaceholder
	ope.RepositoryStrategy.Password = sdk.PasswordPlaceholder

	return &ope, nil
}

func PushOperation(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, data exportentities.WorkflowComponents, vcsServerName, repoFullname, branch, message string, vcsStrategy sdk.RepositoryStrategy, u sdk.Identifiable) (*sdk.Operation, error) {
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
	ope.User.Fullname = u.GetFullname()
	ope.User.Username = u.GetUsername()

	return pushOperation(ctx, db, store, proj, data, ope)
}

func PushOperationUpdate(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, data exportentities.WorkflowComponents, vcsServerName, repoFullname, branch, message string, vcsStrategy sdk.RepositoryStrategy, u sdk.Identifiable) (*sdk.Operation, error) {
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
	ope.User.Fullname = u.GetFullname()
	ope.User.Username = u.GetUsername()

	return pushOperation(ctx, db, store, proj, data, ope)
}

// PostRepositoryOperation creates a new repository operation
func PostRepositoryOperation(ctx context.Context, db gorp.SqlExecutor, prj sdk.Project, ope *sdk.Operation, multipartData *services.MultiPartData) error {
	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeRepositories)
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
		return sdk.WrapError(err, "unable to perform multipart operation")
	}
	return nil
}

// GetRepositoryOperation get repository operation status.
func GetRepositoryOperation(ctx context.Context, db gorp.SqlExecutor, uuid string) (*sdk.Operation, error) {
	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeRepositories)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to found repositories service")
	}
	var ope sdk.Operation
	if _, _, err := services.NewClient(db, srvs).DoJSONRequest(ctx, http.MethodGet, "/operations/"+uuid, nil, &ope); err != nil {
		return nil, sdk.WrapError(err, "unable to get operation")
	}
	return &ope, nil
}

// Poll repository operatino for given uuid.
func Poll(ctx context.Context, db gorp.SqlExecutor, operationUUID string) (*sdk.Operation, error) {
	f := func() (*sdk.Operation, error) {
		ope, err := GetRepositoryOperation(ctx, db, operationUUID)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get repository operation %s", operationUUID)
		}
		switch ope.Status {
		case sdk.OperationStatusError, sdk.OperationStatusDone:
			return ope, nil
		default:
			return nil, nil
		}
	}

	ope, err := f()
	if ope != nil || err != nil {
		return ope, err
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, sdk.NewErrorFrom(sdk.ErrRepoOperationTimeout, "updating repository take too much time")
		case <-tick.C:
			ope, err := f()
			if ope != nil || err != nil {
				return ope, err
			}
		}
	}
}
