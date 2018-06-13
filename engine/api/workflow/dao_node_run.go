package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/venom"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/tracing"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const nodeRunFields string = `
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
workflow_node_run.workflow_node_name
`

const nodeRunTestsField string = ", workflow_node_run.tests"
const withLightNodeRunTestsField string = ", json_build_object('ko', workflow_node_run.tests->'ko', 'ok', workflow_node_run.tests->'ok', 'skipped', workflow_node_run.tests->'skipped', 'total', workflow_node_run.tests->'total') AS tests"

//LoadNodeRun load a specific node run on a workflow
func LoadNodeRun(db gorp.SqlExecutor, projectkey, workflowname string, number, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowNodeRun, error) {
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
	and workflow_run.num = $3
	and workflow_node_run.id = $4`, nodeRunFields, testsField)

	if err := db.SelectOne(&rr, query, projectkey, workflowname, number, id); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeRun> Unable to load workflow_node_run proj=%s, workflow=%s, num=%d, node=%d", projectkey, workflowname, number, id)
	}

	r, err := fromDBNodeRun(rr, loadOpts)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNodeRun>")
	}

	if loadOpts.WithArtifacts {
		arts, errA := loadArtifactByNodeRunID(db, r.ID)
		if errA != nil {
			return nil, sdk.WrapError(errA, "LoadNodeRun>Error loading artifacts for run %d", r.ID)
		}
		r.Artifacts = arts
	}

	return r, nil

}

//LoadAndLockNodeRunByID load and lock a specific node run on a workflow
func LoadAndLockNodeRunByID(ctx context.Context, db gorp.SqlExecutor, id int64, wait bool) (*sdk.WorkflowNodeRun, error) {
	var end func()
	_, end = tracing.Span(ctx, "workflow.LoadAndLockNodeRunByID")
	defer end()

	var rr = NodeRun{}

	query := fmt.Sprintf(`select %s %s
	from workflow_node_run
	where workflow_node_run.id = $1 for update`, nodeRunFields, nodeRunTestsField)
	if !wait {
		query += " nowait"
	}
	if err := db.SelectOne(&rr, query, id); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadAndLockNodeRunByID> Unable to load workflow_node_run node=%d", id)
	}
	return fromDBNodeRun(rr, LoadRunOptions{})
}

//LoadNodeRunByID load a specific node run on a workflow
func LoadNodeRunByID(db gorp.SqlExecutor, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowNodeRun, error) {
	var rr = NodeRun{}
	var testsField string
	if loadOpts.WithTests {
		testsField = nodeRunTestsField
	} else if loadOpts.WithLightTests {
		testsField = withLightNodeRunTestsField
	}

	query := fmt.Sprintf(`select %s %s
	from workflow_node_run
	where workflow_node_run.id = $1`, nodeRunFields, testsField)
	if err := db.SelectOne(&rr, query, id); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeRunByID> Unable to load workflow_node_run node=%d", id)
	}

	r, err := fromDBNodeRun(rr, loadOpts)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNodeRun>")
	}

	if loadOpts.WithArtifacts {
		arts, errA := loadArtifactByNodeRunID(db, r.ID)
		if errA != nil {
			return nil, sdk.WrapError(errA, "LoadNodeRun>Error loading artifacts for run %d", r.ID)
		}
		r.Artifacts = arts
	}

	return r, nil

}

//insertWorkflowNodeRun insert in table workflow_node_run
func insertWorkflowNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	nodeRunDB, err := makeDBNodeRun(*n)
	if err != nil {
		return err
	}
	if err := db.Insert(nodeRunDB); err != nil {
		return err
	}
	n.ID = nodeRunDB.ID

	log.Debug("insertWorkflowNodeRun> new node run: %d (%d)", n.ID, n.WorkflowNodeID)
	return nil
}

func nodeRunExist(db gorp.SqlExecutor, nodeID, num int64, subnumber int) (bool, error) {
	nb, err := db.SelectInt("SELECT COUNT(1) FROM workflow_node_run WHERE workflow_node_id = $1 AND num = $2 AND sub_num = $3", nodeID, num, subnumber)
	return nb > 0, err
}

func fromDBNodeRun(rr NodeRun, opts LoadRunOptions) (*sdk.WorkflowNodeRun, error) {
	r := new(sdk.WorkflowNodeRun)
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

	if rr.VCSHash.Valid {
		r.VCSHash = rr.VCSHash.String
	}
	if rr.VCSRepository.Valid {
		r.VCSRepository = rr.VCSRepository.String
	}
	if rr.VCSBranch.Valid {
		r.VCSBranch = rr.VCSBranch.String
	}

	if err := gorpmapping.JSONNullString(rr.TriggersRun, &r.TriggersRun); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run trigger %d", r.ID)
	}
	if err := gorpmapping.JSONNullString(rr.Stages, &r.Stages); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d", r.ID)
	}
	for i := range r.Stages {
		s := &r.Stages[i]
		for j := range s.RunJobs {
			rj := &s.RunJobs[j]
			if rj.Status == sdk.StatusWaiting.String() {
				rj.QueuedSeconds = time.Now().Unix() - rj.Queued.Unix()
			}
		}
	}

	if err := gorpmapping.JSONNullString(rr.Payload, &r.Payload); err != nil {
		return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: Payload", r.ID)
	}

	if !opts.DisableDetailledNodeRun {
		if err := gorpmapping.JSONNullString(rr.SourceNodeRuns, &r.SourceNodeRuns); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d : SourceNodeRuns", r.ID)
		}
		if err := gorpmapping.JSONNullString(rr.Commits, &r.Commits); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: Commits", r.ID)
		}
		if rr.HookEvent.Valid {
			r.HookEvent = new(sdk.WorkflowNodeRunHookEvent)
			if err := gorpmapping.JSONNullString(rr.HookEvent, r.HookEvent); err != nil {
				return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: HookEvent", r.ID)
			}
		}
		if rr.Manual.Valid {
			r.Manual = new(sdk.WorkflowNodeRunManual)
			if err := gorpmapping.JSONNullString(rr.Manual, r.Manual); err != nil {
				return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: Manual", r.ID)
			}
		}
		if err := gorpmapping.JSONNullString(rr.BuildParameters, &r.BuildParameters); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: BuildParameters", r.ID)
		}
		if rr.PipelineParameters.Valid {
			if err := gorpmapping.JSONNullString(rr.PipelineParameters, &r.PipelineParameters); err != nil {
				return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: PipelineParameters", r.ID)
			}
		}
	}

	if rr.Tests.Valid {
		r.Tests = new(venom.Tests)
		if err := gorpmapping.JSONNullString(rr.Tests, r.Tests); err != nil {
			return nil, sdk.WrapError(err, "fromDBNodeRun>Error loading node run %d: Tests", r.ID)
		}
	}

	return r, nil
}

func makeDBNodeRun(n sdk.WorkflowNodeRun) (*NodeRun, error) {
	nodeRunDB := new(NodeRun)
	nodeRunDB.ID = n.ID
	nodeRunDB.WorkflowRunID = n.WorkflowRunID
	nodeRunDB.WorkflowNodeID = n.WorkflowNodeID
	nodeRunDB.WorkflowNodeName = n.WorkflowNodeName
	nodeRunDB.Number = n.Number
	nodeRunDB.SubNumber = n.SubNumber
	nodeRunDB.Status = n.Status
	nodeRunDB.Start = n.Start
	nodeRunDB.Done = n.Done
	nodeRunDB.LastModified = n.LastModified

	nodeRunDB.VCSHash.Valid = true
	nodeRunDB.VCSHash.String = n.VCSHash
	nodeRunDB.VCSBranch.Valid = true
	nodeRunDB.VCSBranch.String = n.VCSBranch
	nodeRunDB.VCSRepository.Valid = true
	nodeRunDB.VCSRepository.String = n.VCSRepository

	if n.TriggersRun != nil {
		s, err := gorpmapping.JSONToNullString(n.TriggersRun)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from TriggerRun")
		}
		nodeRunDB.TriggersRun = s
	}
	if n.Stages != nil {
		s, err := gorpmapping.JSONToNullString(n.Stages)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from Stages")
		}
		nodeRunDB.Stages = s
	}
	if n.SourceNodeRuns != nil {
		s, err := gorpmapping.JSONToNullString(n.SourceNodeRuns)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from SourceNodeRuns")
		}
		nodeRunDB.SourceNodeRuns = s
	}
	if n.HookEvent != nil {
		s, err := gorpmapping.JSONToNullString(n.HookEvent)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from hook_event")
		}
		nodeRunDB.HookEvent = s
	}
	if n.Manual != nil {
		s, err := gorpmapping.JSONToNullString(n.Manual)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from manual")
		}
		nodeRunDB.Manual = s
	}
	if n.Payload != nil {
		s, err := gorpmapping.JSONToNullString(n.Payload)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from payload")
		}
		nodeRunDB.Payload = s
	}
	if n.PipelineParameters != nil {
		s, err := gorpmapping.JSONToNullString(n.PipelineParameters)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from pipeline_parameters")
		}
		nodeRunDB.PipelineParameters = s
	}
	if n.BuildParameters != nil {
		s, err := gorpmapping.JSONToNullString(n.BuildParameters)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from build_parameters")
		}
		nodeRunDB.BuildParameters = s
	}
	if n.Tests != nil {
		s, err := gorpmapping.JSONToNullString(n.Tests)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from tests")
		}
		nodeRunDB.Tests = s
	}
	if n.Commits != nil {
		s, err := gorpmapping.JSONToNullString(n.Commits)
		if err != nil {
			return nil, sdk.WrapError(err, "makeDBNodeRun> unable to get json from commits")
		}
		nodeRunDB.Commits = s
	}

	return nodeRunDB, nil
}

//UpdateNodeRunBuildParameters updates build_parameters in table workflow_node_run
func UpdateNodeRunBuildParameters(db gorp.SqlExecutor, nodeID int64, buildParameters []sdk.Parameter) error {
	if buildParameters == nil {
		return nil
	}

	bts, err := json.Marshal(&buildParameters)
	if err != nil {
		return sdk.WrapError(err, "UpdateNodeRunBuildParameters> unable to get json from build_parameters")
	}

	_, errU := db.Exec("UPDATE workflow_node_run SET build_parameters = $1 WHERE id = $2", bts, nodeID)

	return sdk.WrapError(errU, "UpdateNodeRunBuildParameters>")
}

//UpdateNodeRun updates in table workflow_node_run
func UpdateNodeRun(db gorp.SqlExecutor, n *sdk.WorkflowNodeRun) error {
	log.Debug("workflow.UpdateNodeRun> node.id=%d, status=%s", n.ID, n.Status)
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
func GetNodeRunBuildCommits(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wf *sdk.Workflow, wNodeName string, number int64, nodeRun *sdk.WorkflowNodeRun, app *sdk.Application, env *sdk.Environment) ([]sdk.VCSCommit, sdk.BuildNumberAndHash, error) {
	var cur sdk.BuildNumberAndHash
	if app == nil {
		log.Debug("GetNodeRunBuildCommits> No app linked")
		return nil, cur, nil
	}

	if app.VCSServer == "" {
		log.Debug("GetNodeRunBuildCommits> No repository linked")
		return nil, cur, nil
	}
	cur.BuildNumber = number

	vcsServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
	if vcsServer == nil {
		log.Debug("GetNodeRunBuildCommits> No vcsServer found")
		return nil, cur, nil
	}

	res := []sdk.VCSCommit{}
	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(db, store, vcsServer)
	if errclient != nil {
		return nil, cur, sdk.WrapError(errclient, "GetNodeRunBuildCommits> Cannot get client")
	}

	cur.Remote = nodeRun.VCSRepository
	cur.Branch = nodeRun.VCSBranch
	cur.Hash = nodeRun.VCSHash

	if cur.Remote == "" {
		cur.Remote = app.RepositoryFullname
	}

	if cur.Branch == "" {
		branches, errBr := client.Branches(cur.Remote)
		if errBr != nil {
			return nil, cur, sdk.WrapError(errBr, "GetNodeRunBuildCommits> Cannot load branches from vcs api remote %s", cur.Remote)
		}
		found := false
		for _, br := range branches {
			if br.Default {
				cur.Branch = br.DisplayID
				found = true
				break
			}
		}
		if !found {
			return nil, cur, sdk.WrapError(fmt.Errorf("Cannot find default branch from vcs api"), "GetNodeRunBuildCommits>")
		}
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
	if cur.Hash == "" {
		//If we only have the current branch, search for the branch
		br, err := client.Branch(repo, cur.Branch)
		if err != nil {
			return nil, cur, sdk.WrapError(err, "GetNodeRunBuildCommits> Cannot get branch %s", cur.Branch)
		}
		if br != nil {
			if br.LatestCommit == "" {
				return nil, cur, sdk.WrapError(sdk.ErrNoBranch, "GetNodeRunBuildCommits> Branch or lastest commit not found")
			}

			//and return the last commit of the branch
			log.Debug("get the last commit : %s", br.LatestCommit)
			cm, errcm := client.Commit(repo, br.LatestCommit)
			if errcm != nil {
				// not found is not an error, it's probably that the br.LatestCommit commit is not found on repo (git push)
				if !sdk.ErrorIs(errcm, sdk.ErrNotFound) {
					return nil, cur, sdk.WrapError(errcm, "GetNodeRunBuildCommits> Cannot get commits")
				}
			}
			lastCommit = cm
			cur.Hash = cm.Hash
		}
	}

	log.Debug("GetNodeRunBuildCommits> Before PreviousNodeRunVCSInfos %s current: %+v appId: %d envId: %d", wNodeName, cur, app.ID, envID)
	//Get the commit hash for the node run number and the hash for the previous node run for the same branch and same remote
	prev, errcurr := PreviousNodeRunVCSInfos(db, p.Key, wf, wNodeName, cur, app.ID, envID)
	if errcurr != nil {
		return nil, cur, sdk.WrapError(errcurr, "GetNodeRunBuildCommits> Cannot get build number and hashes (buildNumber=%d, nodeName=%s, applicationID=%d)", number, wNodeName, app.ID)
	}

	if prev.Hash == "" {
		log.Warning("GetNodeRunBuildCommits> No previous build was found for branch %s", cur.Branch)
	} else {
		log.Info("GetNodeRunBuildCommits> Current Build number: %d - Current Hash: %s - Previous Build number: %d - Previous Hash: %s", cur.BuildNumber, cur.Hash, prev.BuildNumber, prev.Hash)
	}

	//TODO: to delete after debug commits
	log.Debug("GetNodeRunBuildCommits> current : %+v , previous: %+v", cur, prev)
	if prev.Hash != "" && cur.Hash == prev.Hash {
		log.Info("GetNodeRunBuildCommits> there is not difference between the previous build and the current build for node %s", nodeRun.WorkflowNodeName)
	} else if prev.Hash != "" {
		log.Debug("GetNodeRunBuildCommits> Previous hash is NOT empty, node %s and current hash %v ", nodeRun.WorkflowNodeName, cur)
		if cur.Hash == "" {
			br, err := client.Branch(repo, cur.Branch)
			if err != nil {
				return nil, cur, sdk.WrapError(err, "GetNodeRunBuildCommits> Cannot get branch %s", cur.Branch)
			}
			cur.Hash = br.LatestCommit
		}
		//If we are lucky, return a true diff
		commits, err := client.Commits(repo, cur.Branch, prev.Hash, cur.Hash)
		if err != nil {
			// prev.Hash could be empty -> 404 returned from vcs, it's not an error
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, cur, sdk.WrapError(err, "GetNodeRunBuildCommits> Cannot get commits")
			}
		}
		res = commits
	} else if prev.Hash == "" {
		log.Debug("GetNodeRunBuildCommits> Previous hash is empty, return just the last commit for node %s ", nodeRun.WorkflowNodeName)
		if lastCommit.Hash != "" {
			res = []sdk.VCSCommit{lastCommit}
		}
	} else {
		//If we only get current node run hash
		log.Debug("GetNodeRunBuildCommits>  Looking for every commit until %s ", cur.Hash)
		c, err := client.Commits(repo, cur.Branch, "", cur.Hash)
		if err != nil {
			// cur.Hash could be empty -> 404 returned from vcs, it's not an error
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, cur, sdk.WrapError(err, "GetNodeRunBuildCommits> Cannot get commits")
			}
		}
		res = c
	}

	return res, cur, nil
}

// PreviousNodeRun find previous node run
func PreviousNodeRun(db gorp.SqlExecutor, nr sdk.WorkflowNodeRun, n sdk.WorkflowNode, workflowID int64) (sdk.WorkflowNodeRun, error) {
	query := fmt.Sprintf(`
					SELECT %s FROM workflow_node_run
					JOIN workflow_node ON workflow_node.name = $1 AND workflow_node.workflow_id = $2
					WHERE vcs_branch = $3 AND workflow_node_run.num <= $4 AND workflow_node_run.workflow_node_id = $5 AND workflow_node_run.id != $6
					ORDER BY workflow_node_run.num, workflow_node_run.sub_num DESC
					LIMIT 1
				`, nodeRunFields)
	var nodeRun sdk.WorkflowNodeRun
	var rr = NodeRun{}
	if err := db.SelectOne(&rr, query, n.Name, workflowID, nr.VCSBranch, nr.Number, nr.WorkflowNodeID, nr.ID); err != nil {
		return nodeRun, sdk.WrapError(err, "PreviousNodeRun> Cannot load previous run: %s [%s %d %s %d %d]", query, n.Name, workflowID, nr.VCSBranch, nr.Number, nr.WorkflowNodeID)
	}
	pNodeRun, errF := fromDBNodeRun(rr, LoadRunOptions{})
	if errF != nil {
		return nodeRun, sdk.WrapError(errF, "PreviousNodeRun> Cannot read node run")
	}
	nodeRun = *pNodeRun
	return nodeRun, nil
}

//PreviousNodeRunVCSInfos returns a struct with BuildNumber, Commit Hash, Branch, Remote, Remote_url
//for the current node run and the previous one on the same branch.
//Returned value may be zero if node run are not found
//If you don't have environment linked set envID to 0 or -1
func PreviousNodeRunVCSInfos(db gorp.SqlExecutor, projectKey string, wf *sdk.Workflow, nodeName string, current sdk.BuildNumberAndHash, appID int64, envID int64) (sdk.BuildNumberAndHash, error) {
	var previous sdk.BuildNumberAndHash
	var prevHash, prevBranch, prevRepository sql.NullString
	var previousBuildNumber sql.NullInt64

	queryPrevious := `
		SELECT workflow_node_run.vcs_branch, workflow_node_run.vcs_hash, workflow_node_run.vcs_repository, workflow_node_run.num
		FROM workflow_node_run
		JOIN workflow_node ON workflow_node.name = workflow_node_run.workflow_node_name AND workflow_node.name = $1 AND workflow_node.workflow_id = $2
		JOIN workflow_node_context ON workflow_node_context.workflow_node_id = workflow_node.id
		WHERE workflow_node_run.vcs_hash IS NOT NULL
		AND workflow_node_run.num < $3
    AND workflow_node_context.application_id = $4
	`

	argPrevious := []interface{}{nodeName, wf.ID, current.BuildNumber, appID}
	if envID > 0 {
		argPrevious = append(argPrevious, envID)
		queryPrevious += "AND workflow_node_context.environment_id = $5"
	}
	queryPrevious += fmt.Sprintf(" ORDER BY workflow_node_run.num DESC LIMIT 1")

	errPrev := db.QueryRow(queryPrevious, argPrevious...).Scan(&prevBranch, &prevHash, &prevRepository, &previousBuildNumber)
	if errPrev == sql.ErrNoRows {
		log.Warning("PreviousNodeRunVCSInfos> no result with previous %d %s , arguments %v", current.BuildNumber, nodeName, argPrevious)
		return previous, nil
	}
	if errPrev != nil {
		return previous, errPrev
	}

	if prevBranch.Valid {
		previous.Branch = prevBranch.String
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

func updateNodeRunCommits(db gorp.SqlExecutor, id int64, commits []sdk.VCSCommit) error {
	log.Debug("updateNodeRunCommits> Updating %d commits for workflow_node_run #%d", len(commits), id)
	commitsBtes, errMarshal := json.Marshal(commits)
	if errMarshal != nil {
		return sdk.WrapError(errMarshal, "updateNodeRunCommits> Unable to marshal commits")
	}

	if _, err := db.Exec("UPDATE workflow_node_run SET commits = $1 where id = $2", commitsBtes, id); err != nil {
		return sdk.WrapError(err, "updateNodeRunCommits> Unable to update workflow_node_run id=%d", id)
	}
	return nil
}

func updateNodeRunStatusAndStage(db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun) error {
	stagesBts, errMarshal := json.Marshal(nodeRun.Stages)
	if errMarshal != nil {
		return sdk.WrapError(errMarshal, "updateNodeRunStatusAndStage> Unable to marshal stages")
	}

	if _, err := db.Exec("UPDATE workflow_node_run SET status = $1, stages = $2, done = $3 where id = $4", nodeRun.Status, stagesBts, nodeRun.Done, nodeRun.ID); err != nil {
		return sdk.WrapError(err, "updateNodeRunStatusAndStage> Unable to update workflow_node_run %s", nodeRun.WorkflowNodeName)
	}
	return nil
}

func updateNodeRunStatusAndTriggersRun(db gorp.SqlExecutor, nodeRun *sdk.WorkflowNodeRun) error {
	triggersRunbts, errMarshal := json.Marshal(nodeRun.TriggersRun)
	if errMarshal != nil {
		return sdk.WrapError(errMarshal, "updateNodeRunStatusAndStage> Unable to marshal triggers run")
	}

	if _, err := db.Exec("UPDATE workflow_node_run SET status = $1, triggers_run = $2 where id = $3", nodeRun.Status, triggersRunbts, nodeRun.ID); err != nil {
		return sdk.WrapError(err, "updateNodeRunStatusAndStage> Unable to update workflow_node_run %s", nodeRun.WorkflowNodeName)
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
