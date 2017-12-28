package workflow

import (
	"database/sql"
	"time"

	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// Workflow is a gorp wrapper around sdk.Workflow
type Workflow sdk.Workflow

// Node is a gorp wrapper around sdk.WorkflowNode
type Node sdk.WorkflowNode

// NodeContext is a gorp wrapper around sdk.WorkflowNodeContext
type NodeContext sdk.WorkflowNodeContext

// NodeTrigger is a gorp wrapper around sdk.WorkflowNodeTrigger
type NodeTrigger sdk.WorkflowNodeTrigger

// Join is a gorp wrapper around sdk.WorkflowNodeJoin
type Join sdk.WorkflowNodeJoin

// JoinTrigger  is a gorp wrapper around sdk.WorkflowNodeJoinTrigger
type JoinTrigger sdk.WorkflowNodeJoinTrigger

// Notification is a gorp wrapper around sdk.WorkflowNotification
type Notification sdk.WorkflowNotification

// Run is a gorp wrapper around sdk.WorkflowRun
type Run sdk.WorkflowRun

// NodeRun is a gorp wrapper around sdk.WorkflowNodeRun
type NodeRun struct {
	WorkflowRunID      int64          `db:"workflow_run_id"`
	ID                 int64          `db:"id"`
	WorkflowNodeID     int64          `db:"workflow_node_id"`
	Number             int64          `db:"num"`
	SubNumber          int64          `db:"sub_num"`
	Status             string         `db:"status"`
	Start              time.Time      `db:"start"`
	Done               time.Time      `db:"done"`
	LastModified       time.Time      `db:"last_modified"`
	HookEvent          sql.NullString `db:"hook_event"`
	Manual             sql.NullString `db:"manual"`
	SourceNodeRuns     sql.NullString `db:"source_node_runs"`
	Payload            sql.NullString `db:"payload"`
	PipelineParameters sql.NullString `db:"pipeline_parameters"`
	BuildParameters    sql.NullString `db:"build_parameters"`
	Tests              sql.NullString `db:"tests"`
	Commits            sql.NullString `db:"commits"`
	Stages             sql.NullString `db:"stages"`
	TriggersRun        sql.NullString `db:"triggers_run"`
	VCSRepository      sql.NullString `db:"vcs_repository"`
	VCSBranch          sql.NullString `db:"vcs_branch"`
	VCSHash            sql.NullString `db:"vcs_hash"`
}

// JobRun is a gorp wrapper around sdk.WorkflowNodeJobRun
type JobRun sdk.WorkflowNodeJobRun

// NodeRunArtifact is a gorp wrapper around sdk.WorkflowNodeRunArtifact
type NodeRunArtifact sdk.WorkflowNodeRunArtifact

// RunTag is a gorp wrapper around sdk.WorkflowRunTag
type RunTag sdk.WorkflowRunTag

// NodeHook is a gorp wrapper around sdk.WorkflowNodeHook
type NodeHook sdk.WorkflowNodeHook

// NodeHookModel is a gorp wrapper around sdk.WorkflowHookModel
type NodeHookModel sdk.WorkflowHookModel

func init() {
	gorpmapping.Register(gorpmapping.New(Workflow{}, "workflow", true, "id"))
	gorpmapping.Register(gorpmapping.New(Node{}, "workflow_node", true, "id"))
	gorpmapping.Register(gorpmapping.New(NodeTrigger{}, "workflow_node_trigger", true, "id"))
	gorpmapping.Register(gorpmapping.New(NodeContext{}, "workflow_node_context", true, "id"))
	gorpmapping.Register(gorpmapping.New(sqlContext{}, "workflow_node_context", true, "id"))
	gorpmapping.Register(gorpmapping.New(NodeHook{}, "workflow_node_hook", true, "id"))
	gorpmapping.Register(gorpmapping.New(Join{}, "workflow_node_join", true, "id"))
	gorpmapping.Register(gorpmapping.New(JoinTrigger{}, "workflow_node_join_trigger", true, "id"))
	gorpmapping.Register(gorpmapping.New(Run{}, "workflow_run", true, "id"))
	gorpmapping.Register(gorpmapping.New(NodeRun{}, "workflow_node_run", true, "id"))
	gorpmapping.Register(gorpmapping.New(JobRun{}, "workflow_node_run_job", true, "id"))
	gorpmapping.Register(gorpmapping.New(NodeRunArtifact{}, "workflow_node_run_artifacts", true, "id"))
	gorpmapping.Register(gorpmapping.New(RunTag{}, "workflow_run_tag", false, "workflow_run_id", "tag"))
	gorpmapping.Register(gorpmapping.New(NodeHookModel{}, "workflow_hook_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(Notification{}, "workflow_notification", true, "id"))
}
