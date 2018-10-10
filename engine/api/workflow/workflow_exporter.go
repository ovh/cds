package workflow

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/pipeline"
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

	return exportWorkflow(*wf, f, w, opts...)
}

func exportWorkflow(wf sdk.Workflow, f exportentities.Format, w io.Writer, opts ...exportentities.WorkflowOptions) (int, error) {
	e, err := exportentities.NewWorkflow(wf, opts...)
	if err != nil {
		return 0, err
	}

	// Useful to not display history_length in yaml or json if it's his default value
	if e.HistoryLength == sdk.DefaultHistoryLength {
		e.HistoryLength = 0
	}

	// Marshal to the desired format
	b, err := exportentities.Marshal(e, f)
	if err != nil {
		return 0, sdk.WrapError(err, "workflow.Export>")
	}

	return w.Write(b)
}

// Pull a workflow with all it dependencies; it writes a tar buffer in the writer
func Pull(ctx context.Context, db gorp.SqlExecutor, cache cache.Store, proj *sdk.Project, name string, f exportentities.Format, encryptFunc sdk.EncryptFunc, u *sdk.User, w io.Writer, opts ...exportentities.WorkflowOptions) error {
	ctx, end := observability.Span(ctx, "workflow.Pull")
	defer end()

	options := LoadOptions{
		DeepPipeline: true,
	}
	wf, errload := Load(ctx, db, cache, proj, name, u, options)
	if errload != nil {
		return sdk.WrapError(errload, "workflow.Pull> Cannot load workflow %s", name)
	}

	apps := wf.GetApplications()
	envs := wf.GetEnvironments()
	pips := wf.GetPipelines()

	//Reload app to retrieve secrets
	for i := range apps {
		app := &apps[i]
		vars, errv := application.GetAllVariable(db, proj.Key, app.Name, application.WithClearPassword())
		if errv != nil {
			return sdk.WrapError(errv, "workflow.Pull> Cannot load application variables %s", app.Name)
		}
		app.Variable = vars

		if errk := application.LoadAllDecryptedKeys(db, app); errk != nil {
			return sdk.WrapError(errk, "workflow.Pull> Cannot load application keys %s", app.Name)
		}
	}

	//Reload env to retrieve secrets
	for i := range envs {
		env := &envs[i]
		vars, errv := environment.GetAllVariable(db, proj.Key, env.Name, environment.WithClearPassword())
		if errv != nil {
			return sdk.WrapError(errv, "workflow.Pull> Cannot load environment variables %s", env.Name)
		}
		env.Variable = vars

		if errk := environment.LoadAllDecryptedKeys(db, env); errk != nil {
			return sdk.WrapError(errk, "workflow.Pull> Cannot load environment keys %s", env.Name)
		}
	}

	tw := tar.NewWriter(w)

	buffw := new(bytes.Buffer)
	size, errw := exportWorkflow(*wf, f, buffw, opts...)
	if errw != nil {
		tw.Close()
		return sdk.WrapError(errw, "workflow.Pull> Unable to export workflow")
	}

	hdr := &tar.Header{
		Name: fmt.Sprintf("%s.yml", wf.Name),
		Mode: 0644,
		Size: int64(size),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		tw.Close()
		return sdk.WrapError(err, "workflow.Pull> Unable to write workflow header %+v", hdr)
	}
	if _, err := io.Copy(tw, buffw); err != nil {
		tw.Close()
		return sdk.WrapError(err, "workflow.Pull> Unable to copy workflow buffer")
	}

	var withPermissions bool
	for _, f := range opts {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(exportentities.WorkflowWithPermissions).Pointer() {
			withPermissions = true
		}
	}

	for _, a := range apps {
		buff := new(bytes.Buffer)
		size, err := application.ExportApplication(db, a, f, withPermissions, encryptFunc, buff)
		if err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to export app %s", a.Name)
		}
		hdr := &tar.Header{
			Name: fmt.Sprintf("%s.app.yml", a.Name),
			Mode: 0644,
			Size: int64(size),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to write app header %+v", hdr)
		}
		if _, err := io.Copy(tw, buff); err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to copy app buffer")
		}
	}

	for _, e := range envs {
		buff := new(bytes.Buffer)
		size, err := environment.ExportEnvironment(db, e, f, withPermissions, encryptFunc, buff)
		if err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to export env %s", e.Name)
		}

		hdr := &tar.Header{
			Name: fmt.Sprintf("%s.env.yml", e.Name),
			Mode: 0644,
			Size: int64(size),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to write env header %+v", hdr)
		}
		if _, err := io.Copy(tw, buff); err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to copy env buffer")
		}
	}

	for _, p := range pips {
		buff := new(bytes.Buffer)
		size, err := pipeline.ExportPipeline(p, f, withPermissions, buff)
		if err != nil {
			return sdk.WrapError(err, "workflow.Pull> Unable to export pipeline %s", p.Name)
		}
		hdr := &tar.Header{
			Name: fmt.Sprintf("%s.pip.yml", p.Name),
			Mode: 0644,
			Size: int64(size),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to write pipeline header %+v", hdr)
		}
		if _, err := io.Copy(tw, buff); err != nil {
			tw.Close()
			return sdk.WrapError(err, "workflow.Pull> Unable to copy pip buffer")
		}
	}

	if err := tw.Close(); err != nil {
		return sdk.WrapError(err, "workflow.Pull> Unable to close tar writer")
	}
	return nil
}
