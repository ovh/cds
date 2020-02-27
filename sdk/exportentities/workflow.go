package exportentities

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities/v1"
	"github.com/ovh/cds/sdk/exportentities/v2"
)

// Name pattern for pull files.
const (
	PullWorkflowName    = "%s.yml"
	PullPipelineName    = "%s.pip.yml"
	PullApplicationName = "%s.app.yml"
	PullEnvironmentName = "%s.env.yml"
)

type Options struct {
	SkipIfOnlyOneRepoWebhook bool
	WithPermission           bool
}

// WorkflowPulled contains all the yaml base64 that are needed to generate a workflow tar file.
type WorkflowPulled struct {
	Workflow     WorkflowPulledItem   `json:"workflow"`
	Pipelines    []WorkflowPulledItem `json:"pipelines"`
	Applications []WorkflowPulledItem `json:"applications"`
	Environments []WorkflowPulledItem `json:"environments"`
}

// WorkflowPulledItem contains data for a workflow item.
type WorkflowPulledItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Workflow interface {
	GetName() string
	GetVersion() string
}

const (
	WorkflowVersion1 = "v1.0"
	WorkflowVersion2 = "v2.0"
)

type WorkflowVersion struct {
	Version string `yaml:"version"`
}

func UnmarshalWorklow(body []byte) (Workflow, error) {
	var workflowVersion WorkflowVersion
	if err := yaml.Unmarshal(body, &workflowVersion); err != nil {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow data: %v", err)
	}
	switch workflowVersion.Version {
	case WorkflowVersion1:
		var workflowV1 v1.Workflow
		if err := yaml.Unmarshal(body, &workflowV1); err != nil {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow v1 format: %v", err)
		}
		return workflowV1, nil
	case WorkflowVersion2:
		var workflowV2 v2.Workflow
		if err := yaml.Unmarshal(body, &workflowV2); err != nil {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow v2 format: %v", err)
		}
		return workflowV2, nil
	}
	return nil, sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow version: %s", workflowVersion.Version)
}

func ParseWorkflow(exportWorkflow Workflow) (*sdk.Workflow, error) {
	switch exportWorkflow.GetVersion() {
	case WorkflowVersion2:
		workflowV2, ok := exportWorkflow.(v2.Workflow)
		if ok {
			return workflowV2.GetWorkflow()
		}
	case WorkflowVersion1:
		workflowV1, ok := exportWorkflow.(v1.Workflow)
		if ok {
			return workflowV1.GetWorkflow()
		}
	default:
		return nil, sdk.WithStack(fmt.Errorf("exportentities workflow cannot be cast, unknown version %s", exportWorkflow.GetVersion()))
	}
	return nil, sdk.WithStack(fmt.Errorf("exportentities workflow cannot be cast %+v", exportWorkflow))
}

func SetTemplate(w Workflow, path string) (Workflow, error) {
	switch w.GetVersion() {
	case WorkflowVersion2:
		workflowV2, ok := w.(v2.Workflow)
		if ok {
			workflowV2.Template = path
			return workflowV2, nil
		}
	case WorkflowVersion1:
		workflowV1, ok := w.(v1.Workflow)
		if ok {
			workflowV1.Template = path
			return workflowV1, nil
		}
	}
	return nil, sdk.WithStack(fmt.Errorf("exportentities workflow cannot be cast %+v", w))
}

func NewWorkflow(ctx context.Context, w sdk.Workflow, opts Options) (Workflow, error) {
	exportOptions := make([]v2.WorkflowOptions, 0)
	if opts.WithPermission {
		exportOptions = append(exportOptions, v2.WorkflowWithPermissions)
	}
	if opts.SkipIfOnlyOneRepoWebhook {
		exportOptions = append(exportOptions, v2.WorkflowSkipIfOnlyOneRepoWebhook)
	}
	workflowToExport, err := v2.NewWorkflow(ctx, w, WorkflowVersion2, exportOptions...)
	if err != nil {
		return workflowToExport, err
	}
	return workflowToExport, nil
}
