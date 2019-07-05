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
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
workflow_run.to_delete
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
	_, end := observability.Span(ctx, "workflow.UpdateWorkflowRun")
	defer end()

	wr.LastModified = time.Now()
	for _, info := range wr.Infos {
		if info.IsError && info.SubNumber == wr.LastSubNumber {
			wr.Status = string(sdk.StatusFail)
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
	w, errw := json.Marshal(r.Workflow)
	if errw != nil {
		return sdk.WrapError(errw, "Unable to marshal workflow")
	}

	jtr, erri := json.Marshal(r.JoinTriggersRun)
	if erri != nil {
		return sdk.WrapError(erri, "Unable to marshal JoinTriggersRun")
	}

	i, erri := json.Marshal(r.Infos)
	if erri != nil {
		return sdk.WrapError(erri, "Unable to marshal infos")
	}

	h, errh := json.Marshal(r.Header)
	if errh != nil {
		return sdk.WrapError(erri, "Unable to marshal header")
	}

	if _, err := db.Exec("update workflow_run set workflow = $3, infos = $2, join_triggers_run = $4, header = $5 where id = $1", r.ID, i, w, jtr, h); err != nil {
		return sdk.WrapError(err, "Unable to store marshalled infos")
	}

	if err := updateTags(db, r); err != nil {
		return sdk.WrapError(err, "Unable to store tags")
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
		H sql.NullString `db:"header"`
		O sql.NullString `db:"outgoing_hook_runs"`
	}{}

	if err := db.SelectOne(&res, "select workflow, infos, join_triggers_run, header, outgoing_hook_runs from workflow_run where id = $1", r.ID); err != nil {
		return sdk.WrapError(err, "Unable to load marshalled workflow")
	}

	w := sdk.Workflow{}
	if err := gorpmapping.JSONNullString(res.W, &w); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal workflow")
	}
	r.Workflow = w

	i := []sdk.WorkflowRunInfo{}
	if err := gorpmapping.JSONNullString(res.I, &i); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal infos")
	}
	r.Infos = i

	j := map[int64]sdk.WorkflowNodeTriggerRun{}
	if err := gorpmapping.JSONNullString(res.J, &j); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal join_triggers_run")
	}
	r.JoinTriggersRun = j

	h := sdk.WorkflowRunHeaders{}
	if err := gorpmapping.JSONNullString(res.H, &h); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal header")
	}
	r.Header = h

	return nil
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

// LockRun locks a workflow run
func LockRun(db gorp.SqlExecutor, id int64) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`SELECT %s
	FROM workflow_run
	WHERE id = $1 FOR UPDATE SKIP LOCKED`, wfRunfields)
	wr, err := loadRun(db, LoadRunOptions{}, query, id)
	if err == sdk.ErrWorkflowNotFound {
		err = sdk.ErrLocked
	}
	return wr, sdk.WithStack(err)
}

// LoadRunIDsWithOldModel loads all ids for run that use old workflow model
func LoadRunIDsWithOldModel(db gorp.SqlExecutor) ([]int64, error) {
	query := "SELECT id FROM workflow_run WHERE workflow->'workflow_data' IS NULL LIMIT 100"
	var ids []int64
	_, err := db.Select(&ids, query)
	return ids, sdk.WithStack(err)
}

// LoadRun returns a specific run
func LoadRun(ctx context.Context, db gorp.SqlExecutor, projectkey, workflowname string, number int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	_, end := observability.Span(ctx, "workflow.LoadRun",
		observability.Tag(observability.TagProjectKey, projectkey),
		observability.Tag(observability.TagWorkflow, workflowname),
		observability.Tag(observability.TagWorkflowRun, number),
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

// LoadRunByNodeRunID returns a specific run
func LoadRunByNodeRunID(db gorp.SqlExecutor, nodeRunID int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	join workflow_node_run on workflow_node_run.workflow_run_id = workflow_run.id
	where workflow_node_run.id = $1`, wfRunfields)
	return loadRun(db, loadOpts, query, nodeRunID)
}

// LoadRunByID loads run by ID
func LoadRunByID(db gorp.SqlExecutor, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	where workflow_run.id = $1`, wfRunfields)
	return loadRun(db, loadOpts, query, id)
}

// LoadAndLockRunByJobID loads a run by a job id
func LoadAndLockRunByJobID(db gorp.SqlExecutor, id int64, loadOpts LoadRunOptions) (*sdk.WorkflowRun, error) {
	query := fmt.Sprintf(`select %s
	from workflow_run
	join workflow_node_run on workflow_run.id = workflow_node_run.workflow_run_id
	join workflow_node_run_job on workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
	where workflow_node_run_job.id = $1 for update`, wfRunfields)
	return loadRun(db, loadOpts, query, id)
}

//LoadRuns loads all runs
//It retuns runs, offset, limit count and an error
func LoadRuns(db gorp.SqlExecutor, projectkey, workflowname string, offset, limit int, tagFilter map[string]string) ([]sdk.WorkflowRun, int, int, int, error) {
	var args = []interface{}{projectkey}
	queryCount := `select count(workflow_run.id)
				from workflow_run
				join project on workflow_run.project_id = project.id
				join workflow on workflow_run.workflow_id = workflow.id
				where project.projectkey = $1 AND workflow_run.to_delete = false`

	if workflowname != "" {
		args = []interface{}{projectkey, workflowname}
		queryCount = `select count(workflow_run.id)
					from workflow_run
					join project on workflow_run.project_id = project.id
					join workflow on workflow_run.workflow_id = workflow.id
					where project.projectkey = $1
					and workflow.name = $2
					AND workflow_run.to_delete = false`
	}

	count, errc := db.SelectInt(queryCount, args...)
	if errc != nil {
		return nil, 0, 0, 0, sdk.WrapError(errc, "Unable to load runs")
	}
	if count == 0 {
		return nil, 0, 0, 0, nil
	}

	args = []interface{}{projectkey, limit, offset}
	query := fmt.Sprintf(`select %s
	from workflow_run
	join project on workflow_run.project_id = project.id
	join workflow on workflow_run.workflow_id = workflow.id
	where project.projectkey = $1 AND workflow_run.to_delete = false
	order by workflow_run.start desc
	limit $2 offset $3`, wfRunfields)

	if workflowname != "" {
		args = []interface{}{projectkey, workflowname, limit, offset}
		query = fmt.Sprintf(`select %s
			from workflow_run
			join project on workflow_run.project_id = project.id
			join workflow on workflow_run.workflow_id = workflow.id
			where project.projectkey = $1
			and workflow.name = $2
			AND workflow_run.to_delete = false
			order by workflow_run.start desc
			limit $3 offset $4`, wfRunfields)
	}

	if len(tagFilter) > 0 {
		// Posgres operator: '<@' means 'is contained by' eg. 'ARRAY[2,7] <@ ARRAY[1,7,4,2,6]' ==> returns true
		query = fmt.Sprintf(`select %s
		from workflow_run
		join project on workflow_run.project_id = project.id
		join workflow on workflow_run.workflow_id = workflow.id
		join (
			select workflow_run_id, string_agg(all_tags, ',') as tags
			from (
				select workflow_run_id, tag || '=' || value "all_tags"
				from workflow_run_tag
				order by tag
			) as all_wr_tags
			group by workflow_run_id
		) as tags on workflow_run.id = tags.workflow_run_id
		where project.projectkey = $1
		and workflow.name = $2
		AND workflow_run.to_delete = false
		and string_to_array($5, ',') <@ string_to_array(tags.tags, ',')
		order by workflow_run.start desc
		limit $3 offset $4`, wfRunfields)

		var tags []string
		for k, v := range tagFilter {
			tags = append(tags, k+"="+v)
		}

		log.Debug("tags=%v", tags)

		args = append(args, strings.Join(tags, ","))
	}

	runs := []Run{}
	if _, err := db.Select(&runs, query, args...); err != nil {
		return nil, 0, 0, 0, sdk.WrapError(errc, "Unable to load runs")
	}
	wruns := make([]sdk.WorkflowRun, len(runs))
	for i := range runs {
		wr := sdk.WorkflowRun(runs[i])
		if err := loadRunTags(db, &wr); err != nil {
			return nil, 0, 0, 0, sdk.WrapError(err, "Unable to load tags")
		}

		wruns[i] = wr
	}

	return wruns, offset, limit, int(count), nil
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

func loadRunTags(db gorp.SqlExecutor, run *sdk.WorkflowRun) error {
	dbRunTags := []RunTag{}
	if _, err := db.Select(&dbRunTags, "SELECT * from workflow_run_tag WHERE workflow_run_id=$1", run.ID); err != nil {
		return sdk.WithStack(err)
	}

	run.Tags = make([]sdk.WorkflowRunTag, len(dbRunTags))
	for i := range dbRunTags {
		run.Tags[i] = sdk.WorkflowRunTag(dbRunTags[i])
	}
	return nil
}

func loadRun(db gorp.SqlExecutor, loadOpts LoadRunOptions, query string, args ...interface{}) (*sdk.WorkflowRun, error) {
	runDB := &Run{}
	if err := db.SelectOne(runDB, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNotFound
		}
		return nil, sdk.WrapError(err, "Unable to load workflow run. query:%s args:%v", query, args)
	}
	wr := sdk.WorkflowRun(*runDB)
	if !loadOpts.WithDeleted && wr.ToDelete {
		return nil, sdk.WithStack(sdk.ErrWorkflowNotFound)
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

// CreateRun creates a new workflow run and insert it
func CreateRun(db *gorp.DbMap, wf *sdk.Workflow, opts *sdk.WorkflowRunPostHandlerOption, u *sdk.User) (*sdk.WorkflowRun, error) {
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
		Status:        sdk.StatusPending.String(),
		LastExecution: time.Now(),
		Tags:          make([]sdk.WorkflowRunTag, 0),
		Workflow:      sdk.Workflow{Name: wf.Name},
	}

	if opts != nil && opts.Hook != nil {
		if trigg, ok := opts.Hook.Payload["cds.triggered_by.username"]; ok {
			wr.Tag(tagTriggeredBy, trigg)
		} else {
			wr.Tag(tagTriggeredBy, "cds.hook")
		}
	} else {
		wr.Tag(tagTriggeredBy, u.Username)
	}

	tags := wf.Metadata["default_tags"]
	var payload map[string]string
	if opts != nil && opts.Hook != nil {
		payload = opts.Hook.Payload
	}
	if opts != nil && opts.Manual != nil {
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
func PurgeAllWorkflowRunsByWorkflowID(db gorp.SqlExecutor, id int64) (int, error) {
	query := "UPDATE workflow_run SET to_delete = true WHERE workflow_id = $1"
	res, err := db.Exec(query, id)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	n, _ := res.RowsAffected() // nolint
	log.Info("PurgeAllWorkflowRunsByWorkflowID> will delete %d workflow runs for workflow %d", n, id)
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
func PurgeWorkflowRun(ctx context.Context, db gorp.SqlExecutor, wf sdk.Workflow, workflowRunsMarkToDelete *stats.Int64Measure) error {
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

		lastWfrID, errID := db.SelectInt(qLastSuccess, wf.ID, wf.HistoryLength, sdk.StatusSuccess.String())
		if errID != nil && errID != sql.ErrNoRows {
			log.Warning("PurgeWorkflowRun> Unable to last success run for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, errID)
			return errID
		}

		qDelete := `
			UPDATE workflow_run SET to_delete = true
			WHERE workflow_run.id IN (
				SELECT workflow_run.id
				FROM workflow_run
				WHERE workflow_run.workflow_id = $1
				AND workflow_run.id < $2
				AND workflow_run.status <> $3
				AND workflow_run.status <> $4
				AND workflow_run.status <> $5
				LIMIT 100
			)
		`
		res, err := db.Exec(qDelete, wf.ID, lastWfrID, sdk.StatusBuilding.String(), sdk.StatusChecking.String(), sdk.StatusWaiting.String())
		if err != nil {
			log.Warning("PurgeWorkflowRun> Unable to update workflow run for purge without tags for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, err)
			return err
		}

		n, _ := res.RowsAffected()
		if workflowRunsMarkToDelete != nil {
			observability.Record(ctx, workflowRunsMarkToDelete, n)
		}

		return nil
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
				ORDER BY workflow_run.id DESC
			) as wr
		GROUP BY tag, value HAVING COUNT(id) > $3
	`

	_, errS := db.Select(
		&ids,
		queryGetIds,
		wf.ID,
		strings.Join(filteredPurgeTags, ","),
		wf.HistoryLength,
		sdk.StatusWaiting.String(),
		sdk.StatusBuilding.String(),
		sdk.StatusChecking.String(),
		sdk.StatusPending.String(),
	)
	if errS != nil {
		log.Warning("PurgeWorkflowRun> Unable to get workflow run for purge with workflow id %d, tags %v and history length %d : %s", wf.ID, wf.PurgeTags, wf.HistoryLength, errS)
		return errS
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
	if _, errS := db.Select(&successIDs, querySuccessIds, wf.ID, strings.Join(filteredPurgeTags, ","), sdk.StatusSuccess.String()); errS != nil {
		log.Warning("PurgeWorkflowRun> Unable to get workflow run in success for purge with workflow id %d, tags %v and history length %d : %s", wf.ID, wf.PurgeTags, wf.HistoryLength, errS)
		return errS
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
					log.Error("PurgeWorkflowRun> Cannot parse int64 %s: %v", id, err)
					return err
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
	res, err := db.Exec(queryUpdate, strings.Join(idsToUpdate, ","))
	if err != nil {
		log.Warning("PurgeWorkflowRun> Unable to update workflow run for purge for workflow id %d and history length %d : %s", wf.ID, wf.HistoryLength, err)
		return err
	}

	n, _ := res.RowsAffected()
	if workflowRunsMarkToDelete != nil {
		observability.Record(ctx, workflowRunsMarkToDelete, n)
	}
	return nil
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
func stopRunsBlocked(db *gorp.DbMap) error {
	query := `SELECT workflow_run.id
		FROM workflow_run
		WHERE (workflow_run.status = $1 or workflow_run.status = $2 or workflow_run.status = $3)
		AND now() - workflow_run.last_execution > interval '1 day'
		LIMIT 30`
	ids := []struct {
		ID int64 `db:"id"`
	}{}

	if _, err := db.Select(&ids, query, sdk.StatusWaiting.String(), sdk.StatusChecking.String(), sdk.StatusBuilding.String()); err != nil {
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
	args := []interface{}{sdk.StatusStopped.String(), wfIdsJoined, sdk.StatusBuilding.String(), sdk.StatusChecking.String(), sdk.StatusWaiting.String()}
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
	argsNodeJobRun := append(args, sdk.StatusFail.String(), sdk.StatusSuccess.String())
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
	if _, err := tx.Exec(queryUpdateWf, sdk.StatusStopped.String(), wfIdsJoined); err != nil {
		return sdk.WrapError(err, "Unable to stop workflow run history")
	}

	resp := []struct {
		ID     int64  `db:"id"`
		Status string `db:"status"`
		Stages string `db:"stages"`
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
		if err := json.Unmarshal([]byte(resp[i].Stages), &nr.Stages); err != nil {
			return sdk.WrapError(err, "cannot unmarshal stages")
		}

		stopWorkflowNodeRunStages(db, &nr)
		if !sdk.StatusIsTerminated(resp[i].Status) {
			nr.Status = sdk.StatusStopped.String()
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
