package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAll(db gorp.SqlExecutor, q gorpmapping.Query, v view) ([]sdk.Action, error) {
	pas := []*sdk.Action{}

	if err := gorpmapping.GetAll(db, q, &pas); err != nil {
		return nil, sdk.WrapError(err, "cannot get actions")
	}
	if err := v.Exec(db, pas...); err != nil {
		return nil, err
	}

	as := make([]sdk.Action, len(pas))
	for i := range pas {
		as[i] = *pas[i]
	}

	return as, nil
}

func get(db gorp.SqlExecutor, q gorpmapping.Query, v view) (*sdk.Action, error) {
	var a sdk.Action

	found, err := gorpmapping.Get(db, q, &a)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get action")
	}
	if !found {
		return nil, nil
	}

	if err := v.Exec(db, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

// GetTypeDefaultByNameAndGroupID returns an action from database with given name and group id.
func GetTypeDefaultByNameAndGroupID(db gorp.SqlExecutor, name string, groupID int64) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = $1 AND lower(name) = lower($2) AND group_id = $3",
	).Args(sdk.DefaultAction, name, groupID)
	return get(db, query, nil)
}

// insert action in database.
func insert(db gorp.SqlExecutor, a *sdk.Action) error {
	return sdk.WrapError(gorpmapping.Insert(db, a), "unable to insert action %s", a.Name)
}

// update action in database.
func update(db gorp.SqlExecutor, a *sdk.Action) error {
	return sdk.WrapError(gorpmapping.Update(db, a), "unable to update action %s", a.Name)
}

// Delete action in database.
func Delete(db gorp.SqlExecutor, a *sdk.Action) error {
	return sdk.WrapError(gorpmapping.Delete(db, a), "unable to delete action %s", a.Name)
}

// DeleteAllTypeJoinedByIDs deletes all joined action by ids.
func DeleteAllTypeJoinedByIDs(db gorp.SqlExecutor, ids []int64) error {
	_, err := db.Exec("DELETE FROM action WHERE type = $1 AND id = ANY(string_to_array($2, ',')::int[])",
		sdk.JoinedAction, gorpmapping.IDsToQueryString(ids))
	return sdk.WithStack(err)
}

// DeleteTypeJoinedByID deletes joined action for id.
func DeleteTypeJoinedByID(db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec("DELETE FROM action WHERE type = $1 AND id = $2", sdk.JoinedAction, id)
	return sdk.WithStack(err)
}

// DeleteRequirementsByActionID deletes all requirements related to given action.
func DeleteRequirementsByActionID(db gorp.SqlExecutor, actionID int64) error {
	_, err := db.Exec("DELETE FROM action_requirement WHERE action_id = $1", actionID)
	return sdk.WithStack(err)
}

// GetRequirementsDistinctBinary retrieves all binary requirements in database.
// Used by worker to automatically declare most capabilities, this func returns denormalized values.
func GetRequirementsDistinctBinary(db gorp.SqlExecutor) (sdk.RequirementList, error) {
	rows, err := db.Query("SELECT distinct value FROM action_requirement where type = 'binary'")
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer rows.Close()

	var rs []sdk.Requirement
	var value string
	for rows.Next() {
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		rs = append(rs, sdk.Requirement{
			Name:  value,
			Type:  sdk.BinaryRequirement,
			Value: value,
		})
	}

	return rs, nil
}

// InsertRequirement in database.
func InsertRequirement(db gorp.SqlExecutor, r *sdk.Requirement) error {
	if r.Name == "" || r.Type == "" || r.Value == "" {
		return sdk.WithStack(sdk.ErrInvalidJobRequirement)
	}
	return sdk.WithStack(gorpmapping.Insert(db, r))
}

// UpdateRequirementsValue updates all action_requirement.value given a value and a type then returns action IDs.
func UpdateRequirementsValue(db gorp.SqlExecutor, oldValue, newValue, reqType string) ([]int64, error) {
	rows, err := db.Query("UPDATE action_requirement SET value = $1 WHERE value = $2 AND type = $3 RETURNING action_id", newValue, oldValue, reqType)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot update action requirements (newValue=%s, oldValue=%s, reqType=%v)", newValue, oldValue, reqType)
	}
	defer rows.Close()

	var actionID int64
	var actionIDs = []int64{}
	for rows.Next() {
		if err := rows.Scan(&actionID); err != nil {
			return nil, sdk.WrapError(err, "unable to scan action id")
		}
		actionIDs = append(actionIDs, actionID)
	}

	return actionIDs, nil
}

func insertParameter(db gorp.SqlExecutor, p *actionParameter) error {
	if string(p.Type) == string(sdk.SecretVariable) {
		return sdk.WithStack(sdk.ErrNoDirectSecretUse)
	}
	return sdk.WrapError(gorpmapping.Insert(db, p), "unable to insert parameter for action %d", p.ActionID)
}

func deleteParametersByActionID(db gorp.SqlExecutor, actionID int64) error {
	_, err := db.Exec("DELETE FROM action_parameter WHERE action_id = $1", actionID)
	return sdk.WithStack(err)
}

func getEdges(db gorp.SqlExecutor, q gorpmapping.Query, ags ...edgeAggregator) ([]actionEdge, error) {
	paes := []*actionEdge{}

	if err := gorpmapping.GetAll(db, q, &paes); err != nil {
		return nil, sdk.WrapError(err, "cannot get action edges")
	}
	if len(paes) > 0 {
		for i := range ags {
			if err := ags[i](db, paes...); err != nil {
				return nil, err
			}
		}
	}

	aes := make([]actionEdge, len(paes))
	for i := range paes {
		aes[i] = *paes[i]
	}

	return aes, nil
}

func insertEdge(db gorp.SqlExecutor, ae *actionEdge) error {
	return sdk.WrapError(gorpmapping.Insert(db, ae), "unable to insert action edge for parent %d and child %d", ae.ParentID, ae.ChildID)
}

func insertEdgeParameter(db gorp.SqlExecutor, aep *actionEdgeParameter) error {
	return sdk.WrapError(gorpmapping.Insert(db, aep), "unable to insert action edge parameter for edge %d", aep.ActionEdgeID)
}

// deleteEdgesByParentID delete all action edge in database for a given parentID
func deleteEdgesByParentID(db gorp.SqlExecutor, parentID int64) error {
	_, err := db.Exec("DELETE FROM action_edge WHERE parent_id = $1", parentID)
	return sdk.WithStack(err)
}

func insertAudit(db gorp.SqlExecutor, aa *sdk.AuditAction) error {
	return sdk.WrapError(gorpmapping.Insert(db, aa), "unable to insert audit for action %d", aa.ActionID)
}

// GetAuditsByActionID returns all action audits by action ids.
func GetAuditsByActionID(db gorp.SqlExecutor, actionID int64) ([]sdk.AuditAction, error) {
	aas := []sdk.AuditAction{}

	if _, err := db.Select(&aas,
		`SELECT * FROM action_audit
     WHERE action_id = $1
     ORDER BY created DESC`,
		actionID,
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get action audits")
	}

	return aas, nil
}

// GetAuditLatestByActionID returns action latest audit by action id.
func GetAuditLatestByActionID(db gorp.SqlExecutor, actionID int64) (*sdk.AuditAction, error) {
	var aa sdk.AuditAction

	if _, err := db.Select(&aa,
		`SELECT * FROM action_audit
     WHERE action_id = $1
     ORDER BY created DESC LIMIT 1`,
		actionID,
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get action latest audit")
	}

	return &aa, nil
}

// GetAuditOldestByActionID returns action oldtest audit by action id.
func GetAuditOldestByActionID(db gorp.SqlExecutor, actionID int64) (*sdk.AuditAction, error) {
	var aa sdk.AuditAction

	if _, err := db.Select(&aa,
		`SELECT * FROM action_audit
     WHERE action_id = $1
     ORDER BY created ASC LIMIT 1`,
		actionID,
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get action oldtest audit")
	}

	return &aa, nil
}
