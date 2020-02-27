package exportentities

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	v1 "github.com/ovh/cds/sdk/exportentities/v1"
	v2 "github.com/ovh/cds/sdk/exportentities/v2"
)

// Name pattern for pull files.
const (
	PullWorkflowName    = "%s.yml"
	PullPipelineName    = "%s.pip.yml"
	PullApplicationName = "%s.app.yml"
	PullEnvironmentName = "%s.env.yml"
)

// WorkflowPulled contains all the yaml base64 that are needed to generate a workflow tar file.
type WorkflowPulled struct {
	Workflow     WorkflowPulledItem   `json:"workflow"`
	Pipelines    []WorkflowPulledItem `json:"pipelines"`
	Applications []WorkflowPulledItem `json:"applications"`
	Environments []WorkflowPulledItem `json:"environments"`
}

type Options struct {
	SkipIfOnlyOneRepoWebhook bool
	WithPermission           bool
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

func UnmarshalWorkflow(body []byte) (Workflow, error) {
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

func NewWorkflow(ctx context.Context, w sdk.Workflow, opts ...v2.ExportOptions) (Workflow, error) {
	workflowToExport, err := v2.NewWorkflow(ctx, w, WorkflowVersion2, opts...)
	if err != nil {
		return workflowToExport, err
	}
	return workflowToExport, nil
}

func InitWorkflow(workName, appName, pipName string) Workflow {
	return v2.Workflow{
		Version: WorkflowVersion2,
		Name:    workName,
		Workflow: map[string]v2.NodeEntry{
			pipName: {
				ApplicationName: appName,
				PipelineName:    pipName,
			},
		},
	}
}
