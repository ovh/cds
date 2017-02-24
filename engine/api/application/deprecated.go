package application

import (
	"encoding/json"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// UpdatePipelineApplication Update arguments passed to pipeline
func UpdatePipelineApplication(db gorp.SqlExecutor, app *sdk.Application, pipelineID int64, params []sdk.Parameter, u *sdk.User) error {
	data, err := json.Marshal(params)
	if err != nil {
		log.Warning("UpdatePipelineApplication> Cannot marshal parameters:  %s \n", err)
		return fmt.Errorf("UpdatePipelineApplication>Cannot marshal parameters:  %s", err)
	}
	return UpdatePipelineApplicationString(db, app, pipelineID, string(data), u)
}

// UpdatePipelineApplicationString Update application pipeline parameters
func UpdatePipelineApplicationString(db gorp.SqlExecutor, app *sdk.Application, pipelineID int64, data string, u *sdk.User) error {
	query := `
		UPDATE application_pipeline SET 
		args = $1,
		last_modified = current_timestamp
		WHERE application_id=$2 AND pipeline_id=$3
		`

	// TODO: cipher args here
	_, err := db.Exec(query, data, app.ID, pipelineID)
	if err != nil {
		return err
	}

	return UpdateLastModified(db, app, u)
}
