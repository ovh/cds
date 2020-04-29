package workflow

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
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
	HookUUID           string
	Force              bool
	OldWorkflow        sdk.Workflow
}

// CreateFromRepository a workflow from a repository.
func CreateFromRepository(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, wf *sdk.Workflow,
	opts sdk.WorkflowRunPostHandlerOption, u sdk.AuthConsumer, decryptFunc keys.DecryptFunc) ([]sdk.Message, error) {
	ctx, end := observability.Span(ctx, "workflow.CreateFromRepository")
	defer end()

	ope, err := createOperationRequest(*wf, opts)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create operation request")
	}

	if err := operation.PostRepositoryOperation(ctx, db, *p, &ope, nil); err != nil {
		return nil, sdk.WrapError(err, "unable to post repository operation")
	}

	if err := pollRepositoryOperation(ctx, db, store, &ope); err != nil {
		return nil, sdk.WrapError(err, "cannot analyse repository")
	}

	var uuid string
	if opts.Hook != nil {
		uuid = opts.Hook.WorkflowNodeHookUUID
	} else {
		// Search for repo web hook uuid
		for _, h := range wf.WorkflowData.Node.Hooks {
			if h.HookModelName == sdk.RepositoryWebHookModelName {
				uuid = h.UUID
				break
			}
		}
	}
	return extractWorkflow(ctx, db, store, p, wf, ope, u, decryptFunc, uuid)
}

func extractWorkflow(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, wf *sdk.Workflow,
	ope sdk.Operation, consumer sdk.AuthConsumer, decryptFunc keys.DecryptFunc, hookUUID string) ([]sdk.Message, error) {
	ctx, end := observability.Span(ctx, "workflow.extractWorkflow")
	defer end()
	var allMsgs []sdk.Message
	// Read files
	tr, err := ReadCDSFiles(ope.LoadFiles.Results)
	if err != nil {
		allMsgs = append(allMsgs, sdk.NewMessage(sdk.MsgWorkflowErrorBadCdsDir))
		return allMsgs, sdk.WrapError(err, "unable to read cds files")
	}
	ope.RepositoryStrategy.SSHKeyContent = ""
	opt := &PushOption{
		VCSServer:          ope.VCSServer,
		RepositoryName:     ope.RepoFullName,
		RepositoryStrategy: ope.RepositoryStrategy,
		Branch:             ope.Setup.Checkout.Branch,
		FromRepository:     ope.RepositoryInfo.FetchURL,
		IsDefaultBranch:    ope.Setup.Checkout.Tag == "" && ope.Setup.Checkout.Branch == ope.RepositoryInfo.DefaultBranch,
		HookUUID:           hookUUID,
		OldWorkflow:        *wf,
	}

	data, err := exportentities.UntarWorkflowComponents(ctx, tr)
	if err != nil {
		return allMsgs, err
	}

	mods := []workflowtemplate.TemplateRequestModifierFunc{
		workflowtemplate.TemplateRequestModifiers.DefaultKeys(*p),
	}
	if !opt.IsDefaultBranch {
		mods = append(mods, workflowtemplate.TemplateRequestModifiers.Detached)
	}
	if opt.FromRepository != "" {
		mods = append(mods, workflowtemplate.TemplateRequestModifiers.DefaultNameAndRepositories(ctx, db, store, *p, opt.FromRepository))
	}
	msgTemplate, wti, err := workflowtemplate.CheckAndExecuteTemplate(ctx, db, consumer, *p, &data, mods...)
	allMsgs = append(allMsgs, msgTemplate...)
	if err != nil {
		return allMsgs, err
	}
	msgPush, workflowPushed, _, err := Push(ctx, db, store, p, data, opt, consumer, decryptFunc)
	// Filter workflow push message if generated from template
	for i := range msgPush {
		if wti != nil && msgPush[i].ID == sdk.MsgWorkflowDeprecatedVersion.ID {
			continue
		}
		allMsgs = append(allMsgs, msgPush[i])
	}
	if err != nil {
		return allMsgs, sdk.WrapError(err, "unable to get workflow from file")
	}
	if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, db, *workflowPushed, consumer, wti); err != nil {
		return allMsgs, err
	}
	*wf = *workflowPushed

	if wf.Name != workflowPushed.Name {
		log.Debug("workflow.extractWorkflow> Workflow has been renamed from %s to %s", wf.Name, workflowPushed.Name)
	}

	return allMsgs, nil
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
			return nil, sdk.WrapError(err, "cannot write header")
		}
		if n, err := tw.Write(fcontent); err != nil {
			return nil, sdk.WrapError(err, "cannot write content")
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
	var err error
	tickTimeout := time.NewTicker(10 * time.Minute)
	tickPoll := time.NewTicker(2 * time.Second)
	defer tickTimeout.Stop()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				return sdk.WrapError(c.Err(), "exiting")
			}
		case <-tickTimeout.C:
			return sdk.WrapError(sdk.ErrRepoOperationTimeout, "timeout analyzing repository")
		case <-tickPoll.C:
			ope, err = operation.GetRepositoryOperation(c, db, ope.UUID)
			if err != nil {
				return sdk.WrapError(err, "cannot get repository operation status")
			}
			switch ope.Status {
			case sdk.OperationStatusError:
				opeTrusted := *ope
				opeTrusted.RepositoryStrategy.SSHKeyContent = "***"
				opeTrusted.RepositoryStrategy.Password = "***"
				return sdk.WrapError(fmt.Errorf("%s", ope.Error), "getImportAsCodeHandler> Operation in error. %+v", opeTrusted)
			case sdk.OperationStatusDone:
				return nil
			}
			continue
		}
	}
}

func createOperationRequest(w sdk.Workflow, opts sdk.WorkflowRunPostHandlerOption) (sdk.Operation, error) {
	ope := sdk.Operation{}
	if w.WorkflowData.Node.Context.ApplicationID == 0 {
		return ope, sdk.WrapError(sdk.ErrNotFound, "workflow node root does not have a application context")
	}
	app := w.Applications[w.WorkflowData.Node.Context.ApplicationID]
	ope = sdk.Operation{
		VCSServer:          app.VCSServer,
		RepoFullName:       app.RepositoryFullname,
		URL:                w.FromRepository,
		RepositoryStrategy: app.RepositoryStrategy,
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

	var branch, commit, tag string
	if opts.Hook != nil {
		tag = opts.Hook.Payload[tagGitTag]
		branch = opts.Hook.Payload[tagGitBranch]
		commit = opts.Hook.Payload[tagGitHash]
	}
	if opts.Manual != nil {
		e := dump.NewDefaultEncoder()
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false
		m1, errm1 := e.ToStringMap(opts.Manual.Payload)
		if errm1 != nil {
			return ope, sdk.WrapError(errm1, "CreateFromRepository> Unable to compute payload")
		}
		tag = m1[tagGitTag]
		branch = m1[tagGitBranch]
		commit = m1[tagGitHash]
	}
	ope.Setup.Checkout.Tag = tag
	ope.Setup.Checkout.Commit = commit
	ope.Setup.Checkout.Branch = branch

	// This should not append because the hook must set a default payload with git.branch
	if ope.Setup.Checkout.Branch == "" && ope.Setup.Checkout.Tag == "" {
		return ope, sdk.WrapError(sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("branch or tag parameter are mandatories")), "createOperationRequest")
	}

	return ope, nil
}
