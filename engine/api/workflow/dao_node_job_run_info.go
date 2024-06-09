package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadNodeRunJobInfo load infos (workflow_node_run_job_infos) for a job (workflow_node_run_job)
func LoadNodeRunJobInfo(ctx context.Context, db gorp.SqlExecutor, nodeRunID int64, jobID int64) ([]sdk.SpawnInfo, error) {
	res := []struct {
		Bytes sql.NullString `db:"spawninfos"`
	}{}
	query := "SELECT spawninfos FROM workflow_node_run_job_info WHERE workflow_node_run_id = $1 AND workflow_node_run_job_id = $2"
	if _, err := db.Select(&res, query, nodeRunID, jobID); err != nil {
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
			log.Warn(ctx, "wrong spawnInfos format: res: %v for id: %v err: %v", res[i].Bytes, jobID, err)
			continue
		}
		spawnInfos = append(spawnInfos, spInfos...)
	}
	// sort here and not in sql, as it's could be a json array in sql value
	sort.Slice(spawnInfos, func(i, j int) bool {
		return spawnInfos[i].APITime.Before(spawnInfos[j].APITime)
	})
	return spawnInfos, nil
}

// insertNodeRunJobInfo inserts spawninfos for a Workflow Node Job Run. This is
// a temporary data, as workflow_node_job_run table. After the end of the Job,
// swpawninfos values will be in WorkflowRun table in stages column
func insertNodeRunJobInfo(db gorp.SqlExecutor, info *sdk.WorkflowNodeJobRunInfo) error {
	spawnJSON, err := json.Marshal(info.SpawnInfos)
	if err != nil {
		return sdk.WithStack(err)
	}

	query := "insert into workflow_node_run_job_info (workflow_node_run_id, workflow_node_run_job_id, spawninfos, created) values ($1, $2, $3, $4)"
	if n, err := db.Exec(query, info.WorkflowNodeRunID, info.WorkflowNodeJobRunID, spawnJSON, time.Now()); err != nil {
		return sdk.WrapError(err, "err while inserting spawninfos into workflow_node_run_job_info")
	} else if n, _ := n.RowsAffected(); n == 0 {
		return sdk.WithStack(fmt.Errorf("unable to insert into workflow_node_run_job_info id = %d", info.WorkflowNodeJobRunID))
	}

	log.Debug(context.TODO(), "insertNodeRunJobInfo> on node run: %d (job run:%d)", info.WorkflowNodeRunID, info.WorkflowNodeJobRunID)
	return nil
}
