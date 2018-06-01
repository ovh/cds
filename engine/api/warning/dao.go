package warning

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"fmt"
	"github.com/mitchellh/hashstructure"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
)

func removeProjectWarning(db gorp.SqlExecutor, warningType string, element string, key string) error {
	result, err := db.Exec("DELETE FROM warning where type = $1 and element = $2 and project_key = $3", warningType, element, key)
	if err != nil {
		return sdk.WrapError(err, "removeProjectWarning> Unable to remove warning %s/%s", warningType, element)
	}
	nb, errR := result.RowsAffected()
	if errR != nil {
		return sdk.WrapError(errR, "removeProjectWarning> Unable to read result")
	}
	if nb == 1 {
		event.PublishDeleteWarning(warningType, element, key, "", "", "", "")
	}
	return err
}

func Insert(db gorp.SqlExecutor, w sdk.WarningV2) error {
	h, err := hashstructure.Hash(w, nil)
	if err != nil {
		return sdk.WrapError(err, "warning.Insert> Unable to calculate hash")
	}
	w.Hash = fmt.Sprintf("%v", h)
	warn := warning(w)
	if err := db.Insert(&warn); err != nil {
		return sdk.WrapError(err, "warning.Insert> Unable to insert warning")
	}
	w = sdk.WarningV2(warn)
	event.PublishAddWarning(w)
	return nil
}

// PostInsert is a db hook
func (w *warning) PostInsert(db gorp.SqlExecutor) error {
	return w.PostUpdate(db)
}

// PostInsert is a db hook
func (w *warning) PostUpdate(db gorp.SqlExecutor) error {
	msgs, errM := gorpmapping.JSONToNullString(w.MessageParams)
	if errM != nil {
		return sdk.WrapError(errM, "warning.PostUpdate: unable to stringify Messageparams")
	}
	query := `
		UPDATE warning SET message_params = $1 WHERE id = $2
	`
	if _, err := db.Exec(query, msgs, w.ID); err != nil {
		return sdk.WrapError(err, "warning.PostUpdate: unable to update warning")
	}
	return nil
}

// PostGet is a db hook
func (w *warning) PostGet(db gorp.SqlExecutor) error {
	var fields = struct {
		MessageParams sql.NullString `db:"message_params"`
	}{}

	if err := db.QueryRow("select message_params from warning where id = $1", w.ID).Scan(&fields.MessageParams); err != nil {
		return err
	}

	if err := gorpmapping.JSONNullString(fields.MessageParams, &w.MessageParams); err != nil {
		return err
	}
	return nil
}

// GetByProject Get all warnings for the given project
func GetByProject(db gorp.SqlExecutor, key string) ([]sdk.WarningV2, error) {
	query := `
		SELECT * FROM warning WHERE project_key = $1
	`
	var ws []warning
	if _, err := db.Select(&ws, query, key); err != nil {
		return nil, sdk.WrapError(err, "warning.GetByProject> Unable to list warnings for project %s", key)
	}

	warnings := make([]sdk.WarningV2, len(ws))
	for i, w := range ws {
		if err := w.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "warning.GetByProject> Unable to post get warnings")
		}
		warnings[i] = sdk.WarningV2(w)
	}

	return warnings, nil
}
