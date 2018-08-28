package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadActionParameters retrieves given action requirements in database
func LoadActionParameters(db gorp.SqlExecutor, actionID int64) ([]sdk.Parameter, error) {
	var req []sdk.Parameter

	query := `SELECT name, type, value, description, advanced FROM action_parameter WHERE action_id = $1 ORDER BY name`
	rows, err := db.Query(query, actionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r sdk.Parameter
		var t, val string
		var d []byte
		var advanced bool
		if err := rows.Scan(&r.Name, &t, &val, &d, &advanced); err != nil {
			return nil, err
		}
		if d != nil {
			r.Description = string(d)
		}
		r.Type = t
		r.Value = val
		r.Advanced = advanced

		req = append(req, r)
	}

	return req, nil
}

// InsertActionParameter inserts given requirement in database
func InsertActionParameter(db gorp.SqlExecutor, actionID int64, r sdk.Parameter) error {
	if string(r.Type) == string(sdk.SecretVariable) {
		return sdk.ErrNoDirectSecretUse
	}

	query := `INSERT INTO action_parameter (action_id, name, type, value, description, advanced) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(query, actionID, r.Name, string(r.Type), r.Value, r.Description, r.Advanced)
	if err != nil {
		log.Warning("InsertActionParameter> Error while insert action parameter: %s while insert actionID(%d), r.Name(%s), r.Type(%s), r.Description(%s), r.Advanced(%t)",
			err.Error(), actionID, r.Name, string(r.Type), r.Description, r.Advanced)
		return err
	}

	return nil
}

// DeleteActionParameters deletes all requirements related to given action
func DeleteActionParameters(db gorp.SqlExecutor, actionID int64) error {
	query := `DELETE FROM action_parameter WHERE action_id = $1`

	if _, err := db.Exec(query, actionID); err != nil {
		return err
	}

	return nil
}
