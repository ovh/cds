package workflow

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func MigrateAsCode(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, wf *sdk.Workflow, encryptFunc sdk.EncryptFunc, u *sdk.User) (sdk.Operation, error) {
	// Get repository
	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return sdk.Operation{}, sdk.WithStack(sdk.ErrApplicationNotFound)
	}

	app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
	if app.VCSServer == "" || app.RepositoryFullname == "" {
		return sdk.Operation{}, sdk.WithStack(sdk.ErrRepoNotFound)
	}

	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, vcsServer)
	if errclient != nil {
		return sdk.Operation{}, sdk.WrapError(errclient, "Cannot get client")
	}

	repo, errR := client.RepoByFullname(ctx, app.RepositoryFullname)
	if errR != nil {
		return sdk.Operation{}, sdk.WrapError(errR, "cannot get repo %s", app.RepositoryFullname)
	}

	// Export workflow
	buf := new(bytes.Buffer)
	if err := Pull(ctx, db, store, proj, wf.Name, exportentities.FormatYAML, encryptFunc, u, buf); err != nil {
		return sdk.Operation{}, sdk.WrapError(err, "cannot pull workflow")
	}

	// Create VCS Operation
	ope := sdk.Operation{
		VCSServer:          app.VCSServer,
		RepoFullName:       app.RepositoryFullname,
		URL:                "",
		RepositoryStrategy: app.RepositoryStrategy,
		Setup: sdk.OperationSetup{
			Push: sdk.OperationPush{
				FromBranch: fmt.Sprintf("cdsAsCode-%d", time.Now().Unix()),
				Message:    "",
			},
		},
	}

	if app.RepositoryStrategy.ConnectionType == "ssh" {
		ope.URL = repo.SSHCloneURL
	} else {
		ope.URL = repo.HTTPCloneURL
	}

	if wf.FromRepository == "" {
		ope.Setup.Push.Message = fmt.Sprintf("feat: Enable workflow as code [@%s]", u.Username)
	} else {
		ope.Setup.Push.Message = fmt.Sprintf("fix: Update workflow [@%s]", u.Username)
	}

	if err := PostRepositoryOperation(ctx, db, store, *proj, &ope, buf); err != nil {
		return ope, sdk.WrapError(err, "unable to post repository operation")
	}

	return ope, nil
}
