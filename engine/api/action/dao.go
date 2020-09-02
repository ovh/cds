package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Action, error) {
	pas := []*sdk.Action{}

	if err := gorpmapping.GetAll(ctx, db, q, &pas); err != nil {
		return nil, sdk.WrapError(err, "cannot get actions")
	}
	if len(pas) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pas...); err != nil {
				return nil, err
			}
		}
	}

	as := make([]sdk.Action, len(pas))
	for i := range pas {
		as[i] = *pas[i]
	}

	return as, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Action, error) {
	var a sdk.Action

	found, err := gorpmapping.Get(ctx, db, q, &a)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get action")
	}
	if !found {
		return nil, nil
	}

	for i := range opts {
		if err := opts[i](ctx, db, &a); err != nil {
			return nil, err
		}
	}

	return &a, nil
}

// LoadAllByIDs retrieves in database action with given ids.
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadOptionFunc) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE id = ANY($1)",
	).Args(pq.Int64Array(ids))
	return getAll(ctx, db, query, opts...)
}

// LoadAllByTypes actions from database.
func LoadAllByTypes(ctx context.Context, db gorp.SqlExecutor, types []string, opts ...LoadOptionFunc) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = ANY(string_to_array($1, ',')::text[]) ORDER BY name",
	).Args(strings.Join(types, ","))
	return getAll(ctx, db, query, opts...)
}

// LoadAllTypeDefaultByGroupIDs actions from database.
func LoadAllTypeDefaultByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM action
    WHERE type = $1 AND group_id = ANY(string_to_array($2, ',')::int[])
    ORDER BY name
  `).Args(sdk.DefaultAction, gorpmapping.IDsToQueryString(groupIDs))
	return getAll(ctx, db, query, opts...)
}

// LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs actions from database.
func LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM action
		WHERE
			type = $1
			OR type = $2
      OR (type = $3 AND group_id = ANY(string_to_array($4, ',')::int[]))
    ORDER BY name
	`).Args(
		sdk.BuiltinAction, sdk.PluginAction, sdk.DefaultAction,
		gorpmapping.IDsToQueryString(groupIDs),
	)
	return getAll(ctx, db, query, opts...)
}

// LoadAllByIDsWithTypeBuiltinOrPluginOrDefaultInGroupIDs returns all actions for given ids. Action should be
// of type builtin, plugin or default. Default action should be in given group ids list.
func LoadAllByIDsWithTypeBuiltinOrPluginOrDefaultInGroupIDs(ctx context.Context, db gorp.SqlExecutor, ids, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Action, error) {
	// children should be builtin, plugin or default with group matching
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM action
    WHERE
      id = ANY(string_to_array($1, ',')::int[])
      AND (
        type = $2
        OR type = $3
        OR (type = $4 AND group_id = ANY(string_to_array($5, ',')::int[]))
      )
  `).Args(
		gorpmapping.IDsToQueryString(ids),
		sdk.BuiltinAction,
		sdk.PluginAction,
		sdk.DefaultAction,
		gorpmapping.IDsToQueryString(groupIDs),
	)
	return getAll(ctx, db, query, opts...)
}

// LoadTypeDefaultByNameAndGroupID returns an action from database with given name and group id.
func LoadTypeDefaultByNameAndGroupID(ctx context.Context, db gorp.SqlExecutor, name string, groupID int64, opts ...LoadOptionFunc) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = $1 AND lower(name) = lower($2) AND group_id = $3",
	).Args(sdk.DefaultAction, name, groupID)
	return get(ctx, db, query, opts...)
}

// LoadByTypesAndName returns an action from database with given name and type in list.
func LoadByTypesAndName(ctx context.Context, db gorp.SqlExecutor, types []string, name string, opts ...LoadOptionFunc) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = ANY(string_to_array($1, ',')::text[]) AND lower(name) = lower($2)",
	).Args(strings.Join(types, ","), name)
	return get(ctx, db, query, opts...)
}

// LoadByID retrieves in database the action with given id.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Action, error) {
	query := gorpmapping.NewQuery("SELECT * FROM action WHERE action.id = $1").Args(id)
	return get(ctx, db, query, opts...)
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

// GetRequirementsTypeModelAndValueStartBy returns action requirements from database for given criteria.
func GetRequirementsTypeModelAndValueStartBy(ctx context.Context, db gorp.SqlExecutor, value string) ([]sdk.Requirement, error) {
	rs := []sdk.Requirement{}

	// if value equals Debian9, the regex should match "Debian9" and "Debian9 --foo", but not "Debian9-Foo"
	reg := fmt.Sprintf("^%s(?!\\S)", value)

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM action_requirement
    WHERE type = 'model' AND (value ~ $1)
  `).Args(reg)

	if err := gorpmapping.GetAll(ctx, db, query, &rs); err != nil {
		return nil, sdk.WrapError(err, "cannot get requirements")
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

// UpdateRequirement in database.
func UpdateRequirement(db gorp.SqlExecutor, r *sdk.Requirement) error {
	return sdk.WrapError(gorpmapping.Update(db, r), "unable to update action requirement %d", r.ID)
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

func getEdges(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, fs ...loadOptionEdgeFunc) ([]actionEdge, error) {
	paes := []*actionEdge{}

	if err := gorpmapping.GetAll(ctx, db, q, &paes); err != nil {
		return nil, sdk.WrapError(err, "cannot get action edges")
	}
	if len(paes) > 0 {
		for i := range fs {
			if err := fs[i](ctx, db, paes...); err != nil {
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

// loadEdgesByParentIDs retrieves in database all action edges for given parent ids.
func loadEdgesByParentIDs(ctx context.Context, db gorp.SqlExecutor, parentIDs []int64) ([]actionEdge, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_edge WHERE parent_id = ANY(string_to_array($1, ',')::int[]) ORDER BY exec_order ASC",
	).Args(gorpmapping.IDsToQueryString(parentIDs))
	return getEdges(ctx, db, query,
		loadEdgeParameters,
		loadEdgeChildren,
	)
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

// InsertAudit in database.
func InsertAudit(db gorp.SqlExecutor, aa *sdk.AuditAction) error {
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

// GetAuditByActionIDAndID returns audit for given action id and audit id.
func GetAuditByActionIDAndID(ctx context.Context, db gorp.SqlExecutor, actionID, auditID int64) (*sdk.AuditAction, error) {
	var aa sdk.AuditAction

	query := gorpmapping.NewQuery(`SELECT * FROM action_audit WHERE action_id = $1 AND id = $2`).
		Args(actionID, auditID)
	found, err := gorpmapping.Get(ctx, db, query, &aa)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get action audit %d for action %d", auditID, actionID)
	}
	if !found {
		return nil, nil
	}

	return &aa, nil
}

// GetAuditLatestByActionID returns action latest audit by action id.
func GetAuditLatestByActionID(ctx context.Context, db gorp.SqlExecutor, actionID int64) (*sdk.AuditAction, error) {
	var aa sdk.AuditAction

	query := gorpmapping.NewQuery(`SELECT * FROM action_audit WHERE action_id = $1 ORDER BY created DESC LIMIT 1`).Args(actionID)
	found, err := gorpmapping.Get(ctx, db, query, &aa)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get latest audit for action %d", actionID)
	}
	if !found {
		return nil, nil
	}

	return &aa, nil
}

// GetAuditOldestByActionID returns action oldtest audit by action id.
func GetAuditOldestByActionID(ctx context.Context, db gorp.SqlExecutor, actionID int64) (*sdk.AuditAction, error) {
	var aa sdk.AuditAction

	query := gorpmapping.NewQuery(`SELECT * FROM action_audit WHERE action_id = $1 ORDER BY created ASC LIMIT 1`).Args(actionID)
	found, err := gorpmapping.Get(ctx, db, query, &aa)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get oldtest audit for action %d", actionID)
	}
	if !found {
		return nil, nil
	}

	return &aa, nil
}
