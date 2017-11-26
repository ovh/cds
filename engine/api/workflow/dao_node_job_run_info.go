package workflow

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//loadNodeRunJobInfo load infos (workflow_node_run_job_infos) for a job (workflow_node_run_job)
func loadNodeRunJobInfo(db gorp.SqlExecutor, jobID int64) ([]sdk.SpawnInfo, error) {
	query := "SELECT spawninfos FROM workflow_node_run_job_info WHERE workflow_node_run_job_id = $1"
	rows, err := db.Query(query, jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadNodeRunJobInfo> cannot QueryRow")
	}
	defer rows.Close()

	spawnInfos := []sdk.SpawnInfo{}
	for rows.Next() {
		infos := []sdk.SpawnInfo{}
		var bts []byte
		if err := rows.Scan(&bts); err != nil {
			return nil, sdk.WrapError(err, "loadNodeRunJobInfo> cannot Scan")
		}
		if err := json.Unmarshal(bts, &infos); err != nil {
			return nil, sdk.WrapError(err, "loadNodeRunJobInfo> cannot Unmarshal")
		}
		spawnInfos = append(spawnInfos, infos...)
	}
	return spawnInfos, nil
}

// insertNodeRunJobInfo inserts spawninfos for a Workflow Node Job Run. This is
// a temporary data, as workflow_node_job_run table. After the end of the Job,
// swpawninfos values will be in WorfklowRun table in stages column
func insertNodeRunJobInfo(db gorp.SqlExecutor, info *sdk.WorkflowNodeJobRunInfo) error {
	log.Warning("insertNodeRunJobInfo > insert info: %+v", info)
	spawnJSON, errJ := json.Marshal(info.SpawnInfos)
	if errJ != nil {
		return errJ
	}

	query := "insert into workflow_node_run_job_info (workflow_node_run_id, workflow_node_run_job_id, spawninfos, created) values ($1, $2, $3, $4)"
	if n, err := db.Exec(query, info.WorkflowNodeRunID, info.WorkflowNodeJobRunID, spawnJSON, time.Now()); err != nil {
		return err
	} else if n, _ := n.RowsAffected(); n == 0 {
		return sdk.WrapError(err, "insertNodeRunJobInfo> Unable to update workflow_node_run_job_info id = %d", info.WorkflowNodeJobRunID)
	}

	log.Debug("insertNodeRunJobInfo> on node run: %d (%d)", info.ID, info.WorkflowNodeJobRunID)
	return nil
}
