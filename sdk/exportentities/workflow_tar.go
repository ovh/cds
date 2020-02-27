package exportentities

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"gopkg.in/yaml.v2"
)

type WorkflowComponents struct {
	Template     TemplateInstance
	Workflow     Workflow
	Applications []Application
	Pipelines    []PipelineV1
	Environments []Environment
}

func (w WorkflowComponents) ToRaw() (WorkflowComponentsRaw, error) {
	res := WorkflowComponentsRaw{
		Applications: make([]string, len(w.Applications)),
		Pipelines:    make([]string, len(w.Pipelines)),
		Environments: make([]string, len(w.Environments)),
	}

	if w.Workflow != nil {
		bs, err := yaml.Marshal(w.Workflow)
		if err != nil {
			return res, sdk.WithStack(err)
		}
		res.Workflow = base64.StdEncoding.EncodeToString(bs)
	}

	for i, a := range w.Applications {
		bs, err := yaml.Marshal(a)
		if err != nil {
			return res, sdk.WithStack(err)
		}
		res.Applications[i] = base64.StdEncoding.EncodeToString(bs)
	}

	for i, p := range w.Pipelines {
		bs, err := yaml.Marshal(p)
		if err != nil {
			return res, sdk.WithStack(err)
		}
		res.Pipelines[i] = base64.StdEncoding.EncodeToString(bs)
	}

	for i, e := range w.Environments {
		bs, err := yaml.Marshal(e)
		if err != nil {
			return res, sdk.WithStack(err)
		}
		res.Environments[i] = base64.StdEncoding.EncodeToString(bs)
	}

	return res, nil
}

type WorkflowComponentsRaw struct {
	Workflow     string
	Applications []string
	Pipelines    []string
	Environments []string
}

// TarWorkflowComponents returns a tar containing all files for a workflow.
func TarWorkflowComponents(ctx context.Context, w WorkflowComponents, writer io.Writer) error {
	tw := tar.NewWriter(writer)
	defer func() {
		if err := tw.Close(); err != nil {
			log.Error(ctx, "%v", sdk.WrapError(err, "unable to close tar writer"))
		}
	}()

	if w.Workflow != nil {
		bs, err := yaml.Marshal(w.Workflow)
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf(PullWorkflowName, w.Workflow.GetName()),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "unable to write workflow header for %s", w.Workflow.GetName())
		}
		if _, err := tw.Write(bs); err != nil {
			return sdk.WrapError(err, "unable to write workflow value")
		}
	}

	for _, a := range w.Applications {
		bs, err := yaml.Marshal(a)
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
		bs, err := yaml.Marshal(e)
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
		bs, err := yaml.Marshal(p)
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

func UntarWorkflowComponents(ctx context.Context, tr *tar.Reader) (WorkflowComponents, error) {
	var res WorkflowComponents

	mError := new(sdk.MultiError)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read tar file"))
		}

		log.Debug("ExtractWorkflowFromTar> Reading %s", hdr.Name)

		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tr); err != nil {
			return res, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read tar file"))
		}

		var templateOrWorkflowFileName string
		b := buff.Bytes()
		switch {
		case strings.Contains(hdr.Name, ".app."):
			var app Application
			if err := yaml.Unmarshal(b, &app); err != nil {
				log.Error(ctx, "ExtractWorkflowFromTar> Unable to unmarshal application %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("unable to unmarshal application %s: %v", hdr.Name, err))
				continue
			}
			res.Applications = append(res.Applications, app)
		case strings.Contains(hdr.Name, ".pip."):
			var pip PipelineV1
			if err := yaml.Unmarshal(b, &pip); err != nil {
				log.Error(ctx, "ExtractWorkflowFromTar> Unable to unmarshal pipeline %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("unable to unmarshal pipeline %s: %v", hdr.Name, err))
				continue
			}
			res.Pipelines = append(res.Pipelines, pip)
		case strings.Contains(hdr.Name, ".env."):
			var env Environment
			if err := yaml.Unmarshal(b, &env); err != nil {
				log.Error(ctx, "ExtractWorkflowFromTar> Unable to unmarshal environment %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("unable to unmarshal environment %s: %v", hdr.Name, err))
				continue
			}
			res.Environments = append(res.Environments, env)
		default:
			if templateOrWorkflowFileName != "" {
				log.Error(ctx, "only one workflow or template file should be given: %s and %s",
					templateOrWorkflowFileName, hdr.Name)
				mError.Append(fmt.Errorf("only one workflow or template file should be given: %s and %s",
					templateOrWorkflowFileName, hdr.Name))
				break
			}

			templateOrWorkflowFileName = hdr.Name

			var tmp TemplateInstance
			isTemplate := yaml.UnmarshalStrict(b, &tmp) == nil && tmp.From != ""
			if isTemplate {
				res.Template = tmp
				continue
			}

			res.Workflow, err = UnmarshalWorkflow(b)
			if err != nil {
				log.Error(ctx, "Push> Unable to unmarshal workflow %s: %v", hdr.Name, err)
				mError.Append(fmt.Errorf("unable to unmarshal workflow %s: %v", hdr.Name, err))
				continue
			}
		}
	}

	// We only use the multiError during unmarshalling steps.
	// When a DB transaction has been started, just return at the first error
	// because transaction may have to be aborted
	if !mError.IsEmpty() {
		return res, sdk.NewError(sdk.ErrWorkflowInvalid, mError)
	}

	return res, nil
}
