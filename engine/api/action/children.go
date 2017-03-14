package action

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func insertEdge(db gorp.SqlExecutor, parentID, childID int64, execOrder int, final, enabled bool) (int64, error) {
	query := `INSERT INTO action_edge (parent_id, child_id, exec_order, final, enabled) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	var id int64
	err := db.QueryRow(query, parentID, childID, execOrder, final, enabled).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func insertActionChild(db gorp.SqlExecutor, actionID int64, child sdk.Action, execOrder int) error {
	if child.ID == 0 {
		return fmt.Errorf("insertActionChild: child action has no id")
	}

	id, err := insertEdge(db, actionID, child.ID, execOrder, child.Final, child.Enabled)
	if err != nil {
		return err
	}

	// Insert all parameters
	for i := range child.Parameters {
		log.Debug("insertActionChild> %s : %v", child.Name, child.Parameters[i])
		err = insertChildActionParameter(db, id, actionID, child.ID, child.Parameters[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func insertChildActionParameter(db gorp.SqlExecutor, edgeID, parentID, childID int64, param sdk.Parameter) error {

	query := `INSERT INTO action_edge_parameter (
					action_edge_id,
					name,
					type,
					value,
					description) VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, edgeID, param.Name, string(param.Type), param.Value, param.Description)
	if err != nil {
		return err
	}

	return nil
}

// loadActionChildren loads all children actions from given action
func loadActionChildren(db gorp.SqlExecutor, actionID int64) ([]sdk.Action, error) {
	var children []sdk.Action
	var edgeIDs []int64
	var childrenIDs []int64
	query := `SELECT id, child_id, exec_order, final, enabled FROM action_edge WHERE parent_id = $1 ORDER BY exec_order ASC`

	rows, err := db.Query(query, actionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edgeID, childID int64
	var execOrder int
	var final, enabled bool
	var mapFinal = make(map[int64]bool)
	var mapEnabled = make(map[int64]bool)

	for rows.Next() {
		err = rows.Scan(&edgeID, &childID, &execOrder, &final, &enabled)
		if err != nil {
			return nil, err
		}
		edgeIDs = append(edgeIDs, edgeID)
		childrenIDs = append(childrenIDs, childID)
		mapFinal[edgeID] = final
		mapEnabled[edgeID] = enabled
	}
	rows.Close()

	for _, childID := range childrenIDs {
		a, err := LoadActionByID(db, childID)
		if err != nil {
			return nil, fmt.Errorf("cannot LoadActionByID> %s", err)
		}
		children = append(children, *a)
	}

	for i := range children {
		// Load child action parameter value
		params, err := loadChildActionParameterValue(db, edgeIDs[i])
		if err != nil {
			return nil, fmt.Errorf("cannot loadChildActionParameterValue> %s", err)
		}

		// If child action has been modified, new parameters will show
		// and delete one won't be there anymore
		replaceChildActionParameters(&children[i], params)
		// Get final flag
		children[i].Final = mapFinal[edgeIDs[i]]
		// Get enable flag
		children[i].Enabled = mapEnabled[edgeIDs[i]]
	}

	return children, nil
}

//func loadChildActionParameterValue(db gorp.SqlExecutor, edgeID int64, args ...LoadActionFuncArg) ([]sdk.Parameter, error) {
func loadChildActionParameterValue(db gorp.SqlExecutor, edgeID int64) ([]sdk.Parameter, error) {
	var params []sdk.Parameter

	query := `SELECT name, type, value, description FROM action_edge_parameter
							WHERE action_edge_id = $1 ORDER BY name`
	rows, err := db.Query(query, edgeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p sdk.Parameter
		var pType, val string
		err = rows.Scan(&p.Name, &pType, &val, &p.Description)
		if err != nil {
			return nil, err
		}
		p.Type = pType
		p.Value = val

		params = append(params, p)
	}

	return params, nil
}

// Replace action parameter with value configured by user when he created the child action
func replaceChildActionParameters(a *sdk.Action, params []sdk.Parameter) {

	// So for each _existing_ parameter in child action
	for i := range a.Parameters {
		// search parameter matching the name
		for _, p := range params {
			if p.Name == a.Parameters[i].Name {
				a.Parameters[i].Value = p.Value
				break
			}
		}
	}

	// New parameter will have their default value
}

// deleteActionChildren delete all action of a given action in database
func deleteActionChildren(db gorp.SqlExecutor, actionID int64) error {
	query := `DELETE FROM action_edge_parameter WHERE action_edge_id IN (select id FROM action_edge WHERE parent_id = $1)`
	_, err := db.Exec(query, actionID)
	if err != nil {
		return err
	}

	query = `DELETE FROM action_edge WHERE parent_id = $1`
	_, err = db.Exec(query, actionID)
	if err != nil {
		return err
	}

	return nil
}
