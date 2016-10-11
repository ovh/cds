package action

import (
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// LoadAllActionRequirements retrieves all requirements in database
// Used by worker to automatically declare most capabilities
func LoadAllActionRequirements(db database.Querier) ([]sdk.Requirement, error) {
	var req []sdk.Requirement

	query := `SELECT name, type, value FROM action_requirement`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r sdk.Requirement
		var t string
		err = rows.Scan(&r.Name, &t, &r.Value)
		if err != nil {
			return nil, err
		}
		r.Type = sdk.RequirementType(t)
		req = append(req, r)
	}

	return req, nil
}

// LoadActionRequirements retrieves given action requirements in database
func LoadActionRequirements(db database.Querier, actionID int64) ([]sdk.Requirement, error) {
	var req []sdk.Requirement

	query := `SELECT name, type, value FROM action_requirement WHERE action_id = $1 ORDER BY name`
	rows, err := db.Query(query, actionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r sdk.Requirement
		var t string
		err = rows.Scan(&r.Name, &t, &r.Value)
		if err != nil {
			return nil, err
		}
		r.Type = sdk.RequirementType(t)
		req = append(req, r)
	}

	return req, nil
}

// InsertActionRequirement inserts given requirement in database
func InsertActionRequirement(db database.Executer, actionID int64, r sdk.Requirement) error {
	query := `INSERT INTO action_requirement (action_id, name, type, value) VALUES ($1, $2, $3, $4)`

	_, err := db.Exec(query, actionID, r.Name, string(r.Type), r.Value)
	if err != nil {
		return err
	}

	return nil
}

// DeleteActionRequirements deletes all requirements related to given action
func DeleteActionRequirements(db database.Executer, actionID int64) error {
	query := `DELETE FROM action_requirement WHERE action_id = $1`

	_, err := db.Exec(query, actionID)
	if err != nil {
		return err
	}

	return nil
}
