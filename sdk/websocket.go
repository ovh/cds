package sdk

const (
	WebsocketFilterTypeProject     = "project"
	WebsocketFilterTypeApplication = "application"
	WebsocketFilterTypePipeline    = "pipeline"
	WebsocketFilterTypeEnvironment = "environment"
	WebsocketFilterTypeWorkflow    = "workflow"
	WebsocketFilterTypeQueue       = "queue"
)

type WebsocketFilter struct {
	Type              string `json:"type"`
	ProjectKey        string `json:"project_key"`
	ApplicationName   string `json:"application_name"`
	PipelineName      string `json:"pipeline_name"`
	EnvironmentName   string `json:"environment_name"`
	WorkflowName      string `json:"workflow_name"`
	WorkflowRunNumber int64  `json:"workflow_run_num"`
	WorkflowNodeRunID int64  `json:"workflow_node_run_id"`
	Favorites         bool   `json:"favorites"`
	Queue             bool   `json:"queue"`
	Operation         string `json:"operation"`
}

type WebsocketEvent struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Event  Event  `json:"event"`
}
