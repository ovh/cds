package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//loadNodeRunJobInfo load infos (workflow_node_run_job_infos) for a job (workflow_node_run_job)
func loadNodeRunJobInfo(db gorp.SqlExecutor, jobID int64) ([]sdk.SpawnInfo, error) {
	res := []struct {
		Bytes sql.NullString `db:"spawninfos"`
	}{}
	query := "SELECT spawninfos FROM workflow_node_run_job_info WHERE workflow_node_run_job_id = $1"
	if _, err := db.Select(&res, query, jobID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadNodeRunJobInfo> cannot QueryRow")
	}

	spawnInfos := []sdk.SpawnInfo{}
	for _, r := range res {
		v := []sdk.SpawnInfo{}
		gorpmapping.JSONNullString(r.Bytes, &v)
		spawnInfos = append(spawnInfos, v...)
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
		return sdk.WrapError(errJ, "insertNodeRunJobInfo> cannot Marshal")
	}

	query := "insert into workflow_node_run_job_info (workflow_node_run_id, workflow_node_run_job_id, spawninfos, created) values ($1, $2, $3, $4)"
	if n, err := db.Exec(query, info.WorkflowNodeRunID, info.WorkflowNodeJobRunID, spawnJSON, time.Now()); err != nil {
		return sdk.WrapError(err, "insertNodeRunJobInfo> err while inserting spawninfos into workflow_node_run_job_info")
	} else if n, _ := n.RowsAffected(); n == 0 {
		return fmt.Errorf("insertNodeRunJobInfo> Unable to update workflow_node_run_job_info id = %d", info.WorkflowNodeJobRunID)
	}

	log.Debug("insertNodeRunJobInfo> on node run: %d (%d)", info.ID, info.WorkflowNodeJobRunID)
	return nil
}
