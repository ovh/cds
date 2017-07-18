package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAllBinaryRequirements retrieves all requirements in database
// Used by worker to automatically declare most capabilities
func LoadAllBinaryRequirements(db gorp.SqlExecutor) ([]sdk.Requirement, error) {
	var req []sdk.Requirement

	query := `SELECT distinct value FROM action_requirement where type = 'binary'`
	rows, errQ := db.Query(query)
	if errQ != nil {
		return nil, errQ
	}
	defer rows.Close()

	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}

		var r = sdk.Requirement{
			Name:  value,
			Type:  sdk.BinaryRequirement,
			Value: value,
		}

		req = append(req, r)
	}

	return req, nil
}

// LoadActionRequirements retrieves given action requirements in database
func LoadActionRequirements(db gorp.SqlExecutor, actionID int64) ([]sdk.Requirement, error) {
	var req []sdk.Requirement

	query := `SELECT name, type, value FROM action_requirement WHERE action_id = $1 ORDER BY name`
	rows, errQ := db.Query(query, actionID)
	if errQ != nil {
		return nil, errQ
	}
	defer rows.Close()

	for rows.Next() {
		var r sdk.Requirement
		if err := rows.Scan(&r.Name, &r.Type, &r.Value); err != nil {
			return nil, err
		}
		req = append(req, r)
	}

	return req, nil
}

// InsertActionRequirement inserts given requirement in database
func InsertActionRequirement(db gorp.SqlExecutor, actionID int64, r sdk.Requirement) error {
	if r.Name == "" || r.Type == "" || r.Value == "" {
		return sdk.ErrInvalidJobRequirement
	}
	query := `INSERT INTO action_requirement (action_id, name, type, value) VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(query, actionID, r.Name, string(r.Type), r.Value)
	return err
}

// DeleteActionRequirements deletes all requirements related to given action
func DeleteActionRequirements(db gorp.SqlExecutor, actionID int64) error {
	query := `DELETE FROM action_requirement WHERE action_id = $1`
	_, err := db.Exec(query, actionID)
	return err
}

// UpdateAllRequirements updates all action_requirement.value given a value and a type. It returns action ID
func UpdateAllRequirements(db gorp.SqlExecutor, oldValue, newValue, reqType string) ([]int64, error) {
	rows, err := db.Query("UPDATE action_requirement SET value = $1 WHERE value = $2 AND type = $3 RETURNING action_id", newValue, oldValue, reqType)
	if err != nil {
		return nil, sdk.WrapError(err, "UpdateAllRequirements> cannot update action requirements (newValue=%s, oldValue=%s, reqType=%v)", newValue, oldValue, reqType)
	}
	defer rows.Close()

	var actionID int64
	var actions = []int64{}
	for rows.Next() {
		if err := rows.Scan(&actionID); err != nil {
			return nil, sdk.WrapError(err, "UpdateAllRequirements> unable to scan action ID")
		}
		actions = append(actions, actionID)
	}

	return actions, nil
}
