package pipeline

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getPipelineActionsByStageID(db gorp.SqlExecutor, stageID int64) ([]pipelineAction, error) {
	var pas []pipelineAction

	query := gorpmapping.NewQuery(
		"SELECT * FROM pipeline_action WHERE pipeline_stage_id = $1",
	).Args(stageID)
	if err := gorpmapping.GetAll(db, query, &pas); err != nil {
		return nil, sdk.WrapError(err, "cannot get pipeline action links for stage id %d", stageID)
	}

	return pas, nil
}

func deletePipelineActionsByIDs(db gorp.SqlExecutor, ids []int64) error {
	_, err := db.Exec(
		"DELETE FROM pipeline_action WHERE id = ANY(string_to_array($1, ',')::int[])",
		gorpmapping.IDsToQueryString(ids),
	)
	return sdk.WithStack(err)
}
