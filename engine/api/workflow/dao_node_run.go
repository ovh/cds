package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const nodeRunFields string = `
workflow_node_run.application_id,
workflow_node_run.workflow_id,
workflow_node_run.workflow_run_id,
workflow_node_run.id,
workflow_node_run.workflow_node_id,
workflow_node_run.num,
workflow_node_run.sub_num,
workflow_node_run.status,
workflow_node_run.start,
workflow_node_run.last_modified,
workflow_node_run.done,
workflow_node_run.hook_event,
workflow_node_run.manual,
workflow_node_run.source_node_runs,
workflow_node_run.payload,
workflow_node_run.pipeline_parameters,
workflow_node_run.build_parameters,
workflow_node_run.commits,
workflow_node_run.stages,
workflow_node_run.triggers_run,
workflow_node_run.vcs_repository,
workflow_node_run.vcs_hash,
workflow_node_run.vcs_branch,
workflow_node_run.vcs_tag,
workflow_node_run.vcs_server,
workflow_node_run.workflow_node_name,
workflow_node_run.header,
workflow_node_run.uuid,
workflow_node_run.outgoinghook,
workflow_node_run.hook_execution_timestamp,
workflow_node_run.execution_id,
workflow_node_run.callback,
workflow_node_run.contexts
`

const nodeRunTestsField string = ", workflow_node_run.tests"
const withLightNodeRunTestsField string = ", json_build_object('ko', workflow_node_run.tests->'ko', 'ok', workflow_node_run.tests->'ok', 'skipped', workflow_node_run.tests->'skipped', 'total', workflow_node_run.tests->'total') AS tests"

// LoadNodeRun load a specific node run on a workflow
func LoadNodeRun(db gorp.SqlExecutor, projectkey, workflowname string, noderunID int64, loadOpts LoadRunOptions) (*sdk.WorkflowNodeRun, error) {
	var rr = NodeRun{}
	var testsField string
	if loadOpts.WithTests {
		testsField = nodeRunTestsField
	} else if loadOpts.WithLightTests {
		testsField = withLightNodeRunTestsField
	}

	query := fmt.Sprintf(`select %s %s
	from workflow_node_run
	join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
	join project on project.id = workflow_run.project_id
	join workflow on workflow.id = workflow_run.workflow_id
	where project.projectkey = $1
	and workflow.name = $2
	and workflow_node_run.id = $3`, nodeRunFields, testsField)

	if err := db.SelectOne(&rr, query, projectkey, workflowname, noderunID); err != nil {
		return nil, sdk.WrapError(err, "Unable to load workflow_node_run proj=%s, workflow=%s, noderun_id=%d", projectkey, workflowname, noderunID)
	}

	r, err := fromDBNodeRun(rr, loadOpts)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if loadOpts.WithVulnerabilities {
		vuln, err := loadVulnerabilityReport(db, r.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, sdk.WrapError(err, "vulnerability report for run %d", r.ID)
		}
		if vuln != nil {
			r.VulnerabilitiesReport = *vuln
		}
	}
	return r, nil
}

// LoadNodeRunByNodeJobID load a specific node run on a workflow from a node job run id
func LoadNodeRunByNodeJobID(db gorp.SqlExecutor, nodeJobRunID int64, loadOpts LoadRunOptions) (*sdk.WorkflowNodeRun, error) {
	var rr = NodeRun{}
	var testsField string
	if loadOpts.WithTests {
		testsField = nodeRunTestsField
	} else if loadOpts.WithLightTests {
		testsField = withLightNodeRunTestsField
	}

	query := fmt.Sprintf(`select %s %s
	from workflow_node_run
	join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
  join workflow_node_run_job on workflow_node_run_job.workflow_node_run_id = workflow_node_run.id
	join project on project.id = workflow_run.project_id
	join workflow on workflow.id = workflow_run.workflow_id
	where workflow_node_run_job.id = $1`, nodeRunFields, testsField)

	if err := db.SelectOne(&rr, query, nodeJobRunID); err != nil {
		return nil, sdk.WrapError(err, "Unable to load workflow_node_run node_job_id=%d", nodeJobRunID)
	}

	r, err := fromDBNodeRun(rr, loadOpts)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	return r, nil
}

// LoadAndLockNodeRunByID load and lock a specific node run on a workflow
func LoadAndLockNodeRunByID(ctx context.Context, db gorp.SqlExecutor, id int64) (*sdk.WorkflowNodeRun, error) {
	var end func()
	_, end = telemetry.Span(ctx, "workflow.LoadAndLockNodeRunByID")
	defer end()

	var rr = NodeRun{}

	query := fmt.Sprintf(`select %s %s
	from workflow_node_run
	where workflow_node_run.id = $1 for update SKIP LOCKED`, nodeRunFields, nodeRunTestsField)
	if err := db.SelectOne(&rr, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrLocked)
		}
		return nil, sdk.WrapError(err, "unable to load workflow_node_run node=%d", id)
	}
	return fromDBNodeRun(rr, LoadRunOptions{})
}

// LoadAndLockNodeRunByJobID load and lock a specific node run on a workflow
func LoadAndLockNodeRunByJobID(ctx context.Context, db gorp.SqlExecutor, jobID int64) (*sdk.WorkflowNodeRun, error) {
	_, end := telemetry.Span(ctx, "workflow.LoadAndLockNodeRunByJobID")
	defer end()

	var rr = NodeRun{}

	query := fmt.Sprintf(`
    SELECT %s %s
    FROM workflow_node_run
    JOIN workflow_node_run_job on workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
    WHERE workflow_node_run_job.id = $1 FOR UPDATE SKIP LOCKED
  `, nodeRunFields, nodeRunTestsField)
	if err := db.SelectOne(&rr, query, jobID); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrLocked)
		}
		return nil, sdk.WrapError(err, "unable to load workflow_node_run with jobID=%d", jobID)
	}
	return fromDBNodeRun(rr, LoadRunOptions{})
}

// LoadNodeRunByID load a specific node run on a workflow
func LoadNodeRunByID(ctx context.Context, db gorp.SqlExecutor, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowNodeRun, error) {
	_, end := telemetry.Span(ctx, "workflow.LoadNodeRunByID")
	defer end()

	var rr = NodeRun{}
	var testsField string
	if loadOpts.WithTests {
		testsField = nodeRunTestsField
	} else if loadOpts.WithLightTests {
		testsField = withLightNodeRunTestsField
	}

	query := fmt.Sprintf(`
    SELECT %s %s
	  FROM workflow_node_run
    WHERE workflow_node_run.id = $1
  `, nodeRunFields, testsField)
	if err := db.SelectOne(&rr, query, id); err != nil {
		return nil, sdk.WrapError(err, "unable to load workflow_node_run with id %d", id)
	}

	r, err := fromDBNodeRun(rr, loadOpts)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	return r, nil
}

// insertWorkflowNodeRun insert in table workflow_node_run
func insertWorkflowNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	nodeRunDB, err := makeDBNodeRun(*n)
	if err != nil {
		return err
	}
	if err := db.Insert(nodeRunDB); err != nil {
		return err
	}
	n.ID = nodeRunDB.ID
	return nil
}

func nodeRunExist(db gorp.SqlExecutor, workflowRunID, nodeID, num int64, subnumber int) (bool, error) {
	nb, err := db.SelectInt("SELECT COUNT(1) FROM workflow_node_run WHERE workflow_run_id = $4 AND workflow_node_id = $1 AND num = $2 AND sub_num = $3", nodeID, num, subnumber, workflowRunID)
	return nb > 0, err
}

func fromDBNodeRun(rr NodeRun, opts LoadRunOptions) (*sdk.WorkflowNodeRun, error) {
	r := new(sdk.WorkflowNodeRun)
	if rr.WorkflowID.Valid {
		r.WorkflowID = rr.WorkflowID.Int64
	} else {
		r.WorkflowID = 0
	}
	if rr.ApplicationID.Valid {
		r.ApplicationID = rr.ApplicationID.Int64
	} else {
		r.ApplicationID = 0
	}
	r.WorkflowRunID = rr.WorkflowRunID
	r.ID = rr.ID
	r.WorkflowNodeID = rr.WorkflowNodeID
	r.WorkflowNodeName = rr.WorkflowNodeName
	r.Number = rr.Number
	r.SubNumber = rr.SubNumber
	r.Status = rr.Status
	r.Start = rr.Start
	r.Done = rr.Done
	r.LastModified = rr.LastModified
	r.Contexts = rr.Contexts

	if rr.VCSHash.Valid {
		r.VCSHash = rr.VCSHash.String
	}
	if rr.VCSRepository.Valid {
		r.VCSRepository = rr.VCSRepository.String
	}
	if rr.VCSBranch.Valid {
		r.VCSBranch = rr.VCSBranch.String
	}
	if rr.VCSTag.Valid {
		r.VCSTag = rr.VCSTag.String
	}
	if rr.VCSServer.Valid {
		r.VCSServer = rr.VCSServer.String
	}

	if err := gorpmapping.JSONNullString(rr.TriggersRun, &r.TriggersRun); err != nil {
		return nil, sdk.WrapError(err, "Error loading node run trigger %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Stages, &r.Stages); err != nil {
		return nil, sdk.WrapError(err, "Error loading node run %d", r.ID)
	}
	for i := range r.Stages {
		s := &r.Stages[i]
		for j := range s.RunJobs {
			rj := &s.RunJobs[j]
			if rj.Status == sdk.StatusWaiting {
				rj.QueuedSeconds = time.Now().Unix() - rj.Queued.Unix()
			}
		}
	}

	if err := gorpmapping.JSONNullString(rr.Payload, &r.Payload); err != nil {
		return nil, sdk.WrapError(err, "Error loading node run %d: Payload", r.ID)
	}

	if rr.HookEvent.Valid {
		r.HookEvent = new(sdk.WorkflowNodeRunHookEvent)
		if err := gorpmapping.JSONNullString(rr.HookEvent, r.HookEvent); err != nil {
			return nil, sdk.WrapError(err, "Error loading node run %d: HookEvent", r.ID)
		}
	}

	if rr.HookEvent.Valid {
		r.HookEvent = new(sdk.WorkflowNodeRunHookEvent)
		if err := gorpmapping.JSONNullString(rr.HookEvent, r.HookEvent); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: HookEvent", r.ID)
		}
	}

	if rr.PipelineParameters.Valid {
		if err := gorpmapping.JSONNullString(rr.PipelineParameters, &r.PipelineParameters); err != nil {
			return nil, sdk.WrapError(err, "Error loading node run %d: PipelineParameters", r.ID)
		}
	}

	if !opts.DisableDetailledNodeRun {
		if err := gorpmapping.JSONNullString(rr.SourceNodeRuns, &r.SourceNodeRuns); err != nil {
			return nil, sdk.WrapError(err, "Error loading node run %d : SourceNodeRuns", r.ID)
		}
		if err := gorpmapping.JSONNullString(rr.Commits, &r.Commits); err != nil {
			return nil, sdk.WrapError(err, "Error loading node run %d: Commits", r.ID)
		}
		if rr.Manual.Valid {
			r.Manual = new(sdk.WorkflowNodeRunManual)
			if err := gorpmapping.JSONNullString(rr.Manual, r.Manual); err != nil {
				return nil, sdk.WrapError(err, "Error loading node run %d: Manual", r.ID)
			}
		}
		if err := gorpmapping.JSONNullString(rr.BuildParameters, &r.BuildParameters); err != nil {
			return nil, sdk.WrapError(err, "Error loading node run %d: BuildParameters", r.ID)
		}
	}

	if rr.Header.Valid {
		if err := gorpmapping.JSONNullString(rr.Header, &r.Header); err != nil {
			return nil, sdk.WrapError(err, "Error loading node run %d: Header", r.ID)
		}
	}

	if rr.Tests.Valid {
		r.Tests = new(sdk.TestsResults)
		if err := gorpmapping.JSONNullString(rr.Tests, r.Tests); err != nil {
			return nil, sdk.WrapError(err, "Error loading node run %d: Tests", r.ID)
		}
	}

	if rr.UUID.Valid {
		r.UUID = rr.UUID.String
	}

	if rr.ExecutionID.Valid {
		r.HookExecutionID = rr.ExecutionID.String
	}

	if rr.HookExecutionTimestamp.Valid {
		r.HookExecutionTimeStamp = rr.HookExecutionTimestamp.Int64
	}

	if rr.OutgoingHook.Valid {
		if err := gorpmapping.JSONNullString(rr.OutgoingHook, &r.OutgoingHook); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: OutgoingHook", r.ID)
		}
	}

	if rr.Callback.Valid {
		if err := gorpmapping.JSONNullString(rr.Callback, &r.Callback); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: Callback", r.ID)
		}
	}

	return r, nil
}

func makeDBNodeRun(n sdk.WorkflowNodeRun) (*NodeRun, error) {
	nodeRunDB := new(NodeRun)
	nodeRunDB.ID = n.ID
	nodeRunDB.WorkflowID.Valid = true
	nodeRunDB.WorkflowID.Int64 = n.WorkflowID
	nodeRunDB.ApplicationID.Int64 = n.ApplicationID
	nodeRunDB.ApplicationID.Valid = true
	nodeRunDB.WorkflowRunID = n.WorkflowRunID
	nodeRunDB.WorkflowNodeID = n.WorkflowNodeID
	nodeRunDB.WorkflowNodeName = n.WorkflowNodeName
	nodeRunDB.Number = n.Number
	nodeRunDB.SubNumber = n.SubNumber
	nodeRunDB.Status = n.Status
	nodeRunDB.Start = n.Start
	nodeRunDB.Done = n.Done
	nodeRunDB.LastModified = n.LastModified

	nodeRunDB.VCSServer.Valid = true
	nodeRunDB.VCSServer.String = n.VCSServer
	nodeRunDB.VCSHash.Valid = true
	nodeRunDB.VCSHash.String = n.VCSHash
	nodeRunDB.VCSBranch.Valid = true
	nodeRunDB.VCSBranch.String = n.VCSBranch
	nodeRunDB.VCSTag.Valid = true
	nodeRunDB.VCSTag.String = n.VCSTag
	nodeRunDB.VCSRepository.Valid = true
	nodeRunDB.VCSRepository.String = n.VCSRepository
	nodeRunDB.ExecutionID.Valid = true
	nodeRunDB.ExecutionID.String = n.HookExecutionID
	nodeRunDB.HookExecutionTimestamp.Valid = true
	nodeRunDB.HookExecutionTimestamp.Int64 = n.HookExecutionTimeStamp
	nodeRunDB.UUID.Valid = true
	nodeRunDB.UUID.String = n.UUID
	nodeRunDB.Contexts = n.Contexts

	if n.TriggersRun != nil {
		s, err := gorpmapping.JSONToNullString(n.TriggersRun)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from TriggerRun")
		}
		nodeRunDB.TriggersRun = s
	}
	if n.Stages != nil {
		s, err := gorpmapping.JSONToNullString(n.Stages)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from Stages")
		}
		nodeRunDB.Stages = s
	}
	if n.SourceNodeRuns != nil {
		s, err := gorpmapping.JSONToNullString(n.SourceNodeRuns)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from SourceNodeRuns")
		}
		nodeRunDB.SourceNodeRuns = s
	}
	if n.HookEvent != nil {
		s, err := gorpmapping.JSONToNullString(n.HookEvent)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from hook_event")
		}
		nodeRunDB.HookEvent = s
	}
	if n.Manual != nil {
		s, err := gorpmapping.JSONToNullString(n.Manual)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from manual")
		}
		nodeRunDB.Manual = s
	}
	if n.Payload != nil {
		s, err := gorpmapping.JSONToNullString(n.Payload)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from payload")
		}
		nodeRunDB.Payload = s
	}
	if n.PipelineParameters != nil {
		s, err := gorpmapping.JSONToNullString(n.PipelineParameters)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from pipeline_parameters")
		}
		nodeRunDB.PipelineParameters = s
	}
	if n.BuildParameters != nil {
		s, err := gorpmapping.JSONToNullString(n.BuildParameters)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from build_parameters")
		}
		nodeRunDB.BuildParameters = s
	}
	if n.Tests != nil {
		s, err := gorpmapping.JSONToNullString(n.Tests)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from tests")
		}
		nodeRunDB.Tests = s
	}
	if n.Commits != nil {
		s, err := gorpmapping.JSONToNullString(n.Commits)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to get json from commits")
		}
		nodeRunDB.Commits = s
	}
	sh, err := gorpmapping.JSONToNullString(n.Header)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get json from header")
	}
	nodeRunDB.Header = sh

	cb, err := gorpmapping.JSONToNullString(n.Callback)
	if err != nil {
		return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from callback")
	}
	nodeRunDB.Callback = cb

	oh, err := gorpmapping.JSONToNullString(n.OutgoingHook)
	if err != nil {
		return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from outgoing hook")
	}
	nodeRunDB.OutgoingHook = oh

	return nodeRunDB, nil
}

// UpdateNodeRunBuildParameters updates build_parameters in table workflow_node_run
func UpdateNodeRunBuildParameters(db gorp.SqlExecutor, nodeID int64, buildParameters []sdk.Parameter) error {
	if buildParameters == nil {
		return nil
	}

	bts, err := json.Marshal(&buildParameters)
	if err != nil {
		return sdk.WrapError(err, "unable to get json from build_parameters")
	}

	_, errU := db.Exec("UPDATE workflow_node_run SET build_parameters = $1 WHERE id = $2", bts, nodeID)

	return sdk.WrapError(errU, "UpdateNodeRunBuildParameters>")
}

// UpdateNodeRun updates in table workflow_node_run
func UpdateNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	log.Debug(context.TODO(), "workflow.UpdateNodeRun> node.id=%d, status=%s", n.ID, n.Status)
	nodeRunDB, err := makeDBNodeRun(*n)
	if err != nil {
		return err
	}
	if _, err := db.Update(nodeRunDB); err != nil {
		return err
	}
	return nil
}

// GetNodeRunBuildCommits gets commits for given node run and return current vcs info
func GetNodeRunBuildCommits(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wf sdk.Workflow, wNodeName string, number int64, nodeRun *sdk.WorkflowNodeRun, app *sdk.Application, env *sdk.Environment) ([]sdk.VCSCommit, sdk.BuildNumberAndHash, error) {
	ctx, end := telemetry.Span(ctx, "workflow.GetNodeRunBuildCommits")
	defer end()

	var cur sdk.BuildNumberAndHash
	if app == nil {
		log.Debug(ctx, "GetNodeRunBuildCommits> No app linked")
		return nil, cur, nil
	}

	if app.VCSServer == "" {
		log.Debug(ctx, "GetNodeRunBuildCommits> No repository linked")
		return nil, cur, nil
	}
	cur.BuildNumber = number

	res := []sdk.VCSCommit{}
	//Get the RepositoriesManager Client
	client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, app.VCSServer)
	if err != nil {
		return nil, cur, sdk.WrapError(err, "cannot get client")
	}
	cur.Remote = nodeRun.VCSRepository
	cur.Branch = nodeRun.VCSBranch
	cur.Hash = nodeRun.VCSHash
	cur.Tag = nodeRun.VCSTag

	if cur.Remote == "" {
		cur.Remote = app.RepositoryFullname
	}

	if cur.Branch == "" && cur.Tag == "" {
		branch, err := repositoriesmanager.DefaultBranch(ctx, client, cur.Remote)
		if err != nil {
			return nil, cur, sdk.WrapError(err, "cannot load branches from vcs api remote %s", cur.Remote)
		}
		cur.Branch = branch.DisplayID
	}

	var envID int64
	if env != nil {
		envID = env.ID
	}

	repo := app.RepositoryFullname
	if cur.Remote != "" {
		repo = cur.Remote
	}

	var lastCommit sdk.VCSCommit
	if cur.Hash == "" && cur.Tag == "" && cur.Branch != "" {
		//If we only have the current branch, search for the branch
		br, err := client.Branch(ctx, repo, sdk.VCSBranchFilters{BranchName: cur.Branch})
		if err != nil {
			return nil, cur, sdk.WrapError(err, "cannot get branch %s", cur.Branch)
		}
		if br != nil {
			if br.LatestCommit == "" {
				return nil, cur, sdk.WrapError(sdk.ErrNoBranch, "branch %s latest commit not found", cur.Branch)
			}

			//and return the last commit of the branch
			cm, err := client.Commit(ctx, repo, br.LatestCommit)
			if err != nil {
				return nil, cur, sdk.WrapError(err, "cannot get commits with cur.Hash %s", cur.Hash)
			}
			lastCommit = cm
			cur.Hash = cm.Hash
		}
	}

	//Get the commit hash for the node run number and the hash for the previous node run for the same branch and same remote
	prev, err := PreviousNodeRunVCSInfos(ctx, db, proj.Key, wf, wNodeName, cur, app.ID, envID)
	if err != nil {
		return nil, cur, sdk.WrapError(err, "cannot get build number and hashes (buildNumber=%d, nodeName=%s, applicationID=%d)", number, wNodeName, app.ID)
	}

	if prev.Hash == "" {
		log.Warn(ctx, "GetNodeRunBuildCommits> No previous build was found for branch %s", cur.Branch)
	}

	if prev.Hash != "" && cur.Hash == prev.Hash {
		log.Debug(ctx, "GetNodeRunBuildCommits> there is not difference between the previous build and the current build for node %s", nodeRun.WorkflowNodeName)
	} else if prev.Hash != "" {
		if cur.Tag == "" {
			if cur.Hash == "" {
				br, err := client.Branch(ctx, repo, sdk.VCSBranchFilters{BranchName: cur.Branch})
				if err != nil {
					return nil, cur, sdk.WrapError(err, "Cannot get branch %s", cur.Branch)
				}
				cur.Hash = br.LatestCommit
			}
			//If we are lucky, return a true diff
			commits, err := client.Commits(ctx, repo, cur.Branch, prev.Hash, cur.Hash)
			if err != nil {
				return nil, cur, sdk.WrapError(err, "Cannot get commits")
			}
			if commits != nil {
				res = commits
			}
		}

		if cur.Hash == "" && cur.Tag != "" {
			c, err := client.CommitsBetweenRefs(ctx, repo, prev.Hash, cur.Tag)
			if err != nil {
				return nil, cur, sdk.WrapError(err, "Cannot get commits")
			}
			if c != nil {
				res = c
			}
		}
	} else if prev.Hash == "" {
		if lastCommit.Hash != "" {
			res = []sdk.VCSCommit{lastCommit}
		} else if cur.Tag != "" {
			base := prev.Tag
			if base == "" {
				base = prev.Hash
			}
			c, err := client.CommitsBetweenRefs(ctx, repo, base, cur.Tag)
			if err != nil {
				return nil, cur, sdk.WrapError(err, "Cannot get commits")
			}
			if c != nil {
				res = c
			}
		}
	} else {
		//If we only get current node run hash
		log.Debug(ctx, "GetNodeRunBuildCommits>  Looking for every commit until %s ", cur.Hash)
		c, err := client.Commits(ctx, repo, cur.Branch, "", cur.Hash)
		if err != nil {
			return nil, cur, sdk.WrapError(err, "Cannot get commits")
		}
		if c != nil {
			res = c
		}
	}

	return res, cur, nil
}

// PreviousNodeRun find previous node run
func PreviousNodeRun(db gorp.SqlExecutor, nr sdk.WorkflowNodeRun, nodeName string, workflowID int64) (*sdk.WorkflowNodeRun, error) {
	// check the first run of a workflow, no need to check previous
	if nr.Number == 1 && nr.SubNumber == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
					SELECT %s FROM workflow_node_run
					JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id AND workflow_run.workflow_id = $1
					WHERE workflow_node_run.workflow_node_name = $2
						AND workflow_node_run.vcs_branch = $3 AND workflow_node_run.vcs_tag = $4
						AND workflow_node_run.num <= $5
						AND workflow_node_run.id != $6
					ORDER BY workflow_node_run.num DESC, workflow_node_run.sub_num DESC
					LIMIT 1
				`, nodeRunFields)

	var rr = NodeRun{}
	if err := db.SelectOne(&rr, query, workflowID, nodeName, nr.VCSBranch, nr.VCSTag, nr.Number, nr.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot load previous run on workflow %d node %s nr.VCSBranch:%s nr.VCSTag:%s nr.Number:%d nr.ID:%d ", workflowID, nodeName, nr.VCSBranch, nr.VCSTag, nr.Number, nr.ID)
	}
	pNodeRun, errF := fromDBNodeRun(rr, LoadRunOptions{})
	if errF != nil {
		return nil, sdk.WrapError(errF, "PreviousNodeRun> Cannot read node run")
	}
	return pNodeRun, nil
}

// PreviousNodeRunVCSInfos returns a struct with BuildNumber, Commit Hash, Branch, Remote, Remote_url
// for the current node run and the previous one on the same branch.
// Returned value may be zero if node run are not found
// If you don't have environment linked set envID to 0 or -1
func PreviousNodeRunVCSInfos(ctx context.Context, db gorp.SqlExecutor, projectKey string, wf sdk.Workflow, nodeName string, current sdk.BuildNumberAndHash, appID int64, envID int64) (sdk.BuildNumberAndHash, error) {
	var previous sdk.BuildNumberAndHash
	var prevHash, prevBranch, prevTag, prevRepository sql.NullString
	var previousBuildNumber sql.NullInt64

	queryPrevious := `
		SELECT workflow_node_run.vcs_branch, workflow_node_run.vcs_tag, workflow_node_run.vcs_hash, workflow_node_run.vcs_repository, workflow_node_run.num
		FROM workflow_node_run
		JOIN w_node ON w_node.name = workflow_node_run.workflow_node_name AND w_node.name = $1 AND w_node.workflow_id = $2
		JOIN w_node_context ON w_node_context.node_id = w_node.id
		WHERE workflow_node_run.vcs_hash IS NOT NULL
		AND workflow_node_run.workflow_id = $2
		AND workflow_node_run.num < $3
		AND w_node_context.application_id = $4
	`

	argPrevious := []interface{}{nodeName, wf.ID, current.BuildNumber, appID}
	if envID > 0 {
		argPrevious = append(argPrevious, envID)
		queryPrevious += "AND w_node_context.environment_id = $5"
	}
	queryPrevious += " ORDER BY workflow_node_run.num DESC LIMIT 1"

	errPrev := db.QueryRow(queryPrevious, argPrevious...).Scan(&prevBranch, &prevTag, &prevHash, &prevRepository, &previousBuildNumber)
	if errPrev == sql.ErrNoRows {
		log.Warn(ctx, "PreviousNodeRunVCSInfos> no result with previous %d %s , arguments %v", current.BuildNumber, nodeName, argPrevious)
		return previous, nil
	}
	if errPrev != nil {
		return previous, errPrev
	}

	if prevBranch.Valid {
		previous.Branch = prevBranch.String
	}
	if prevTag.Valid {
		previous.Tag = prevTag.String
	}
	if prevHash.Valid {
		previous.Hash = prevHash.String
	}
	if prevRepository.Valid {
		previous.Remote = prevRepository.String
	}
	if previousBuildNumber.Valid {
		previous.BuildNumber = previousBuildNumber.Int64
	}

	return previous, nil
}

func updateNodeRunCommits(ctx context.Context, db gorp.SqlExecutor, id int64, commits []sdk.VCSCommit) error {
	ctx, end := telemetry.Span(ctx, "workflow.updateNodeRunCommits")
	defer end()

	log.Debug(ctx, "updateNodeRunCommits> Updating %d commits for workflow_node_run #%d", len(commits), id)
	commitsBtes, errMarshal := json.Marshal(commits)
	if errMarshal != nil {
		return sdk.WrapError(errMarshal, "updateNodeRunCommits> Unable to marshal commits")
	}

	if _, err := db.Exec("UPDATE workflow_node_run SET commits = $1 where id = $2", commitsBtes, id); err != nil {
		return sdk.WrapError(err, "Unable to update workflow_node_run id=%d", id)
	}
	return nil
}

// UpdateNodeRunStatusAndStage update just noderun status and stage
func UpdateNodeRunStatusAndStage(db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun) error {
	stagesBts, errMarshal := json.Marshal(nodeRun.Stages)
	if errMarshal != nil {
		return sdk.WrapError(errMarshal, "unable to marshal stages")
	}

	if _, err := db.Exec("UPDATE workflow_node_run SET status = $1, stages = $2, done = $3 where id = $4", nodeRun.Status, stagesBts, nodeRun.Done, nodeRun.ID); err != nil {
		return sdk.WrapError(err, "unable to update workflow_node_run %s", nodeRun.WorkflowNodeName)
	}
	return nil
}

// RunExist Check if run exist or not
func RunExist(db gorp.SqlExecutor, projectKey string, workflowID int64, hash string) (bool, error) {
	query := `
	SELECT COUNT(1)
		FROM workflow_node_run
			JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id
			JOIN project ON project.id = workflow_run.project_id
	WHERE project.projectkey = $1
	AND workflow_run.workflow_id = $2
	AND workflow_node_run.vcs_hash = $3
	`

	count, err := db.SelectInt(query, projectKey, workflowID, hash)
	return count != 0, err
}

func LoadNodeRunDistinctExecutionIDs(db gorp.SqlExecutor) ([]string, error) {
	query := `
		SELECT DISTINCT execution_id
    FROM workflow_node_run
    WHERE execution_id <> '' AND execution_id IS NOT NULL;
	`
	var executionIDs []string
	if _, err := db.Select(&executionIDs, query); err != nil {
		return nil, sdk.WithStack(err)
	}
	return executionIDs, nil
}
