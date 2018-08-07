package workflow

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WorkflowAsCodePattern is the default code pattern to find cds files
const WorkflowAsCodePattern = ".cds/**/*.yml"

// PushOption is the set of options for workflow push
type PushOption struct {
	VCSServer          string
	FromRepository     string
	Branch             string
	IsDefaultBranch    bool
	RepositoryName     string
	RepositoryStrategy sdk.RepositoryStrategy
	DryRun             bool
	HookUUID           string
}

// CreateFromRepository a workflow from a repository
func CreateFromRepository(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, w *sdk.Workflow, opts sdk.WorkflowRunPostHandlerOption, u *sdk.User, decryptFunc keys.DecryptFunc) ([]sdk.Message, error) {
	ctx, end := observability.Span(ctx, "workflow.CreateFromRepository")
	defer end()

	ope, err := createOperationRequest(*w, opts)
	if err != nil {
		return nil, sdk.WrapError(err, "CreateFromRepository> Unable to create operation request")
	}

	// update user permission if  we come from hook
	if opts.Hook != nil {
		u.Groups = make([]sdk.Group, len(p.ProjectGroups))
		for i, gp := range p.ProjectGroups {
			u.Groups[i] = gp.Group
		}
	}

	if err := PostRepositoryOperation(ctx, db, store, *p, &ope); err != nil {
		return nil, sdk.WrapError(err, "CreateFromRepository> Unable to post repository operation")
	}

	if err := pollRepositoryOperation(ctx, db, store, &ope); err != nil {
		return nil, sdk.WrapError(err, "CreateFromRepository> Cannot analyse repository")
	}

	var uuid string
	if opts.Hook != nil {
		uuid = opts.Hook.WorkflowNodeHookUUID
	}
	allMsg, errE := extractWorkflow(ctx, db, store, p, w, ope, u, decryptFunc, uuid)
	if errE != nil {
		return nil, sdk.WrapError(err, "CreateFromRepository> Unable to extract workflow")
	}

	return allMsg, nil
}

func extractWorkflow(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, w *sdk.Workflow, ope sdk.Operation, u *sdk.User, decryptFunc keys.DecryptFunc, hookUUID string) ([]sdk.Message, error) {
	ctx, end := observability.Span(ctx, "workflow.extractWorkflow")
	defer end()

	// Read files
	tr, err := ReadCDSFiles(ope.LoadFiles.Results)
	if err != nil {
		return nil, sdk.WrapError(err, "extractWorkflow> Unable to read cds files")
	}
	ope.RepositoryStrategy.SSHKeyContent = ""
	opt := &PushOption{
		VCSServer:          ope.VCSServer,
		RepositoryName:     ope.RepoFullName,
		RepositoryStrategy: ope.RepositoryStrategy,
		Branch:             ope.Setup.Checkout.Branch,
		FromRepository:     ope.RepositoryInfo.FetchURL,
		IsDefaultBranch:    ope.Setup.Checkout.Branch == ope.RepositoryInfo.DefaultBranch,
		DryRun:             true,
		HookUUID:           hookUUID,
	}

	allMsg, workflowPushed, errP := Push(ctx, db, store, p, tr, opt, u, decryptFunc)
	if errP != nil {
		return nil, sdk.WrapError(errP, "extractWorkflow> Unable to get workflow from file")
	}
	*w = *workflowPushed
	return allMsg, nil
}

// ReadCDSFiles reads CDS files
func ReadCDSFiles(files map[string][]byte) (*tar.Reader, error) {
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	// Create a new tar archive.
	tw := tar.NewWriter(buf)
	// Add some files to the archive.
	for fname, fcontent := range files {
		log.Debug("ReadCDSFiles> Reading %s", fname)
		hdr := &tar.Header{
			Name: filepath.Base(fname),
			Mode: 0600,
			Size: int64(len(fcontent)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, sdk.WrapError(err, "ReadCDSFiles> Cannot write header")
		}
		if n, err := tw.Write(fcontent); err != nil {
			return nil, sdk.WrapError(err, "ReadCDSFiles> Cannot write content")
		} else if n == 0 {
			return nil, fmt.Errorf("nothing to write")
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		return nil, err
	}

	return tar.NewReader(buf), nil
}

func pollRepositoryOperation(c context.Context, db gorp.SqlExecutor, store cache.Store, ope *sdk.Operation) error {
	tickTimeout := time.NewTicker(10 * time.Minute)
	tickPoll := time.NewTicker(2 * time.Second)
	defer tickTimeout.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				return sdk.WrapError(c.Err(), "pollRepositoryOperation> Exiting")
			}
		case <-tickTimeout.C:
			return sdk.WrapError(sdk.ErrRepoOperationTimeout, "pollRepositoryOperation> Timeout analyzing repository")
		case <-tickPoll.C:
			if err := GetRepositoryOperation(c, db, store, ope); err != nil {
				return sdk.WrapError(err, "pollRepositoryOperation> Cannot get repository operation status")
			}
			switch ope.Status {
			case sdk.OperationStatusError:
				return sdk.WrapError(fmt.Errorf("%s", ope.Error), "getImportAsCodeHandler> Operation in error. %+v", ope)
			case sdk.OperationStatusDone:
				return nil
			}
			continue
		}
	}
}

func createOperationRequest(w sdk.Workflow, opts sdk.WorkflowRunPostHandlerOption) (sdk.Operation, error) {
	ope := sdk.Operation{}
	if w.Root.Context.Application == nil {
		return ope, sdk.WrapError(sdk.ErrApplicationNotFound, "CreateFromRepository> Workflow node root does not have a application context")
	}
	ope = sdk.Operation{
		VCSServer:          w.Root.Context.Application.VCSServer,
		RepoFullName:       w.Root.Context.Application.RepositoryFullname,
		URL:                w.FromRepository,
		RepositoryStrategy: w.Root.Context.Application.RepositoryStrategy,
		Setup: sdk.OperationSetup{
			Checkout: sdk.OperationCheckout{
				Branch: "",
				Commit: "",
			},
		},
		LoadFiles: sdk.OperationLoadFiles{
			Pattern: WorkflowAsCodePattern,
		},
	}

	var branch, commit string
	if opts.Hook != nil {
		branch = opts.Hook.Payload[tagGitBranch]
		commit = opts.Hook.Payload[tagGitHash]
	}
	if opts.Manual != nil {
		e := dump.NewDefaultEncoder(new(bytes.Buffer))
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false
		m1, errm1 := e.ToStringMap(opts.Manual.Payload)
		if errm1 != nil {
			return ope, sdk.WrapError(errm1, "CreateFromRepository> Unable to compute payload")
		}
		branch = m1[tagGitBranch]
		commit = m1[tagGitHash]
	}
	ope.Setup.Checkout.Commit = commit
	ope.Setup.Checkout.Branch = branch

	// This should not append because the hook must set a default payload with git.branch
	if ope.Setup.Checkout.Branch == "" {
		return ope, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("branch parameter is mandatory")), "createOperationRequest")
	}

	return ope, nil
}

// PostRepositoryOperation creates a new repository operation
func PostRepositoryOperation(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, prj sdk.Project, ope *sdk.Operation) error {
	srvs, err := services.FindByType(db, services.TypeRepositories)
	if err != nil {
		return sdk.WrapError(err, "PostRepositoryOperation> Unable to found repositories service")
	}

	if ope.RepositoryStrategy.ConnectionType == "ssh" {
		for _, k := range prj.Keys {
			if k.Name == ope.RepositoryStrategy.SSHKey {
				ope.RepositoryStrategy.SSHKeyContent = k.Private
				break
			}
		}
	}
	if _, err := services.DoJSONRequest(ctx, srvs, http.MethodPost, "/operations", ope, ope); err != nil {
		return sdk.WrapError(err, "PostRepositoryOperation> Unable to perform operation")
	}
	return nil
}

// GetRepositoryOperation get repository operation status
func GetRepositoryOperation(ctx context.Context, db gorp.SqlExecutor, store cache.Store, ope *sdk.Operation) error {
	srvs, err := services.FindByType(db, services.TypeRepositories)
	if err != nil {
		return sdk.WrapError(err, "GetRepositoryOperation> Unable to found repositories service")
	}

	if _, err := services.DoJSONRequest(ctx, srvs, http.MethodGet, "/operations/"+ope.UUID, nil, ope); err != nil {
		return sdk.WrapError(err, "GetRepositoryOperation> Unable to get operation")
	}
	return nil
}
