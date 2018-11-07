package workflowtemplate

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func exportTemplate(wt sdk.WorkflowTemplate, f exportentities.Format, w io.Writer) (int, error) {
	e, err := exportentities.NewTemplate(wt)
	if err != nil {
		return 0, err
	}

	b, err := exportentities.Marshal(e, f)
	if err != nil {
		return 0, err
	}

	n, err := w.Write(b)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return n, nil
}

// Pull writes the content of a template inside the given writer.
func Pull(wt *sdk.WorkflowTemplate, f exportentities.Format, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer func() {
		if err := tw.Close(); err != nil {
			log.Error("%v", sdk.WrapError(err, "Unable to close tar writer"))
		}
	}()

	bufft := new(bytes.Buffer)
	size, errw := exportTemplate(*wt, f, bufft)
	if errw != nil {
		return sdk.WrapError(errw, "Unable to export template")
	}
	hdr := &tar.Header{
		Name: fmt.Sprintf("%s.yml", wt.Slug),
		Mode: 0644,
		Size: int64(size),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return sdk.WrapError(err, "Unable to write tmpl header %+v", hdr)
	}
	if _, err := io.Copy(tw, bufft); err != nil {
		return sdk.WrapError(err, "Unable to copy tmpl buffer")
	}

	data, err := base64.StdEncoding.DecodeString(wt.Value)
	if err != nil {
		return sdk.WrapError(err, "Unable to decode workflow value")
	}
	buffw := bytes.NewBuffer(data)
	hdr = &tar.Header{
		Name: fmt.Sprintf("workflow.yml"),
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
			Name: fmt.Sprintf("%d.pipeline.yml", i+1),
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
			Name: fmt.Sprintf("%d.application.yml", i+1),
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
			Name: fmt.Sprintf("%d.env.yml", i+1),
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
