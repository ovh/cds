package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

const wfRunfields string = `
workflow_run.id,
workflow_run.num,
workflow_run.project_id,
workflow_run.workflow_id,
workflow_run.start,
workflow_run.last_modified,
workflow_run.status,
workflow_run.last_sub_num,
workflow_run.last_execution,
workflow_run.to_delete,
workflow_run.read_only,
workflow_run.version,
workflow_run.to_craft,
workflow_run.to_craft_opts,
workflow_run.workflow,
workflow_run.infos,
workflow_run.join_triggers_run,
workflow_run.header
`

// LoadRunOptions are options for loading a run (node or workflow)
type LoadRunOptions struct {
	WithCoverage            bool
	WithArtifacts           bool
	WithStaticFiles         bool
	WithTests               bool
	WithLightTests          bool
	WithVulnerabilities     bool
	WithDeleted             bool
	DisableDetailledNodeRun bool
	Language                string
}

// insertWorkflowRun inserts in table "workflow_run""
func insertWorkflowRun(db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	runDB := Run(*wr)
	if err := db.Insert(&runDB); err != nil {
		return sdk.WrapError(err, "Unable to insert run")
	}
	wr.ID = runDB.ID
	return nil
}

// UpdateWorkflowRun updates in table "workflow_run""
func UpdateWorkflowRun(ctx context.Context, db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	_, end := telemetry.Span(ctx, "workflow.UpdateWorkflowRun")
	defer end()

	wr.LastModified = time.Now()
	for _, info := range wr.Infos {
		if info.Type == sdk.RunInfoTypeError && info.SubNumber == wr.LastSubNumber {
			wr.Status = sdk.StatusFail
		}
	}

	if sdk.StatusIsTerminated(wr.Status) {
		wr.LastExecution = time.Now()
	}

	runDB := Run(*wr)
	if _, err := db.Update(&runDB); err != nil {
		return sdk.WrapError(err, "Unable to update workflow run")
	}
	wr.ID = runDB.ID
	return nil
}

//UpdateWorkflowRunStatus update status of a workflow run
func UpdateWorkflowRunStatus(db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {
	wr.LastModified = time.Now()
	if sdk.StatusIsTerminated(wr.Status) {
		wr.LastExecution = time.Now()
	}
	//Update workflow run status
	query := "UPDATE workflow_run SET status = $1, last_modified = $2, last_execution = $3 WHERE id = $4"
	if _, err := db.Exec(query, wr.Status, wr.LastModified, wr.LastExecution, wr.ID); err != nil {
		return sdk.WrapError(err, "Unable to set  workflow_run id %d with status %s", wr.ID, wr.Status)
	}
	return nil
}

// LoadWorkflowFromWorkflowRunID loads the workflow for the given workfloxw run id
func LoadWorkflowFromWorkflowRunID(db gorp.SqlExecutor, wrID int64) (sdk.Workflow, error) {
	var workflow sdk.Workflow
	wNS, err := db.SelectNullStr("SELECT workflow FROM workflow_run WHERE id = $1", wrID)
	if err != nil {
		return workflow, sdk.WrapError(err, "Unable to load workflow for workflow run %d", wrID)
	}
	if err := gorpmapping.JSONNullString(wNS, &workflow); err != nil {
		return workflow, sdk.WrapError(err, "Unable to write into workflow struct")
	}
	return workflow, nil
}

//PostInsert is a db hook on WorkflowRun
func (r *Run) PostInsert(db gorp.SqlExecutor) error {
	if err := updateTags(db, r); err != nil {
		return sdk.WrapError(err, "Unable to store tags")
	}

	return nil
}

//PostUpdate is a db hook on WorkflowRun
func (r *Run) PostUpdate(db gorp.SqlExecutor) error {
	return r.PostInsert(db)
}

// InsertWorkflowRunTags  inserts new tags in database
func InsertWorkflowRunTags(db gorp.SqlExecutor, runID int64, runTags []sdk.WorkflowRunTag) error {
	tags := []interface{}{}
	for i := range runTags {
		runTags[i].WorkflowRunID = runID
		t := RunTag(runTags[i])
		// we truncate tag.Value to 250 chars, with '...' after.
		// this will avoid to have an error from db.
		if len(t.Value) >= 256 {
			t.Value = fmt.Sprintf("%s...", t.Value[:250])
		}
		// and we take only 250 first chars for tag name
		if len(t.Tag) >= 256 {
			t.Tag = t.Tag[:250]
		}
		tags = append(tags, &t)
	}

	if len(tags) > 0 {
		if err := db.Insert(tags...); err != nil {
			return sdk.WrapError(err, "Unable to store tags")
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
		return sdk.WrapError(err, "Unable to store tags")
	}
	return InsertWorkflowRunTags(db, r.ID, r.Tags)
}

// LoadLastRun returns the last run for a workflow
func LoadLastRun(db gorp.SqlExecutor, projectkey, workflowname string, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	join project on workflow_run.project_id = project.id
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1
	and workflow.name = $2
	order by workflow_run.num desc limit 1`, wfRunfields)
	return loadRun(db, loadOpts, query, projectkey, workflowname)
}

// LoadLastRuns returns the last run per workflowIDs
func LoadLastRuns(db gorp.SqlExecutor, workflowIDs []int64, limit int) ([]sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	where workflow_run.workflow_id = ANY($1)
	order by workflow_run.workflow_id, workflow_run.num desc limit $2`, wfRunfields)
	return loadRuns(db, query, pq.Int64Array(workflowIDs), limit)
}

// LoadRun returns a specific run
func LoadRun(ctx context.Context, db gorp.SqlExecutor, projectkey, workflowname string, number int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	_, end := telemetry.Span(ctx, "workflow.LoadRun",
		telemetry.Tag(telemetry.TagProjectKey, projectkey),
		telemetry.Tag(telemetry.TagWorkflow, workflowname),
		telemetry.Tag(telemetry.TagWorkflowRun, number),
	)
	defer end()
	query := fmt.Sprintf(`select %s
	from workflow_run
	join project on workflow_run.project_id = project.id
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1
	and workflow.name = $2
	and workflow_run.num = $3`, wfRunfields)
	return loadRun(db, loadOpts, query, projectkey, workflowname, number)
}

// LoadRunByIDAndProjectKey returns a specific run
func LoadRunByIDAndProjectKey(db gorp.SqlExecutor, projectkey string, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	join project on workflow_run.project_id = project.id
	where project.projectkey = $1
	and workflow_run.id = $2`, wfRunfields)
	return loadRun(db, loadOpts, query, projectkey, id)
}

// LoadRunByID loads run by ID
func LoadRunByID(db gorp.SqlExecutor, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	where workflow_run.id = $1`, wfRunfields)
	return loadRun(db, loadOpts, query, id)
}

// LoadAndLockRunByID loads run by ID
func LoadAndLockRunByID(db gorp.SqlExecutor, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	where workflow_run.id = $1 for update skip locked`, wfRunfields)
	return loadRun(db, loadOpts, query, id)
}

// LoadAndLockRunByJobID loads a run by a job id
func LoadAndLockRunByJobID(db gorp.SqlExecutor, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	join workflow_node_run on workflow_run.id = workflow_node_run.workflow_run_id
	join workflow_node_run_job on workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
	where workflow_node_run_job.id = $1 for update skip locked`, wfRunfields)
	return loadRun(db, loadOpts, query, id)
}

func loadRuns(db gorp.SqlExecutor, query string, args ...interface{}) ([]sdk.WorkflowRun, error) {
	runs := []Run{}
	if _, err := db.Select(&runs, query, args...); err != nil {
		return nil, sdk.WrapError(err, "Unable to load runs")
	}
	wruns := make([]sdk.WorkflowRun, len(runs))
	for i := range runs {
		wr := sdk.WorkflowRun(runs[i])
		tags, err := loadRunTags(db, wr.ID)
		if err != nil {
			return nil, err
		}
		wr.Tags = tags
		wruns[i] = wr
	}
	return wruns, nil
}

func LoadRunsIDsToDelete(db gorp.SqlExecutor, offset int64, limit int64) ([]int64, int64, int64, int64, error) {
	queryCount := `SELECT COUNT(id) FROM workflow_run WHERE to_delete = true`
	count, err := db.SelectInt(queryCount)
	if err != nil {
		return nil, 0, 0, 0, sdk.WithStack(err)
	}
	if count == 0 {
		return nil, 0, 0, 0, nil
	}

	var ids []int64
	querySelect := `SELECT id FROM workflow_run 
					WHERE to_delete = true
					ORDER BY workflow_run.start ASC limit $1 offset $2`
	_, err = db.Select(&ids, querySelect, limit, offset)
	if err != nil {
		return nil, 0, 0, 0, sdk.WithStack(err)
	}
	return ids, offset, limit, count, nil
}

//LoadRunsSummaries loads a short version of workflow runs
//It returns runs, offset, limit count and an error
func LoadRunsSummaries(db gorp.SqlExecutor, projectkey, workflowname string, offset, limit int, tagFilter map[string]string) ([]sdk.WorkflowRunSummary, int, int, int, error) {
	queryCount := `select count(workflow_run.id)
					from workflow_run
					join project on workflow_run.project_id = project.id
					join workflow on workflow_run.workflow_id = workflow.id
					where project.projectkey = $1
					and workflow.name = $2
					AND workflow_run.to_delete = false`

	count, errc := db.SelectInt(queryCount, projectkey, workflowname)
	if errc != nil {
		return nil, 0, 0, 0, sdk.WrapError(errc, "unable to load short runs")
	}
	if count == 0 {
		return nil, 0, 0, 0, nil
	}

	selectedColumn := "wr.id, wr.num, wr.status, wr.start, wr.last_modified, wr.last_sub_num, wr.last_execution, wr.version, wr.to_craft_opts"
	args := []interface{}{projectkey, workflowname, limit, offset}
	query := fmt.Sprintf(`
			SELECT %s
			FROM workflow_run wr
			JOIN project ON wr.project_id = project.id
			JOIN workflow ON wr.workflow_id = workflow.id
			WHERE project.projectkey = $1
			AND workflow.name = $2
			AND wr.to_delete = false
			ORDER BY wr.start desc
			LIMIT $3 OFFSET $4`, selectedColumn)

	if len(tagFilter) > 0 {
		// Posgres operator: '<@' means 'is contained by' eg. 'ARRAY[2,7] <@ ARRAY[1,7,4,2,6]' ==> returns true
		query = fmt.Sprintf(`select %s
		from workflow_run wr
		join project on wr.project_id = project.id
		join workflow on wr.workflow_id = workflow.id
		join (
			select workflow_run_id, string_agg(all_tags, ',') as tags
			from (
				select workflow_run_id, tag || '=' || value "all_tags"
				from workflow_run_tag
				order by tag
			) as all_wr_tags
			group by workflow_run_id
		) as tags on wr.id = tags.workflow_run_id
		where project.projectkey = $1
		and workflow.name = $2
		AND wr.to_delete = false
		and string_to_array($5, ',') <@ string_to_array(tags.tags, ',')
		order by wr.start desc
		limit $3 offset $4`, selectedColumn)

		var tags []string
		for k, v := range tagFilter {
			tags = append(tags, k+"="+v)
		}

		log.Debug("tags=%v", tags)

		args = append(args, strings.Join(tags, ","))
	}

	var shortRuns []sdk.WorkflowRunSummary
	_, err := db.Select(&shortRuns, query, args...)
	if err != nil {
		return nil, 0, 0, 0, sdk.WithStack(err)
	}
	for i := range shortRuns {
		run := &shortRuns[i]
		tags, err := loadRunTags(db, run.ID)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		run.Tags = tags
	}
	return shortRuns, offset, limit, int(count), nil
}

// LoadRunsIDByTag load workflow run ids for given tag and his value
func LoadRunsIDByTag(db gorp.SqlExecutor, projectKey, workflowName, tag, tagValue string) ([]int64, error) {
	query := `SELECT workflow_run.id
		FROM workflow_run
		JOIN project on workflow_run.project_id = project.id
		JOIN workflow on workflow_run.workflow_id = workflow.id
		JOIN (
			SELECT workflow_run_id, string_agg(all_tags, ',') AS tags
			FROM (
				SELECT workflow_run_id, tag || '=' || value "all_tags"
				FROM workflow_run_tag
				WHERE workflow_run_tag.tag = $3 AND workflow_run_tag.value = $4
				ORDER BY tag
			) AS all_wr_tags
			GROUP BY workflow_run_id
		) AS tags on workflow_run.id = tags.workflow_run_id
		WHERE project.projectkey = $1
		AND workflow.name = $2
		ORDER BY workflow_run.start DESC`

	idsDB := []struct {
		ID int64 `db:"id"`
	}{}
	if _, err := db.Select(&idsDB, query, projectKey, workflowName, tag, tagValue); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot load runs id by tag")
	}

	ids := make([]int64, len(idsDB))
	for i := range idsDB {
		ids[i] = idsDB[i].ID
	}

	return ids, nil
}

func loadRunTags(db gorp.SqlExecutor, runID int64) ([]sdk.WorkflowRunTag, error) {
	dbRunTags := []RunTag{}
	if _, err := db.Select(&dbRunTags, "SELECT * from workflow_run_tag WHERE workflow_run_id=$1", runID); err != nil {
		return nil, sdk.WithStack(err)
	}

	tags := make([]sdk.WorkflowRunTag, 0, len(dbRunTags))
	for i := range dbRunTags {
		tags = append(tags, sdk.WorkflowRunTag(dbRunTags[i]))
	}
	return tags, nil
}

func loadRun(db gorp.SqlExecutor, loadOpts LoadRunOptions, query string, args ...interface{}) (*sdk.WorkflowRun, error) {
	runDB := &Run{}
	if err := db.SelectOne(runDB, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, sdk.WrapError(err, "Unable to load workflow run. query:%s args:%v", query, args)
	}
	wr := sdk.WorkflowRun(*runDB)
	if !loadOpts.WithDeleted && wr.ToDelete {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	tags, errT := loadTagsByRunID(db, wr.ID)
	if errT != nil {
		return nil, sdk.WrapError(errT, "loadRun> Error loading tags for run %d", wr.ID)
	}
	wr.Tags = tags

	if err := syncNodeRuns(db, &wr, loadOpts); err != nil {
		return nil, sdk.WrapError(err, "Unable to load workflow node run")
	}

	return &wr, nil
}

// CanBeRun return boolean to know if a workflow node run can be run or not
//TODO: if no bugs are found, it could be used to refactor process.go
func CanBeRun(workflowRun *sdk.WorkflowRun, workflowNodeRun *sdk.WorkflowNodeRun) bool {
	if !sdk.StatusIsTerminated(workflowNodeRun.Status) {
		return false
	}
	if workflowRun == nil {
		return false
	}

	node := workflowRun.Workflow.WorkflowData.NodeByID(workflowNodeRun.WorkflowNodeID)
	if node == nil {
		return true
	}
	ancestorsID := node.Ancestors(workflowRun.Workflow.WorkflowData)

	if len(ancestorsID) == 0 {
		return true
	}
	for _, ancestorID := range ancestorsID {
		nodeRuns, ok := workflowRun.WorkflowNodeRuns[ancestorID]
		if ok && (len(nodeRuns) == 0 || !sdk.StatusIsTerminated(nodeRuns[0].Status) ||
			nodeRuns[0].Status == sdk.StatusNeverBuilt) {
			return false
		}
	}
	return true
}

func loadTagsByRunID(db gorp.SqlExecutor, runID int64) ([]sdk.WorkflowRunTag, error) {
	tags := []sdk.WorkflowRunTag{}
	dbTags := []sdk.WorkflowRunTag{}
	if _, err := db.Select(&dbTags, "select * from workflow_run_tag where workflow_run_id = $1", runID); err != nil {
		return nil, sdk.WrapError(err, "Unable to load tags for run %d", runID)
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
		return nil, sdk.WrapError(err, "Unable to load tags and values")
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
		return 0, sdk.WrapError(err, "Cannot load workflow run current num")
	}
	return int64(i), nil
}

// InsertRunNum Insert run number for the given workflow
func InsertRunNum(db gorp.SqlExecutor, w *sdk.Workflow, num int64) error {
	query := `
		INSERT INTO workflow_sequences (workflow_id, current_val) VALUES ($1, $2)
	`
	if _, err := db.Exec(query, w.ID, num); err != nil {
		return sdk.WrapError(err, "Cannot insert run number")
	}
	return nil
}

func LoadCratingWorkflowRunIDs(db gorp.SqlExecutor) ([]int64, error) {
	query := `
		SELECT id
		FROM workflow_run
		WHERE to_craft = true
		LIMIT 10
	`
	var ids []int64
	_, err := db.Select(&ids, query)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to load crafting workflow runs")
	}
	return ids, nil
}

func UpdateCraftedWorkflowRun(db gorp.SqlExecutor, id int64) error {
	query := `UPDATE workflow_run
	SET to_craft = false
	WHERE id = $1
	`
	if _, err := db.Exec(query, id); err != nil {
		return sdk.WrapError(err, "unable to update crafting workflow run %d", id)
	}
	return nil
}

// CreateRun creates a new workflow run and insert it
func CreateRun(db *gorp.DbMap, wf *sdk.Workflow, opts sdk.WorkflowRunPostHandlerOption) (*sdk.WorkflowRun, error) {
	number, err := NextRunNumber(db, wf.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get next run number")
	}

	wr := &sdk.WorkflowRun{
		Number:        number,
		WorkflowID:    wf.ID,
		Start:         time.Now(),
		LastModified:  time.Now(),
		ProjectID:     wf.ProjectID,
		Status:        sdk.StatusPending,
		LastExecution: time.Now(),
		Tags:          make([]sdk.WorkflowRunTag, 0),
		Workflow:      *wf,
		ToCraft:       true,
		ToCraftOpts:   &opts,
	}

	if opts.Hook != nil {
		if trigg, ok := opts.Hook.Payload["cds.triggered_by.username"]; ok {
			wr.Tag(tagTriggeredBy, trigg)
		} else {
			wr.Tag(tagTriggeredBy, "cds.hook")
		}
	} else {
		c, err := authentication.LoadConsumerByID(context.Background(), db, opts.AuthConsumerID,
			authentication.LoadConsumerOptions.WithAuthentifiedUser,
			authentication.LoadConsumerOptions.WithConsumerGroups)
		if err != nil {
			return nil, err
		}

		// Add service for consumer if exists
		s, err := services.LoadByConsumerID(context.Background(), db, c.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, err
		}
		c.Service = s

		wr.Tag(tagTriggeredBy, c.GetUsername())
	}

	tags := wf.Metadata["default_tags"]
	var payload map[string]string
	if opts.Hook != nil {
		payload = opts.Hook.Payload
	}
	if opts.Manual != nil {
		e := dump.NewDefaultEncoder()
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false
		m1, errm1 := e.ToStringMap(opts.Manual)
		if errm1 != nil {
			return nil, sdk.WrapError(errm1, "unable to compute manual payload")
		}
		payload = m1
	}

	for payloadKey := range payload {
		if strings.HasPrefix(payloadKey, "workflownoderunmanual.payload.cds.") {
			return nil, sdk.WrapError(sdk.ErrInvalidPayloadVariable, "cannot use cds. in a payload key (%s)", payloadKey)
		}
	}

	if tags != "" {
		tagsSplited := strings.Split(tags, ",")
		for _, t := range tagsSplited {
			if pTag, hash := payload[t]; hash {
				wr.Tags = append(wr.Tags, sdk.WorkflowRunTag{
					Tag:   t,
					Value: pTag,
				})
			}
		}
	}

	if err := insertWorkflowRun(db, wr); err != nil {
		return nil, sdk.WrapError(err, "unable to create workflow run")
	}
	return wr, nil
}

// UpdateRunNum Update run number for the given workflow
func UpdateRunNum(db gorp.SqlExecutor, w *sdk.Workflow, num int64) error {
	if num == 1 {
		if _, err := NextRunNumber(db, w.ID); err != nil {
			return sdk.WrapError(err, "Cannot create run number")
		}
		return nil
	}

	query := `
		UPDATE workflow_sequences set current_val = $1 WHERE workflow_id = $2
	`
	if _, err := db.Exec(query, num, w.ID); err != nil {
		return sdk.WrapError(err, "Cannot update run number")
	}
	return nil
}

func NextRunNumber(db gorp.SqlExecutor, workflowID int64) (int64, error) {
	i, err := db.SelectInt("select workflow_sequences_nextval($1)", workflowID)
	if err != nil {
		return 0, sdk.WrapError(err, "nextRunNumber")
	}
	return int64(i), nil
}

// PurgeAllWorkflowRunsByWorkflowID marks all workflow to delete given a workflow
func PurgeAllWorkflowRunsByWorkflowID(ctx context.Context, db gorp.SqlExecutor, id int64) (int, error) {
	query := "UPDATE workflow_run SET to_delete = true WHERE workflow_id = $1"
	res, err := db.Exec(query, id)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	n, _ := res.RowsAffected() // nolint
	log.Info(ctx, "PurgeAllWorkflowRunsByWorkflowID> will delete %d workflow runs for workflow %d", n, id)
	return int(n), nil
}

// MarkWorkflowRunsAsDelete marks workflow runs to be deleted
func MarkWorkflowRunsAsDelete(db gorp.SqlExecutor, ids []int64) error {
	idsStr := gorpmapping.IDsToQueryString(ids)
	if _, err := db.Exec("update workflow_run set to_delete = true where id = ANY(string_to_array($1, ',')::int[])", idsStr); err != nil {
		return sdk.WrapError(err, "Unable to mark as delete workflow id %s", idsStr)
	}
	return nil
}

type byInt64Desc []int64

func (a byInt64Desc) Len() int           { return len(a) }
func (a byInt64Desc) Less(i, j int) bool { return a[i] > a[j] }
func (a byInt64Desc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// PurgeWorkflowRun mark all workflow run to delete
func PurgeWorkflowRun(ctx context.Context, db gorp.SqlExecutor, wf sdk.Workflow) error {
	ids := []struct {
		Ids string `json:"ids" db:"ids"`
	}{}

	if wf.HistoryLength == 0 {
		log.Debug("PurgeWorkflowRun> history length equals 0, skipping purge")
		return nil
	}

	filteredPurgeTags := []string{}
	for _, t := range wf.PurgeTags {
		if t != "" {
			filteredPurgeTags = append(filteredPurgeTags, t)
		}
	}

	// Only if there aren't tags
	if len(filteredPurgeTags) == 0 {
		return purgeWorkflowRunWithoutTags(ctx, db, wf)
	}

	//  Only where there are tags
	queryGetIds := `
		SELECT string_agg(id::text, ',') AS ids
			FROM (
				SELECT workflow_run.id AS id, workflow_run_tag.tag AS tag, workflow_run_tag.value AS value
					FROM workflow_run
						JOIN workflow_run_tag ON workflow_run.id = workflow_run_tag.workflow_run_id
					WHERE workflow_run.workflow_id = $1
					AND workflow_run_tag.tag = ANY(string_to_array($2, ',')::text[])
					AND workflow_run.status <> $4
					AND workflow_run.status <> $5
					AND workflow_run.status <> $6
					AND workflow_run.status <> $7
					AND workflow_run.to_delete = false
				ORDER BY workflow_run.id DESC
			) as wr
		GROUP BY tag, value HAVING COUNT(id) > $3
	`

	_, err := db.Select(
		&ids,
		queryGetIds,
		wf.ID,
		strings.Join(filteredPurgeTags, ","),
		wf.HistoryLength,
		sdk.StatusWaiting,
		sdk.StatusBuilding,
		sdk.StatusChecking,
		sdk.StatusPending,
	)
	if err != nil {
		log.Error(ctx, "PurgeWorkflowRun> Unable to get workflow run for purge with workflow id %d, tags %v and history length %d : %s", wf.ID, wf.PurgeTags, wf.HistoryLength, err)
		return sdk.WithStack(err)
	}

	querySuccessIds := `
	SELECT id
		FROM (
		   	SELECT max(id::bigint)::text AS id, status
		     	FROM (
		       		SELECT workflow_run.id AS id, workflow_run_tag.tag AS tag, workflow_run_tag.value AS value, workflow_run.status AS status
		         	FROM workflow_run
		           	JOIN workflow_run_tag ON workflow_run.id = workflow_run_tag.workflow_run_id
					WHERE workflow_run.workflow_id = $1
					AND workflow_run.to_delete = false
		         	AND workflow_run_tag.tag = ANY(string_to_array($2, ',')::text[])
		       		ORDER BY workflow_run.id DESC
		    	) as wr
		    GROUP BY tag, value, status
		) as wrGrouped
	WHERE status = $3;
	`

	successIDs := []struct {
		ID string `db:"id"`
	}{}
	if _, err := db.Select(&successIDs, querySuccessIds, wf.ID, strings.Join(filteredPurgeTags, ","), sdk.StatusSuccess); err != nil {
		log.Error(ctx, "PurgeWorkflowRun> Unable to get workflow run in success for purge with workflow id %d, tags %v and history length %d : %s", wf.ID, wf.PurgeTags, wf.HistoryLength, err)
		return sdk.WithStack(err)
	}

	idsToUpdate := []string{}
	for _, idToUp := range ids {
		if idToUp.Ids != "" {
			// NEED TO SORT idToUp.Ids
			splittedIds := strings.Split(idToUp.Ids, ",")
			idsInt64 := make([]int64, len(splittedIds))
			for i, id := range splittedIds {
				nu, err := strconv.ParseInt(id, 10, 64)
				if err != nil {
					log.Error(ctx, "PurgeWorkflowRun> Cannot parse int64 %s: %v", id, err)
					return sdk.WithStack(err)
				}
				idsInt64[i] = nu
			}

			sort.Sort(byInt64Desc(idsInt64))
			idsStr := make([]string, 0, int64(len(idsInt64))-wf.HistoryLength)
			for _, id := range idsInt64[wf.HistoryLength:] {
				found := false
				strID := fmt.Sprintf("%d", id)
				for _, successID := range successIDs {
					if successID.ID == strID {
						found = true
						break
					}
				}
				// If id is the last success id don't add in the id's array to delete
				if !found {
					idsStr = append(idsStr, strID)
				}
			}
			idsToUpdate = append(idsToUpdate, idsStr...)
		}
	}

	//Don't mark as to_delete more than 100 workflow_runs
	if len(idsToUpdate) > 100 {
		idsToUpdate = idsToUpdate[:100]
	}

	if len(idsToUpdate) == 0 {
		return nil
	}

	queryUpdate := `UPDATE workflow_run SET to_delete = true WHERE workflow_run.id = ANY(string_to_array($1, ',')::bigint[])`
	if _, err := db.Exec(queryUpdate, strings.Join(idsToUpdate, ",")); err != nil {
		log.Error(ctx, "PurgeWorkflowRun> Unable to update workflow run for purge for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, err)
		return sdk.WithStack(err)
	}

	return nil
}

func purgeWorkflowRunWithoutTags(ctx context.Context, db gorp.SqlExecutor, wf sdk.Workflow) error {
	qLastSuccess := `
	SELECT id
		FROM (
			SELECT id, status
				FROM workflow_run
			WHERE workflow_id = $1
			ORDER BY id DESC
			OFFSET $2
		) as wr
	WHERE status = $3
	LIMIT 1`

	lastWfrID, errID := db.SelectInt(qLastSuccess, wf.ID, wf.HistoryLength, sdk.StatusSuccess)
	if errID != nil && errID != sql.ErrNoRows {
		log.Warning(ctx, "PurgeWorkflowRun> Unable to last success run for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, errID)
		return errID
	}

	qDelete := `
		WITH run_to_delete AS (
			SELECT workflow_run.id
			FROM workflow_run
			WHERE workflow_run.workflow_id = $1
			AND to_delete = false
			AND workflow_run.id < $2
			AND workflow_run.status <> $3
			AND workflow_run.status <> $4
			AND workflow_run.status <> $5
		)
		UPDATE workflow_run SET to_delete = true
		FROM run_to_delete
		WHERE workflow_run.id = run_to_delete.id
	`
	if _, err := db.Exec(qDelete, wf.ID, lastWfrID, sdk.StatusBuilding, sdk.StatusChecking, sdk.StatusWaiting); err != nil {
		log.Warning(ctx, "PurgeWorkflowRun> Unable to update workflow run for purge without tags for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, err)
		return err
	}

	return nil
}

func CountNotPendingWorkflowRunsByWorkflowID(db gorp.SqlExecutor, workflowID int64) (int64, error) {
	n, err := db.SelectInt("SELECT COUNT(id) FROM workflow_run WHERE workflow_id = $1 AND to_delete = false AND status <> $2", workflowID, sdk.StatusPending)
	return n, sdk.WithStack(err)
}

func CountWorkflowRunsMarkToDelete(ctx context.Context, db gorp.SqlExecutor, workflowRunsMarkToDelete *stats.Int64Measure) int64 {
	n, err := db.SelectInt("select count(1) from workflow_run where to_delete = true")
	if err != nil {
		log.Error(ctx, "countWorkflowRunsMarkToDelete> %v", err)
		return 0
	}
	log.Debug("CountWorkflowRunsMarkToDelete> %d workflow to delete", n)
	if workflowRunsMarkToDelete != nil {
		telemetry.Record(ctx, workflowRunsMarkToDelete, n)
	}
	return n
}

// syncNodeRuns load the workflow node runs for a workflow run
func syncNodeRuns(db gorp.SqlExecutor, wr *sdk.WorkflowRun, loadOpts LoadRunOptions) error {
	var testsField string
	if loadOpts.WithTests {
		testsField = nodeRunTestsField
	} else if loadOpts.WithLightTests {
		testsField = withLightNodeRunTestsField
	}

	wr.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	q := fmt.Sprintf("select %s %s from workflow_node_run where workflow_run_id = $1 ORDER BY workflow_node_run.sub_num DESC", nodeRunFields, testsField)
	dbNodeRuns := []NodeRun{}
	if _, err := db.Select(&dbNodeRuns, q, wr.ID); err != nil {
		if err != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load workflow nodes run")
		}
	}

	for _, n := range dbNodeRuns {
		wnr, err := fromDBNodeRun(n, loadOpts)
		if err != nil {
			return err
		}
		wnr.CanBeRun = CanBeRun(wr, wnr)
		if loadOpts.WithArtifacts {
			arts, errA := loadArtifactByNodeRunID(db, wnr.ID)
			if errA != nil {
				return sdk.WrapError(errA, "syncNodeRuns>Error loading artifacts for node run %d", wnr.ID)
			}
			wnr.Artifacts = arts
		}

		if loadOpts.WithStaticFiles {
			staticFiles, errS := loadStaticFilesByNodeRunID(db, wnr.ID)
			if errS != nil {
				return sdk.WrapError(errS, "syncNodeRuns>Error loading static files for node run %d", wnr.ID)
			}
			wnr.StaticFiles = staticFiles
		}

		if loadOpts.WithCoverage {
			cov, errCov := LoadCoverageReport(db, wnr.ID)
			if errCov != nil && !sdk.ErrorIs(errCov, sdk.ErrNotFound) {
				return sdk.WrapError(errCov, "syncNodeRuns> Error loading code coverage report for node run %d", wnr.ID)
			}
			wnr.Coverage = cov
		}
		var l = loadOpts.Language
		if l == "" {
			l = "en"
		}
		wnr.Translate(l)
		wr.WorkflowNodeRuns[wnr.WorkflowNodeID] = append(wr.WorkflowNodeRuns[wnr.WorkflowNodeID], *wnr)
	}

	for k := range wr.WorkflowNodeRuns {
		sort.Slice(wr.WorkflowNodeRuns[k], func(i, j int) bool {
			return wr.WorkflowNodeRuns[k][i].SubNumber > wr.WorkflowNodeRuns[k][j].SubNumber
		})
	}

	return nil
}

// stopRunsBlocked is useful to force stop all workflow that is running more than 24hrs
func stopRunsBlocked(ctx context.Context, db *gorp.DbMap) error {
	query := `SELECT workflow_run.id
		FROM workflow_run
		WHERE (workflow_run.status = $1 or workflow_run.status = $2 or workflow_run.status = $3)
		AND now() - workflow_run.last_execution > interval '1 day'
		LIMIT 30`
	ids := []struct {
		ID int64 `db:"id"`
	}{}

	if _, err := db.Select(&ids, query, sdk.StatusWaiting, sdk.StatusChecking, sdk.StatusBuilding); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WithStack(err)
	}

	if len(ids) == 0 {
		return nil
	}

	tx, errTx := db.Begin()
	if errTx != nil {
		return sdk.WrapError(errTx, "stopRunsBlocked>")
	}
	defer tx.Rollback() // nolint

	wfIds := make([]string, len(ids))
	for i := range wfIds {
		wfIds[i] = fmt.Sprintf("%d", ids[i].ID)
	}
	wfIdsJoined := strings.Join(wfIds, ",")
	args := []interface{}{sdk.StatusStopped, wfIdsJoined, sdk.StatusBuilding, sdk.StatusChecking, sdk.StatusWaiting}
	queryUpdateNodeJobRun := `DELETE FROM workflow_node_run_job
	WHERE (workflow_node_run_job.workflow_node_run_id IN (
			SELECT workflow_node_run.id
			FROM workflow_node_run
			WHERE (
					workflow_node_run.workflow_run_id = ANY(string_to_array($2, ',')::bigint[])
					AND (status = $3 OR status = $4 OR status = $5)
				)
				OR
				(workflow_node_run.status = $6 OR workflow_node_run.status = $1 OR workflow_node_run.status = $7)
		)
	)`
	argsNodeJobRun := append(args, sdk.StatusFail, sdk.StatusSuccess)
	if _, err := tx.Exec(queryUpdateNodeJobRun, argsNodeJobRun...); err != nil {
		return sdk.WrapError(err, "Unable to stop workflow node job run history")
	}

	queryUpdateNodeRun := `UPDATE workflow_node_run SET status = $1, done = now()
	WHERE workflow_run_id = ANY(string_to_array($2, ',')::bigint[])
	AND (status = $3 OR status = $4 OR status = $5)`
	if _, err := tx.Exec(queryUpdateNodeRun, args...); err != nil {
		return sdk.WrapError(err, "Unable to stop workflow node run history")
	}

	queryUpdateWf := `UPDATE workflow_run SET status = $1 WHERE id = ANY(string_to_array($2, ',')::bigint[])`
	if _, err := tx.Exec(queryUpdateWf, sdk.StatusStopped, wfIdsJoined); err != nil {
		return sdk.WrapError(err, "Unable to stop workflow run history")
	}

	resp := []struct {
		ID     int64          `db:"id"`
		Status string         `db:"status"`
		Stages sql.NullString `db:"stages"`
	}{}

	querySelectNodeRuns := `
	SELECT workflow_node_run.id, workflow_node_run.status, workflow_node_run.stages
		FROM workflow_node_run
		WHERE workflow_node_run.workflow_run_id = ANY(string_to_array($1, ',')::bigint[])
	`
	if _, err := tx.Select(&resp, querySelectNodeRuns, wfIdsJoined); err != nil {
		return sdk.WrapError(err, "cannot get workflow node run infos")
	}

	now := time.Now()
	for i := range resp {
		nr := sdk.WorkflowNodeRun{
			ID:     resp[i].ID,
			Status: resp[i].Status,
		}
		if resp[i].Stages.Valid {
			if err := json.Unmarshal([]byte(resp[i].Stages.String), &nr.Stages); err != nil {
				return sdk.WrapError(err, "cannot unmarshal stages")
			}
		}

		stopWorkflowNodeRunStages(ctx, db, &nr)
		if !sdk.StatusIsTerminated(resp[i].Status) {
			nr.Status = sdk.StatusStopped
			nr.Done = now
		}

		if err := updateNodeRunStatusAndStage(tx, &nr); err != nil {
			return sdk.WrapError(err, "cannot update node runs stages")
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "Unable to commit transaction")
	}
	return nil
}
