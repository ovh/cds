package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
)

// InsertActionBuild Insert new action build
func InsertActionBuild(db database.QueryExecuter, b *sdk.ActionBuild) error {
	query := `INSERT INTO action_build (pipeline_action_id, args, status, pipeline_build_id, queued, start, done) VALUES($1, $2, $3, $4, $5, $5, $6) RETURNING id`

	if b.PipelineActionID == 0 {
		return fmt.Errorf("invalid pipeline action ID (0)")
	}

	if b.PipelineBuildID == 0 {
		return fmt.Errorf("invalid pipeline build ID (0)")
	}

	argsJSON, err := json.Marshal(b.Args)
	if err != nil {
		return err
	}

	if b.Status == "" {
		b.Status = sdk.StatusWaiting
	}

	//Set action_build.done to null is not set
	var done interface{}
	if b.Done.IsZero() {
		done = sql.NullString{
			String: "",
			Valid:  false,
		}
	} else {
		done = b.Done
	}

	err = db.QueryRow(query, b.PipelineActionID, string(argsJSON), b.Status.String(), b.PipelineBuildID, time.Now(), done).Scan(&b.ID)
	if err != nil {
		return err
	}

	event.PublishActionBuild(b, sdk.CreateEvent)
	return nil
}

func loadPipelineActionArguments(db database.Querier, pipelineActionID int64) ([]sdk.Parameter, error) {
	query := `SELECT args FROM pipeline_action WHERE id = $1`

	var argsJSON sql.NullString
	if err := db.QueryRow(query, pipelineActionID).Scan(&argsJSON); err != nil {
		return nil, err
	}

	var parameters []sdk.Parameter
	if argsJSON.Valid {
		if err := json.Unmarshal([]byte(argsJSON.String), &parameters); err != nil {
			return nil, err
		}
	}

	return parameters, nil
}
