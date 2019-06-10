package workflow

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export a workflow
func Export(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, name string, f exportentities.Format, u *sdk.User, w io.Writer, opts ...exportentities.WorkflowOptions) (int, error) {
	ctx, end := observability.Span(ctx, "workflow.Export")
	defer end()

	wf, errload := Load(ctx, db, cache, proj, name, u, LoadOptions{})
	if errload != nil {
		return 0, sdk.WrapError(errload, "workflow.Export> Cannot load workflow %s", name)
	}

	// If repo is from as-code do not export WorkflowSkipIfOnlyOneRepoWebhook
	if wf.FromRepository != "" {
		opts = append(opts, exportentities.WorkflowSkipIfOnlyOneRepoWebhook)
	}

	return exportWorkflow(*wf, f, w, opts...)
}

func exportWorkflow(wf sdk.Workflow, f exportentities.Format, w io.Writer, opts ...exportentities.WorkflowOptions) (int, error) {
	e, err := exportentities.NewWorkflow(wf, opts...)
	if err != nil {
		return 0, sdk.WrapError(err, "exportWorkflow")
	}

	// Useful to not display history_length in yaml or json if it's his default value
	if e.HistoryLength != nil && *e.HistoryLength == sdk.DefaultHistoryLength {
		e.HistoryLength = nil
	}

	// Marshal to the desired format
	b, err := exportentities.Marshal(e, f)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return w.Write(b)
}

// Pull a workflow with all it dependencies; it writes a tar buffer in the writer
func Pull(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, name string, f exportentities.Format,
	encryptFunc sdk.EncryptFunc, u *sdk.User, opts ...exportentities.WorkflowOptions) (exportentities.WorkflowPulled, error) {
	ctx, end := observability.Span(ctx, "workflow.Pull")
	defer end()

	var wp exportentities.WorkflowPulled

	options := LoadOptions{
		DeepPipeline: true,
	}
	wf, errload := Load(ctx, db, cache, proj, name, u, options)
	if errload != nil {
		return wp, sdk.WrapError(errload, "cannot load workflow %s", name)
	}

	i, err := workflowtemplate.LoadInstanceByWorkflowID(ctx, db, wf.ID, workflowtemplate.LoadInstanceOptions.WithTemplate)
	if err != nil {
		return wp, err
	}
	if i != nil {
		wf.Template = i.Template
	}

	apps := wf.GetApplications()
	envs := wf.GetEnvironments()
	pips := wf.GetPipelines()

	//Reload app to retrieve secrets
	for i := range apps {
		app := &apps[i]
		vars, errv := application.GetAllVariable(db, proj.Key, app.Name, application.WithClearPassword())
		if errv != nil {
			return wp, sdk.WrapError(errv, "cannot load application variables %s", app.Name)
		}
		app.Variable = vars

		if err := application.LoadAllDecryptedKeys(db, app); err != nil {
			return wp, sdk.WrapError(err, "cannot load application keys %s", app.Name)
		}
	}

	//Reload env to retrieve secrets
	for i := range envs {
		env := &envs[i]
		vars, errv := environment.GetAllVariable(db, proj.Key, env.Name, environment.WithClearPassword())
		if errv != nil {
			return wp, sdk.WrapError(errv, "cannot load environment variables %s", env.Name)
		}
		env.Variable = vars

		if err := environment.LoadAllDecryptedKeys(db, env); err != nil {
			return wp, sdk.WrapError(err, "cannot load environment keys %s", env.Name)
		}
	}

	buffw := new(bytes.Buffer)
	// If the repository is "as-code", hide the hook
	if wf.FromRepository != "" {
		opts = append(opts, exportentities.WorkflowSkipIfOnlyOneRepoWebhook)
	}
	if _, err := exportWorkflow(*wf, f, buffw, opts...); err != nil {
		return wp, sdk.WrapError(err, "unable to export workflow")
	}
	wp.Workflow.Name = wf.Name
	wp.Workflow.Value = base64.StdEncoding.EncodeToString(buffw.Bytes())

	for _, a := range apps {
		if a.FromRepository != wf.FromRepository { // don't export if coming from an other repository
			continue
		}
		buff := new(bytes.Buffer)
		if _, err := application.ExportApplication(db, a, f, encryptFunc, buff); err != nil {
			return wp, sdk.WrapError(err, "unable to export app %s", a.Name)
		}
		wp.Applications = append(wp.Applications, exportentities.WorkflowPulledItem{
			Name:  a.Name,
			Value: base64.StdEncoding.EncodeToString(buff.Bytes()),
		})
	}

	for _, e := range envs {
		if e.FromRepository != wf.FromRepository { // don't export if coming from an other repository
			continue
		}
		buff := new(bytes.Buffer)
		if _, err := environment.ExportEnvironment(db, e, f, encryptFunc, buff); err != nil {
			return wp, sdk.WrapError(err, "unable to export env %s", e.Name)
		}
		wp.Environments = append(wp.Environments, exportentities.WorkflowPulledItem{
			Name:  e.Name,
			Value: base64.StdEncoding.EncodeToString(buff.Bytes()),
		})
	}

	for _, p := range pips {
		if p.FromRepository != wf.FromRepository { // don't export if coming from an other repository
			continue
		}
		buff := new(bytes.Buffer)
		if _, err := pipeline.ExportPipeline(p, f, buff); err != nil {
			return wp, sdk.WrapError(err, "unable to export pipeline %s", p.Name)
		}
		wp.Pipelines = append(wp.Pipelines, exportentities.WorkflowPulledItem{
			Name:  p.Name,
			Value: base64.StdEncoding.EncodeToString(buff.Bytes()),
		})
	}

	return wp, nil
}
