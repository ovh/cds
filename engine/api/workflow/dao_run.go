package workflow

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// insertWorkflowRun inserts in table "workflow_run""
func insertWorkflowRun(db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	runDB := Run(*wr)
	if err := db.Insert(&runDB); err != nil {
		return sdk.WrapError(err, "insertWorkflowRun> Unable to insert run")
	}
	wr.ID = runDB.ID
	return nil
}

// updateWorkflowRun updates in table "workflow_run""
func updateWorkflowRun(db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	wr.LastModified = time.Now()

	for _, info := range wr.Infos {
		if info.IsError {
			wr.Status = string(sdk.StatusFail)
		}
	}

	runDB := Run(*wr)
	if _, err := db.Update(&runDB); err != nil {
		return sdk.WrapError(err, "updateWorkflowRun> Unable to update workflow run")
	}
	wr.ID = runDB.ID
	return nil
}

//UpdateWorkflowRunStatus update status of a workflow run
func UpdateWorkflowRunStatus(db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	wr.LastModified = time.Now()
	//Update workflow run status
	query := "UPDATE workflow_run SET status = $1, last_modified = $2 WHERE id = $3"
	if _, err := db.Exec(query, wr.Status, wr.LastModified, wr.ID); err != nil {
		return sdk.WrapError(err, "updateWorkflowRunStatus> Unable to set  workflow_run id %d with status %s", wr.ID, wr.Status)
	}
	return nil
}

//PostInsert is a db hook on WorkflowRun
func (r *Run) PostInsert(db gorp.SqlExecutor) error {
	w, errw := json.Marshal(r.Workflow)
	if errw != nil {
		return sdk.WrapError(errw, "Run.PostInsert> Unable to marshal workflow")
	}

	jtr, erri := json.Marshal(r.JoinTriggersRun)
	if erri != nil {
		return sdk.WrapError(erri, "Run.PostInsert> Unable to marshal JoinTriggersRun")
	}

	i, erri := json.Marshal(r.Infos)
	if erri != nil {
		return sdk.WrapError(erri, "Run.PostInsert> Unable to marshal infos")
	}

	if _, err := db.Exec("update workflow_run set workflow = $3, infos = $2, join_triggers_run = $4 where id = $1", r.ID, i, w, jtr); err != nil {
		return sdk.WrapError(err, "Run.PostInsert> Unable to store marshalled infos")
	}

	if err := updateTags(db, r); err != nil {
		return sdk.WrapError(err, "Run.PostInsert> Unable to store tags")
	}

	return nil
}

//PostUpdate is a db hook on WorkflowRun
func (r *Run) PostUpdate(db gorp.SqlExecutor) error {
	return r.PostInsert(db)
}

//PostGet is a db hook on WorkflowRun
//It loads column workflow wich is in JSONB in table workflow_run
func (r *Run) PostGet(db gorp.SqlExecutor) error {
	var res = struct {
		W sql.NullString `db:"workflow"`
		I sql.NullString `db:"infos"`
		J sql.NullString `db:"join_triggers_run"`
	}{}

	if err := db.SelectOne(&res, "select workflow, infos, join_triggers_run from workflow_run where id = $1", r.ID); err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to load marshalled workflow")
	}

	w := sdk.Workflow{}
	if err := gorpmapping.JSONNullString(res.W, &w); err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to unmarshal workflow")
	}
	r.Workflow = w

	i := []sdk.WorkflowRunInfo{}
	if err := gorpmapping.JSONNullString(res.I, &i); err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to unmarshal infos")
	}
	r.Infos = i

	j := map[int64]sdk.WorkflowNodeTriggerRun{}
	if err := gorpmapping.JSONNullString(res.J, &j); err != nil {
		return sdk.WrapError(err, "Run.PostGet> Unable to unmarshal join_triggers_run")
	}
	r.JoinTriggersRun = j

	return nil
}

// InsertWorkflowRunTags  inserts new tags in database
func InsertWorkflowRunTags(db gorp.SqlExecutor, runID int64, runTags []sdk.WorkflowRunTag) error {
	tags := []interface{}{}
	for i := range runTags {
		runTags[i].WorkflowRunID = runID
		t := RunTag(runTags[i])
		tags = append(tags, &t)
	}

	if len(tags) > 0 {
		if err := db.Insert(tags...); err != nil {
			return sdk.WrapError(err, "Run.updateTags> Unable to store tags")
		}
	}
	return nil
}

// UpdateWorkflowRunTags updates new tags in database
func UpdateWorkflowRunTags(db gorp.SqlExecutor, r *sdk.WorkflowRun) error {
	run := Run(*r)
	return updateTags(db, &run)
}

func updateTags(db gorp.SqlExecutor, r *Run) error {
	if _, err := db.Exec("delete from workflow_run_tag where workflow_run_id = $1", r.ID); err != nil {
		return sdk.WrapError(err, "Run.updateTags> Unable to store tags")
	}

	return InsertWorkflowRunTags(db, r.ID, r.Tags)
}

// LoadLastRun returns the last run for a workflow
func LoadLastRun(db gorp.SqlExecutor, projectkey, workflowname string, withArtifacts bool) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.*
	from workflow_run
	join project on workflow_run.project_id = project.id
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1
	and workflow.name = $2
	order by workflow_run.num desc limit 1`
	return loadRun(db, withArtifacts, query, projectkey, workflowname)
}

// LoadRun returns a specific run
func LoadRun(db gorp.SqlExecutor, projectkey, workflowname string, number int64, withArtifacts bool) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.*
	from workflow_run
	join project on workflow_run.project_id = project.id
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1
	and workflow.name = $2
	and workflow_run.num = $3`
	return loadRun(db, withArtifacts, query, projectkey, workflowname, number)
}

// LoadRunByIDAndProjectKey returns a specific run
func LoadRunByIDAndProjectKey(db gorp.SqlExecutor, projectkey string, id int64, withArtifacts bool) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.*
	from workflow_run
	join project on workflow_run.project_id = project.id
	where project.projectkey = $1
	and workflow_run.id = $2`
	return loadRun(db, withArtifacts, query, projectkey, id)
}

// LoadRunByID loads run by ID
func LoadRunByID(db gorp.SqlExecutor, id int64, withArtifacts bool) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.*
	from workflow_run
	where workflow_run.id = $1`
	return loadRun(db, withArtifacts, query, id)
}

// LoadAndLockRunByJobID loads a run by a job id
func LoadAndLockRunByJobID(db gorp.SqlExecutor, id int64, withArtifacts bool) (*sdk.WorkflowRun, error) {
	query := `select workflow_run.*
	from workflow_run
	join workflow_node_run on workflow_run.id = workflow_node_run.workflow_run_id
	join workflow_node_run_job on workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
	where workflow_node_run_job.id = $1 for update`
	return loadRun(db, withArtifacts, query, id)
}

//LoadRuns loads all runs
//It retuns runs, offset, limit count and an error
func LoadRuns(db gorp.SqlExecutor, projectkey, workflowname string, offset, limit int) ([]sdk.WorkflowRun, int, int, int, error) {
	queryCount := `select count(workflow_run.id)
	from workflow_run
	join project on workflow_run.project_id = project.id
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1
	and workflow.name = $2`

	count, errc := db.SelectInt(queryCount, projectkey, workflowname)
	if errc != nil {
		return nil, 0, 0, 0, sdk.WrapError(errc, "LoadRuns> unable to load runs")
	}
	if count == 0 {
		return nil, 0, 0, 0, nil
	}

	query := `select workflow_run.*
	from workflow_run
	join project on workflow_run.project_id = project.id
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1
	and workflow.name = $2
	order by workflow_run.start desc
	limit $3 offset $4`

	runs := []Run{}
	if _, err := db.Select(&runs, query, projectkey, workflowname, limit, offset); err != nil {
		return nil, 0, 0, 0, sdk.WrapError(errc, "LoadRuns> unable to load runs")
	}
	wruns := make([]sdk.WorkflowRun, len(runs))
	for i := range runs {
		wr := sdk.WorkflowRun(runs[i])
		if err := loadRunTags(db, &wr); err != nil {
			return nil, 0, 0, 0, sdk.WrapError(err, "LoadRuns> unable to load tags")
		}

		wruns[i] = wr
	}

	return wruns, offset, limit, int(count), nil
}

func loadRunTags(db gorp.SqlExecutor, run *sdk.WorkflowRun) error {
	dbRunTags := []RunTag{}
	if _, err := db.Select(&dbRunTags, "SELECT * from workflow_run_tag WHERE workflow_run_id=$1", run.ID); err != nil {
		return sdk.WrapError(err, "loadRunTags")
	}

	run.Tags = make([]sdk.WorkflowRunTag, len(dbRunTags))
	for i := range dbRunTags {
		run.Tags[i] = sdk.WorkflowRunTag(dbRunTags[i])
	}
	return nil
}

func loadRun(db gorp.SqlExecutor, withArtifacts bool, query string, args ...interface{}) (*sdk.WorkflowRun, error) {
	runDB := &Run{}
	if err := db.SelectOne(runDB, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "loadRun> Unable to load workflow run. query:%s args:%v", query, args)
	}
	wr := sdk.WorkflowRun(*runDB)

	q := "select workflow_node_run.* from workflow_node_run where workflow_run_id = $1 ORDER BY workflow_node_run.sub_num DESC"
	dbNodeRuns := []NodeRun{}
	if _, err := db.Select(&dbNodeRuns, q, wr.ID); err != nil {
		if err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "loadRun> Unable to load workflow nodes run")
		}
	}

	for _, n := range dbNodeRuns {
		wnr, err := fromDBNodeRun(n)
		if err != nil {
			return nil, err
		}
		if wr.WorkflowNodeRuns == nil {
			wr.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
		}
		wnr.CanBeRun = canBeRun(&wr, wnr)

		if withArtifacts {
			arts, errA := loadArtifactByNodeRunID(db, wnr.ID)
			if errA != nil {
				return nil, sdk.WrapError(errA, "loadRun>Error loading artifacts for run %d", wnr.ID)
			}
			wnr.Artifacts = arts
		}

		wr.WorkflowNodeRuns[wnr.WorkflowNodeID] = append(wr.WorkflowNodeRuns[wnr.WorkflowNodeID], *wnr)
	}

	for k := range wr.WorkflowNodeRuns {
		sort.Slice(wr.WorkflowNodeRuns[k], func(i, j int) bool {
			return wr.WorkflowNodeRuns[k][i].SubNumber > wr.WorkflowNodeRuns[k][j].SubNumber
		})
	}

	tags, errT := loadTagsByRunID(db, wr.ID)
	if errT != nil {
		return nil, sdk.WrapError(errT, "loadRun> Error loading tags for run %d", wr.ID)
	}
	wr.Tags = tags

	return &wr, nil
}

//TODO: if no bugs are found, it could be used to refactor process.go
// canBeRun return boolean to know if a wokrflow node run can be run or not
func canBeRun(workflowRun *sdk.WorkflowRun, workflowNodeRun *sdk.WorkflowNodeRun) bool {
	if !sdk.StatusIsTerminated(workflowNodeRun.Status) {
		return false
	}
	if workflowRun == nil {
		return false
	}
	node := workflowRun.Workflow.GetNode(workflowNodeRun.WorkflowNodeID)
	if node == nil {
		return true
	}

	ancestorsID := node.Ancestors(&workflowRun.Workflow, true)
	if ancestorsID == nil || len(ancestorsID) == 0 {
		return true
	}

	for _, ancestorID := range ancestorsID {
		nodeRuns, ok := workflowRun.WorkflowNodeRuns[ancestorID]
		if ok && (len(nodeRuns) == 0 || !sdk.StatusIsTerminated(nodeRuns[0].Status) ||
			nodeRuns[0].Status == "" || nodeRuns[0].Status == sdk.StatusNeverBuilt.String()) {
			return false
		}
	}

	return true
}

func loadTagsByRunID(db gorp.SqlExecutor, runID int64) ([]sdk.WorkflowRunTag, error) {
	tags := []sdk.WorkflowRunTag{}
	dbTags := []sdk.WorkflowRunTag{}
	if _, err := db.Select(&dbTags, "select * from workflow_run_tag where workflow_run_id = $1", runID); err != nil {
		return nil, sdk.WrapError(err, "loadTagsByRunID> Unable to load tags for run %d", runID)
	}
	for i := range dbTags {
		tags = append(tags, sdk.WorkflowRunTag(dbTags[i]))
	}
	return tags, nil
}

// GetTagsAndValue returns a map of tags and all the values available on all runs of a workflow
func GetTagsAndValue(db gorp.SqlExecutor, key, name string) (map[string][]string, error) {
	query := `
SELECT tags.tag "tag", STRING_AGG(tags.value, ',') "values"
FROM (
        SELECT distinct tag "tag", value "value"
        FROM workflow_run_tag
		JOIN workflow_run ON workflow_run_tag.workflow_run_id = workflow_run.id
		JOIN workflow ON workflow_run.workflow_id = workflow.id
		JOIN project ON workflow.project_id = project.id
		WHERE project.projectkey = $1
		AND workflow.name = $2
		order by value
    ) AS "tags"
GROUP BY tags.tag
ORDER BY tags.tag;
`

	res := []struct {
		Tag    string `db:"tag"`
		Values string `db:"values"`
	}{}

	if _, err := db.Select(&res, query, key, name); err != nil {
		return nil, sdk.WrapError(err, "GetTagsAndValue> Unable to load tags and values")
	}

	rmap := map[string][]string{}
	for _, r := range res {
		rmap[r.Tag] = strings.Split(r.Values, ",")
	}

	return rmap, nil
}

// LoadCurrentRunNum load the current num from workflow_sequences table
func LoadCurrentRunNum(db gorp.SqlExecutor, projectkey, workflowname string) (int64, error) {
	query := `SELECT COALESCE(workflow_sequences.current_val, 0) as run_num
			FROM workflow
			LEFT JOIN workflow_sequences ON workflow.id = workflow_sequences.workflow_id
			JOIN project ON project.id = workflow.project_id
			WHERE project.projectkey = $1 AND workflow.name = $2
    `
	i, err := db.SelectInt(query, projectkey, workflowname)
	if err != nil {
		return 0, sdk.WrapError(err, "LoadCurrentRunNum> Cannot load workflow run current num")
	}
	return int64(i), nil
}

// InsertRunNum Insert run number for the given workflow
func InsertRunNum(db gorp.SqlExecutor, w *sdk.Workflow, num int64) error {
	query := `
		INSERT INTO workflow_sequences (workflow_id, current_val) VALUES ($1, $2)
	`
	if _, err := db.Exec(query, w.ID, num); err != nil {
		return sdk.WrapError(err, "InsertRunNum> Cannot insert run number")
	}
	return nil
}

// UpdateRunNum Update run number for the given workflow
func UpdateRunNum(db gorp.SqlExecutor, w *sdk.Workflow, num int64) error {
	if num == 1 {
		if _, err := nextRunNumber(db, w); err != nil {
			return sdk.WrapError(err, "UpdateRunNum> Cannot create run number")
		}
		return nil
	}

	query := `
		UPDATE workflow_sequences set current_val = $1 WHERE workflow_id = $2
	`
	if _, err := db.Exec(query, num, w.ID); err != nil {
		return sdk.WrapError(err, "UpdateRunNum> Cannot update run number")
	}
	return nil
}

func nextRunNumber(db gorp.SqlExecutor, w *sdk.Workflow) (int64, error) {
	i, err := db.SelectInt("select workflow_sequences_nextval($1)", w.ID)
	if err != nil {
		return 0, sdk.WrapError(err, "nextRunNumber")
	}
	log.Debug("nextRunNumber> %s/%s %d", w.ProjectKey, w.Name, i)
	return int64(i), nil
}

// PurgeWorkflowRun mark all workflow run to delete
func PurgeWorkflowRun(db gorp.SqlExecutor, wf sdk.Workflow) error {
	ids := []struct {
		Ids string `json:"ids" db:"ids"`
	}{}

	if wf.HistoryLength == 0 {
		log.Warning("PurgeWorkflowRun> history length equals 0, skipping purge")
		return nil
	}

	if len(wf.PurgeTags) == 0 {
		qDelete := `
			UPDATE workflow_run SET to_delete = true
			WHERE workflow_run.id IN (
				SELECT workflow_run.id FROM workflow_run
				WHERE workflow_run.workflow_id = $1
				ORDER BY workflow_run.id DESC OFFSET $2 ROWS
			)
		`
		if _, err := db.Exec(qDelete, wf.ID, wf.HistoryLength); err != nil {
			log.Warning("PurgeWorkflowRun> Unable to update workflow run for purge without tags for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, err)
			return err
		}

		return nil
	}

	queryGetIds := `SELECT string_agg(id::text, ',') AS ids FROM
		(SELECT workflow_run.id AS id, workflow_run_tag.tag AS tag, workflow_run_tag.value AS value FROM workflow_run
		JOIN workflow_run_tag ON workflow_run.id = workflow_run_tag.workflow_run_id
		WHERE workflow_run.workflow_id = $1 AND workflow_run_tag.tag = ANY(string_to_array($2, ',')::text[]) ORDER BY workflow_run.id DESC) as wr
		GROUP BY tag, value HAVING COUNT(id) > $3
	`
	if _, errS := db.Select(&ids, queryGetIds, wf.ID, strings.Join(wf.PurgeTags, ","), wf.HistoryLength); errS != nil {
		log.Warning("PurgeWorkflowRun> Unable to get workflow run for purge with workflow id %d, tags %v and history length %d : %s", wf.ID, wf.PurgeTags, wf.HistoryLength, errS)
		return errS
	}

	idsToUpdate := []string{}
	for _, idToUp := range ids {
		if idToUp.Ids != "" {
			idsToUpdate = append(idsToUpdate, strings.Join(strings.Split(idToUp.Ids, ",")[wf.HistoryLength:], ","))
		}
	}

	queryUpdate := `UPDATE workflow_run SET to_delete = true WHERE workflow_run.id = ANY(string_to_array($1, ',')::bigint[])`
	if _, err := db.Exec(queryUpdate, strings.Join(idsToUpdate, ",")); err != nil {
		log.Warning("PurgeWorkflowRun> Unable to update workflow run for purge for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, err)
		return err
	}
	return nil
}

// deleteWorkflowRunsHistory is useful to delete all the workflow run marked with to delete flag in db
func deleteWorkflowRunsHistory(db gorp.SqlExecutor) error {
	query := `DELETE FROM workflow_run WHERE to_delete = true`

	if _, err := db.Exec(query); err != nil {
		log.Warning("deleteWorkflowRunsHistory> Unable to delete workflow history %s", err)
		return err
	}
	return nil
}
