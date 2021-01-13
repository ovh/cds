package workflowtemplate

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func exportTemplate(wt sdk.WorkflowTemplate) (exportentities.Template, error) {
	e, err := exportentities.NewTemplate(wt)
	if err != nil {
		return exportentities.Template{}, err
	}
	return e, nil
}

// Pull writes the content of a template inside the given writer.
func Pull(ctx context.Context, wt *sdk.WorkflowTemplate, f exportentities.Format, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer func() {
		if err := tw.Close(); err != nil {
			log.Error(ctx, "%v", sdk.WrapError(err, "unable to close tar writer"))
		}
	}()

	ewt, err := exportTemplate(*wt)
	if err != nil {
		return sdk.WrapError(err, "unable to export template")
	}

	bufft, err := exportentities.Marshal(ewt, f)
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name: fmt.Sprintf(exportentities.PullWorkflowName, wt.Slug),
		Mode: 0644,
		Size: int64(len(bufft)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return sdk.WrapError(err, "Unable to write tmpl header %+v", hdr)
	}
	if _, err := io.Copy(tw, bytes.NewReader(bufft)); err != nil {
		return sdk.WrapError(err, "Unable to copy tmpl buffer")
	}

	data, err := base64.StdEncoding.DecodeString(wt.Workflow)
	if err != nil {
		return sdk.WrapError(err, "Unable to decode workflow value")
	}
	buffw := bytes.NewBuffer(data)
	hdr = &tar.Header{
		Name: fmt.Sprintf(exportentities.TemplateWorkflowName),
		Mode: 0644,
		Size: int64(buffw.Len()),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return sdk.WrapError(err, "Unable to write workflow header %+v", hdr)
	}
	if _, err := io.Copy(tw, buffw); err != nil {
		return sdk.WrapError(err, "Unable to copy workflow buffer")
	}

	for i, p := range wt.Pipelines {
		data, err := base64.StdEncoding.DecodeString(p.Value)
		if err != nil {
			return sdk.WrapError(err, "Unable to decode pipeline value")
		}
		buff := bytes.NewBuffer(data)
		hdr := &tar.Header{
			Name: fmt.Sprintf(exportentities.TemplatePipelineName, i+1),
			Mode: 0644,
			Size: int64(buff.Len()),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return sdk.WrapError(err, "Unable to write pipeline header %+v", hdr)
		}
		if _, err := io.Copy(tw, buff); err != nil {
			return sdk.WrapError(err, "Unable to copy pipeline buffer")
		}
	}

	for i, a := range wt.Applications {
		data, err := base64.StdEncoding.DecodeString(a.Value)
		if err != nil {
			return sdk.WrapError(err, "Unable to decode application value")
		}
		buff := bytes.NewBuffer(data)
		hdr := &tar.Header{
			Name: fmt.Sprintf(exportentities.TemplateApplicationName, i+1),
			Mode: 0644,
			Size: int64(buff.Len()),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return sdk.WrapError(err, "Unable to write application header %+v", hdr)
		}
		if _, err := io.Copy(tw, buff); err != nil {
			return sdk.WrapError(err, "Unable to copy application buffer")
		}
	}

	for i, e := range wt.Environments {
		data, err := base64.StdEncoding.DecodeString(e.Value)
		if err != nil {
			return sdk.WrapError(err, "Unable to decode environment value")
		}
		buff := bytes.NewBuffer(data)
		hdr := &tar.Header{
			Name: fmt.Sprintf(exportentities.TemplateEnvironmentName, i+1),
			Mode: 0644,
			Size: int64(buff.Len()),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return sdk.WrapError(err, "Unable to write environment header %+v", hdr)
		}
		if _, err := io.Copy(tw, buff); err != nil {
			return sdk.WrapError(err, "Unable to copy environment buffer")
		}
	}

	return nil
}
