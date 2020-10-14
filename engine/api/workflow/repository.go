package workflow

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
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
	opts sdk.WorkflowRunPostHandlerOption, u sdk.AuthConsumer, decryptFunc keys.DecryptFunc) (*PushSecrets, []sdk.Message, error) {
	ctx, end := telemetry.Span(ctx, "workflow.CreateFromRepository")
	defer end()

	newOperation, err := createOperationRequest(*wf, opts)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "unable to create operation request")
	}

	if err := operation.PostRepositoryOperation(ctx, db, *p, &newOperation, nil); err != nil {
		return nil, nil, sdk.WrapError(err, "unable to post repository operation")
	}

	log.Info(ctx, "polling operation %v for workflow %s/%s", newOperation.UUID, p.Key, wf.Name)
	ope, err := operation.Poll(ctx, db, newOperation.UUID)

	if err != nil {
		isErrWithStack := sdk.IsErrorWithStack(err)
		fields := log.Fields{}
		if isErrWithStack {
			fields["stack_trace"] = fmt.Sprintf("%+v", err)
		}
		log.ErrorWithFields(ctx, fields, "cannot analyse repository (operation %s for workflow %s/%s): %v", newOperation.UUID, p.Key, wf.Name, err)
		return nil, nil, sdk.NewError(sdk.ErrRepoAnalyzeFailed, err)
	}

	if ope.Status == sdk.OperationStatusError {
		err := ope.Error.ToError()
		isErrWithStack := sdk.IsErrorWithStack(err)
		fields := log.Fields{}
		if isErrWithStack {
			fields["stack_trace"] = fmt.Sprintf("%+v", err)
		}
		log.ErrorWithFields(ctx, fields, "cannot analyse repository (operation %s for workflow %s/%s): %v", newOperation.UUID, p.Key, wf.Name, err)
		return nil, nil, sdk.NewError(sdk.ErrRepoAnalyzeFailed, err)
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
	return extractWorkflow(ctx, db, store, p, wf, *ope, u, decryptFunc, uuid)
}

func extractWorkflow(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, wf *sdk.Workflow,
	ope sdk.Operation, consumer sdk.AuthConsumer, decryptFunc keys.DecryptFunc, hookUUID string) (*PushSecrets, []sdk.Message, error) {
	ctx, end := telemetry.Span(ctx, "workflow.extractWorkflow")
	defer end()
	var allMsgs []sdk.Message
	// Read files
	tr, err := ReadCDSFiles(ope.LoadFiles.Results)
	if err != nil {
		allMsgs = append(allMsgs, sdk.NewMessage(sdk.MsgWorkflowErrorBadCdsDir))
		return nil, allMsgs, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWorkflowInvalid, "unable to read cds files"))
	}

	ope.RepositoryStrategy.SSHKeyContent = sdk.PasswordPlaceholder
	ope.RepositoryStrategy.Password = sdk.PasswordPlaceholder
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
		return nil, allMsgs, err
	}

	mods := []workflowtemplate.TemplateRequestModifierFunc{
		workflowtemplate.TemplateRequestModifiers.DefaultKeys(*p),
	}
	if !opt.IsDefaultBranch {
		mods = append(mods, workflowtemplate.TemplateRequestModifiers.Detached)
	}
	if opt.FromRepository != "" {
		mods = append(mods, workflowtemplate.TemplateRequestModifiers.DefaultNameAndRepositories(*p, opt.FromRepository))
	}
	msgTemplate, wti, err := workflowtemplate.CheckAndExecuteTemplate(ctx, db, store, consumer, *p, &data, mods...)
	allMsgs = append(allMsgs, msgTemplate...)
	if err != nil {
		return nil, allMsgs, err
	}
	msgPush, workflowPushed, _, secrets, err := Push(ctx, db, store, p, data, opt, consumer, decryptFunc)
	// Filter workflow push message if generated from template
	for i := range msgPush {
		if wti != nil && msgPush[i].ID == sdk.MsgWorkflowDeprecatedVersion.ID {
			continue
		}
		allMsgs = append(allMsgs, msgPush[i])
	}
	if err != nil {
		return nil, allMsgs, sdk.WrapError(err, "unable to get workflow from file")
	}
	if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, db, *workflowPushed, consumer, wti); err != nil {
		return nil, allMsgs, err
	}
	*wf = *workflowPushed

	if wf.Name != workflowPushed.Name {
		log.Debug("workflow.extractWorkflow> Workflow has been renamed from %s to %s", wf.Name, workflowPushed.Name)
	}

	return secrets, allMsgs, nil
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
		if _, err := tw.Write(fcontent); err != nil {
			return nil, sdk.WrapError(err, "cannot write content")
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		return nil, err
	}

	return tar.NewReader(buf), nil
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
			return ope, sdk.WrapError(errm1, "unable to compute payload")
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
		return ope, sdk.NewErrorFrom(sdk.ErrWrongRequest, "branch or tag parameter are mandatories")
	}

	return ope, nil
}
