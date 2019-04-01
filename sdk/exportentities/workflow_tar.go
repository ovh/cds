package exportentities

import (
	"archive/tar"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Tar returns a tar containing all files for a pulled workflow.
func (w WorkflowPulled) Tar(writer io.Writer) error {
	tw := tar.NewWriter(writer)
	defer func() {
		if err := tw.Close(); err != nil {
			log.Error("%v", sdk.WrapError(err, "unable to close tar writer"))
		}
	}()

	bs, err := base64.StdEncoding.DecodeString(w.Workflow.Value)
	if err != nil {
		return sdk.WithStack(err)
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: fmt.Sprintf(PullWorkflowName, w.Workflow.Name),
		Mode: 0644,
		Size: int64(len(bs)),
	}); err != nil {
		return sdk.WrapError(err, "unable to write workflow header for %s", w.Workflow.Name)
	}
	if _, err := tw.Write(bs); err != nil {
		return sdk.WrapError(err, "unable to write workflow value")
	}

	for _, a := range w.Applications {
		bs, err := base64.StdEncoding.DecodeString(a.Value)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf(PullApplicationName, a.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "unable to write application header for %s", a.Name)
		}
		if _, err := tw.Write(bs); err != nil {
			return sdk.WrapError(err, "unable to write application value")
		}
	}

	for _, e := range w.Environments {
		bs, err := base64.StdEncoding.DecodeString(e.Value)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf(PullEnvironmentName, e.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "unable to write env header for %s", e.Name)
		}
		if _, err := tw.Write(bs); err != nil {
			return sdk.WrapError(err, "unable to copy env buffer")
		}
	}

	for _, p := range w.Pipelines {
		bs, err := base64.StdEncoding.DecodeString(p.Value)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf(PullPipelineName, p.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "unable to write pipeline header for %s", p.Name)
		}
		if _, err := tw.Write(bs); err != nil {
			return sdk.WrapError(err, "unable to write pipeline value")
		}
	}

	return nil
}
