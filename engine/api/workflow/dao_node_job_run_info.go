package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//loadNodeRunJobInfo load infos (workflow_node_run_job_infos) for a job (workflow_node_run_job)
func loadNodeRunJobInfo(db gorp.SqlExecutor, jobIDs []int64) ([]sdk.SpawnInfo, error) {
	ids := make([]string, len(jobIDs))
	for i := range ids {
		ids[i] = fmt.Sprintf("%d", jobIDs[i])
	}
	idsJoined := strings.Join(ids, ",")
	res := []struct {
		Bytes                sql.NullString `db:"spawninfos"`
		WorkflowNodeJobRunID int64          `db:"workflow_node_job_run_id"`
	}{}
	query := "SELECT workflow_node_run_job_id, spawninfos FROM workflow_node_run_job_info WHERE workflow_node_run_job_id = ANY(string_to_array($1, ',')::bigint[])"
	if _, err := db.Select(&res, query, idsJoined); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "cannot QueryRow")
	}

	spawnInfos := []sdk.SpawnInfo{}
	for i := range res {
		spInfos := []sdk.SpawnInfo{}
		if err := gorpmapping.JSONNullString(res[i].Bytes, &spInfos); err != nil {
			// should never append, but log error
			log.Warning("wrong spawnInfos format: res: %v for ids: %v", res[i].Bytes, ids[i])
			continue
		}
		for i := range spInfos {
			spInfos[i].WorkflowNodeJobRunID = res[i].WorkflowNodeJobRunID
		}
		spawnInfos = append(spawnInfos, spInfos...)
	}
	return spawnInfos, nil
}

// insertNodeRunJobInfo inserts spawninfos for a Workflow Node Job Run. This is
// a temporary data, as workflow_node_job_run table. After the end of the Job,
// swpawninfos values will be in WorfklowRun table in stages column
func insertNodeRunJobInfo(db gorp.SqlExecutor, info *sdk.WorkflowNodeJobRunInfo) error {
	spawnJSON, errJ := json.Marshal(info.SpawnInfos)
	if errJ != nil {
		return sdk.WrapError(errJ, "insertNodeRunJobInfo> cannot Marshal")
	}

	query := "insert into workflow_node_run_job_info (workflow_node_run_id, workflow_node_run_job_id, spawninfos, created) values ($1, $2, $3, $4)"
	if n, err := db.Exec(query, info.WorkflowNodeRunID, info.WorkflowNodeJobRunID, spawnJSON, time.Now()); err != nil {
		return sdk.WrapError(err, "err while inserting spawninfos into workflow_node_run_job_info")
	} else if n, _ := n.RowsAffected(); n == 0 {
		return fmt.Errorf("insertNodeRunJobInfo> Unable to insert into workflow_node_run_job_info id = %d", info.WorkflowNodeJobRunID)
	}

	log.Debug("insertNodeRunJobInfo> on node run: %v (%d)", info.SpawnInfos, info.WorkflowNodeJobRunID)
	return nil
}
