package workflow

import (
	"database/sql"
	"time"

	"github.com/lib/pq"

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

// Coverage is a gorp wrapper around sdk.WorkflowNodeRunCoverage
type Coverage sdk.WorkflowNodeRunCoverage

// NodeRun is a gorp wrapper around sdk.WorkflowNodeRun
type NodeRun struct {
	WorkflowID         sql.NullInt64  `db:"workflow_id"`
	WorkflowRunID      int64          `db:"workflow_run_id"`
	ApplicationID      sql.NullInt64  `db:"application_id"`
	ID                 int64          `db:"id"`
	WorkflowNodeID     int64          `db:"workflow_node_id"`
	WorkflowNodeName   string         `db:"workflow_node_name"`
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
	VCSServer          sql.NullString `db:"vcs_server"`
}

// JobRun is a gorp wrapper around sdk.WorkflowNodeJobRun
type JobRun struct {
	ProjectID              int64          `db:"project_id"`
	ID                     int64          `db:"id"`
	WorkflowNodeRunID      int64          `db:"workflow_node_run_id"`
	Job                    sql.NullString `db:"job"`
	Parameters             sql.NullString `db:"variables"`
	Status                 string         `db:"status"`
	Retry                  int            `db:"retry"`
	SpawnAttempts          *pq.Int64Array `db:"spawn_attempts"`
	Queued                 time.Time      `db:"queued"`
	Start                  time.Time      `db:"start"`
	Done                   time.Time      `db:"done"`
	Model                  string         `db:"model"`
	ExecGroups             sql.NullString `db:"exec_groups"`
	PlatformPluginBinaries sql.NullString `db:"platform_plugin_binaries"`
	BookedBy               sdk.Hatchery   `db:"-"`
}

// ToJobRun transform the JobRun with data of the provided sdk.WorkflowNodeJobRun
func (j *JobRun) ToJobRun(jr *sdk.WorkflowNodeJobRun) (err error) {
	j.ProjectID = jr.ProjectID
	j.ID = jr.ID
	j.WorkflowNodeRunID = jr.WorkflowNodeRunID
	j.Job, err = gorpmapping.JSONToNullString(jr.Job)
	if err != nil {
		return sdk.WrapError(err, "column job")
	}
	j.Parameters, err = gorpmapping.JSONToNullString(jr.Parameters)
	if err != nil {
		return sdk.WrapError(err, "column variables")
	}
	j.Status = jr.Status
	j.Retry = jr.Retry
	array := pq.Int64Array(jr.SpawnAttempts)
	j.SpawnAttempts = &array
	j.Queued = jr.Queued
	j.Start = jr.Start
	j.Done = jr.Done
	j.Model = jr.Model
	j.ExecGroups, err = gorpmapping.JSONToNullString(jr.ExecGroups)
	if err != nil {
		return sdk.WrapError(err, "column exec_groups")
	}
	j.PlatformPluginBinaries, err = gorpmapping.JSONToNullString(jr.PlatformPluginBinaries)
	if err != nil {
		return sdk.WrapError(err, "column platform_plugin_binaries")
	}
	return nil
}

// WorkflowNodeRunJob returns a sdk.WorkflowNodeRunJob
func (j JobRun) WorkflowNodeRunJob() (sdk.WorkflowNodeJobRun, error) {
	jr := sdk.WorkflowNodeJobRun{
		ProjectID:         j.ProjectID,
		ID:                j.ID,
		WorkflowNodeRunID: j.WorkflowNodeRunID,
		Status:            j.Status,
		Retry:             j.Retry,
		Queued:            j.Queued,
		QueuedSeconds:     time.Now().Unix() - j.Queued.Unix(),
		Start:             j.Start,
		Done:              j.Done,
	}
	if j.SpawnAttempts != nil {
		jr.SpawnAttempts = *j.SpawnAttempts
	}
	if err := gorpmapping.JSONNullString(j.Job, &jr.Job); err != nil {
		return jr, sdk.WrapError(err, "column job")
	}
	if err := gorpmapping.JSONNullString(j.Parameters, &jr.Parameters); err != nil {
		return jr, sdk.WrapError(err, "column variables")
	}
	if err := gorpmapping.JSONNullString(j.ExecGroups, &jr.ExecGroups); err != nil {
		return jr, sdk.WrapError(err, "column exec_groups")
	}
	if err := gorpmapping.JSONNullString(j.PlatformPluginBinaries, &jr.PlatformPluginBinaries); err != nil {
		return jr, sdk.WrapError(err, "platform_plugin_binaries")
	}
	if defaultOS != "" && defaultArch != "" {
		var modelFound, osArchFound bool
		for _, req := range jr.Job.Action.Requirements {
			if req.Type == sdk.ModelRequirement {
				modelFound = true
			}
			if req.Type == sdk.OSArchRequirement {
				osArchFound = true
			}
		}

		if !modelFound && !osArchFound {
			jr.Job.Action.Requirements = append(jr.Job.Action.Requirements, sdk.Requirement{
				Name:  defaultOS + "/" + defaultArch,
				Type:  sdk.OSArchRequirement,
				Value: defaultOS + "/" + defaultArch,
			})
		}
	}
	return jr, nil
}

// NodeRunArtifact is a gorp wrapper around sdk.WorkflowNodeRunArtifact
type NodeRunArtifact sdk.WorkflowNodeRunArtifact

// RunTag is a gorp wrapper around sdk.WorkflowRunTag
type RunTag sdk.WorkflowRunTag

// NodeHook is a gorp wrapper around sdk.WorkflowNodeHook
type NodeHook sdk.WorkflowNodeHook

// NodeHookModel is a gorp wrapper around sdk.WorkflowHookModel
type NodeHookModel sdk.WorkflowHookModel

type auditWorkflow sdk.AuditWorklflow

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
	gorpmapping.Register(gorpmapping.New(auditWorkflow{}, "workflow_audit", true, "id"))
	gorpmapping.Register(gorpmapping.New(Coverage{}, "workflow_node_run_coverage", false, "workflow_id", "workflow_run_id", "workflow_node_run_id", "repository", "branch"))
}
