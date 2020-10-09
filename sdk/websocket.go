package sdk

import "fmt"

type WebsocketFilterType string

const (
	WebsocketFilterTypeGlobal                  WebsocketFilterType = "global"
	WebsocketFilterTypeProject                 WebsocketFilterType = "project"
	WebsocketFilterTypeWorkflow                WebsocketFilterType = "workflow"
	WebsocketFilterTypeWorkflowRun             WebsocketFilterType = "workflow-run"
	WebsocketFilterTypeWorkflowNodeRun         WebsocketFilterType = "workflow-node-run"
	WebsocketFilterTypePipeline                WebsocketFilterType = "pipeline"
	WebsocketFilterTypeApplication             WebsocketFilterType = "application"
	WebsocketFilterTypeEnvironment             WebsocketFilterType = "environment"
	WebsocketFilterTypeQueue                   WebsocketFilterType = "queue"
	WebsocketFilterTypeOperation               WebsocketFilterType = "operation"
	WebsocketFilterTypeTimeline                WebsocketFilterType = "timeline"
	WebsocketFilterTypeAscodeEvent             WebsocketFilterType = "ascode-event"
	WebsocketFilterTypeDryRunRetentionWorkflow WebsocketFilterType = "workflow-retention-dryrun"
)

func (f WebsocketFilterType) IsValid() bool {
	switch f {
	case WebsocketFilterTypeGlobal,
		WebsocketFilterTypeProject,
		WebsocketFilterTypeWorkflow,
		WebsocketFilterTypeWorkflowRun,
		WebsocketFilterTypeWorkflowNodeRun,
		WebsocketFilterTypePipeline,
		WebsocketFilterTypeApplication,
		WebsocketFilterTypeEnvironment,
		WebsocketFilterTypeQueue,
		WebsocketFilterTypeOperation,
		WebsocketFilterTypeTimeline,
		WebsocketFilterTypeDryRunRetentionWorkflow,
		WebsocketFilterTypeAscodeEvent:
		return true
	}
	return false
}

type WebsocketFilters []WebsocketFilter

type WebsocketFilter struct {
	Type              WebsocketFilterType `json:"type"`
	ProjectKey        string              `json:"project_key"`
	ApplicationName   string              `json:"application_name"`
	PipelineName      string              `json:"pipeline_name"`
	EnvironmentName   string              `json:"environment_name"`
	WorkflowName      string              `json:"workflow_name"`
	WorkflowRunNumber int64               `json:"workflow_run_num"`
	WorkflowNodeRunID int64               `json:"workflow_node_run_id"`
	OperationUUID     string              `json:"operation_uuid"`
}

// Key generates the unique key associated to given filter.
func (f WebsocketFilter) Key() string {
	switch f.Type {
	case WebsocketFilterTypeProject:
		return fmt.Sprintf("%s-%s", f.Type, f.ProjectKey)
	case WebsocketFilterTypeWorkflow, WebsocketFilterTypeAscodeEvent:
		return fmt.Sprintf("%s-%s-%s", f.Type, f.ProjectKey, f.WorkflowName)
	case WebsocketFilterTypeWorkflowRun:
		return fmt.Sprintf("%s-%s-%s-%d", f.Type, f.ProjectKey, f.WorkflowName, f.WorkflowRunNumber)
	case WebsocketFilterTypeWorkflowNodeRun:
		return fmt.Sprintf("%s-%s-%s-%d", f.Type, f.ProjectKey, f.WorkflowName, f.WorkflowNodeRunID)
	case WebsocketFilterTypePipeline:
		return fmt.Sprintf("%s-%s", f.Type, f.PipelineName)
	case WebsocketFilterTypeApplication:
		return fmt.Sprintf("%s-%s", f.Type, f.ApplicationName)
	case WebsocketFilterTypeEnvironment:
		return fmt.Sprintf("%s-%s", f.Type, f.EnvironmentName)
	case WebsocketFilterTypeOperation:
		return fmt.Sprintf("%s-%s-%s", f.Type, f.ProjectKey, f.OperationUUID)
	case WebsocketFilterTypeDryRunRetentionWorkflow:
		return fmt.Sprintf("%s-%s-%s", f.Type, f.ProjectKey, f.WorkflowName)
	default:
		return string(f.Type)
	}
}

// IsValid return an error if given filter is not valid.
func (f WebsocketFilter) IsValid() error {
	if !f.Type.IsValid() {
		return NewErrorFrom(ErrWrongRequest, "invalid or empty given filter type: %s", f.Type)
	}

	switch f.Type {
	case WebsocketFilterTypeProject:
		if f.ProjectKey == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key")
		}
	case WebsocketFilterTypeWorkflow, WebsocketFilterTypeAscodeEvent:
		if f.ProjectKey == "" || f.WorkflowName == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key or workflow name")
		}
	case WebsocketFilterTypeWorkflowRun:
		if f.ProjectKey == "" || f.WorkflowName == "" || f.WorkflowRunNumber == 0 {
			return NewErrorFrom(ErrWrongRequest, "missing project key, workflow name or run number")
		}
	case WebsocketFilterTypeWorkflowNodeRun:
		if f.ProjectKey == "" || f.WorkflowName == "" || f.WorkflowNodeRunID == 0 {
			return NewErrorFrom(ErrWrongRequest, "missing project key, workflow name or node run id")
		}
	case WebsocketFilterTypePipeline:
		if f.ProjectKey == "" || f.PipelineName == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key or pipeline name")
		}
	case WebsocketFilterTypeApplication:
		if f.ProjectKey == "" || f.ApplicationName == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key or application name")
		}
	case WebsocketFilterTypeEnvironment:
		if f.ProjectKey == "" || f.EnvironmentName == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key or environment name")
		}
	case WebsocketFilterTypeOperation:
		if f.ProjectKey == "" || f.OperationUUID == "" {
			return NewErrorFrom(ErrWrongRequest, "missing project key or operation uuid")
		}
	}

	return nil
}

type WebsocketEvent struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Event  Event  `json:"event"`
}
