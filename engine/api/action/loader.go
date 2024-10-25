package action

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for action.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.Action) error

// LoadOptions provides all options on action loads functions.
var LoadOptions = struct {
	Default          LoadOptionFunc
	WithRequirements LoadOptionFunc
	WithParameters   LoadOptionFunc
	WithChildren     LoadOptionFunc
	WithAudits       LoadOptionFunc
	WithGroup        LoadOptionFunc
}{
	Default:          loadDefault,
	WithRequirements: loadRequirements,
	WithParameters:   loadParameters,
	WithChildren:     loadChildren,
	WithAudits:       loadAudits,
	WithGroup:        loadGroup,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	if err := loadRequirements(ctx, db, as...); err != nil {
		return err
	}

	if err := loadParameters(ctx, db, as...); err != nil {
		return err
	}

	if err := loadChildren(ctx, db, as...); err != nil {
		return err
	}

	return loadGroup(ctx, db, as...)
}

func loadRequirements(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	actionIDs := sdk.ActionsToIDs(as)

	var rs []sdk.Requirement
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_requirement WHERE action_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(actionIDs))
	if err := gorpmapping.GetAll(ctx, db, query, &rs); err != nil {
		return sdk.WrapError(err, "cannot get requirements for action ids %v", actionIDs)
	}

	m := make(map[int64][]sdk.Requirement, len(rs))
	for i := range rs {
		if _, ok := m[rs[i].ActionID]; !ok {
			m[rs[i].ActionID] = make([]sdk.Requirement, 0)
		}
		m[rs[i].ActionID] = append(m[rs[i].ActionID], rs[i])
	}
	for i := range as {
		if rs, ok := m[as[i].ID]; ok {
			as[i].Requirements = rs
		}
	}

	return nil
}

func loadParameters(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	actionIDs := sdk.ActionsToIDs(as)

	var ps []actionParameter
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_parameter WHERE action_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(actionIDs))
	if err := gorpmapping.GetAll(ctx, db, query, &ps); err != nil {
		return sdk.WrapError(err, "cannot get parameters for action ids %v", actionIDs)
	}

	m := make(map[int64][]actionParameter, len(ps))
	for i := range ps {
		if _, ok := m[ps[i].ActionID]; !ok {
			m[ps[i].ActionID] = make([]actionParameter, 0)
		}
		m[ps[i].ActionID] = append(m[ps[i].ActionID], ps[i])
	}
	for i := range as {
		if ps, ok := m[as[i].ID]; ok {
			as[i].Parameters = actionParametersToParameters(ps)
		}
	}

	return nil
}

func loadAudits(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	for i := range as {
		latestAudit, err := GetAuditLatestByActionID(ctx, db, as[i].ID)
		if err != nil {
			return err
		}

		oldestAudit, err := GetAuditOldestByActionID(ctx, db, as[i].ID)
		if err != nil {
			return err
		}

		as[i].FirstAudit = oldestAudit
		as[i].LastAudit = latestAudit
	}

	return nil
}

func loadChildren(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	// don't try to load children if action is builtin
	actionsNotBuiltIn := sdk.ActionsFilterNotTypes(as, sdk.BuiltinAction)
	if len(actionsNotBuiltIn) == 0 {
		return nil
	}

	// get edges for all actions, then init a map of edges for all actions
	edges, err := loadEdgesByParentIDs(ctx, db, sdk.ActionsToIDs(actionsNotBuiltIn))
	if err != nil {
		return err
	}
	mEdges := make(map[int64][]actionEdge, len(edges))
	for i := range edges {
		if _, ok := mEdges[edges[i].ParentID]; !ok {
			mEdges[edges[i].ParentID] = make([]actionEdge, 0)
		}
		mEdges[edges[i].ParentID] = append(mEdges[edges[i].ParentID], edges[i])
	}

	// for all actions set children from its edges
	for i := range actionsNotBuiltIn {
		edges, ok := mEdges[actionsNotBuiltIn[i].ID]
		if !ok {
			continue
		}

		children := make([]sdk.Action, len(edges))
		for i := range edges {
			// init child from edge child then override with edge attributes and parameters
			child := *edges[i].Child
			child.StepName = edges[i].StepName
			child.Optional = edges[i].Optional
			child.AlwaysExecuted = edges[i].AlwaysExecuted
			child.Enabled = edges[i].Enabled

			if child.Type != sdk.AsCodeAction {
				// replace action parameter with value configured by user when he created the child action
				params := make([]sdk.Parameter, len(child.Parameters))
				for j := range child.Parameters {
					params[j] = child.Parameters[j]
					for k := range edges[i].Parameters {
						if edges[i].Parameters[k].Name == params[j].Name {
							params[j].Value = edges[i].Parameters[k].Value
							break
						}
					}
				}
				child.Parameters = params
			} else {
				child.Parameters = make([]sdk.Parameter, 0, len(edges[i].Parameters))
				for _, ep := range edges[i].Parameters {
					child.Parameters = append(child.Parameters, sdk.Parameter{
						Name:        ep.Name,
						Type:        ep.Type,
						Value:       ep.Value,
						Description: ep.Description,
					})
				}
			}
			children[i] = child
		}

		actionsNotBuiltIn[i].Actions = children
	}

	// for all actions update its requirements from its children
	for i := range actionsNotBuiltIn {
		actionsNotBuiltIn[i].Requirements = actionsNotBuiltIn[i].FlattenRequirements()
	}

	return nil
}

func loadGroup(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	groupIDs := sdk.ActionsToGroupIDs(as)
	if len(groupIDs) == 0 {
		return nil
	}

	gs, err := group.LoadAllByIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}

	m := make(map[int64]sdk.Group, len(gs))
	for i := range gs {
		m[gs[i].ID] = gs[i]
	}

	for _, a := range as {
		if a.GroupID != nil {
			if g, ok := m[*a.GroupID]; ok {
				a.Group = &g
			}
		}
	}

	return nil
}

type loadOptionEdgeFunc func(context.Context, gorp.SqlExecutor, ...*actionEdge) error

func loadEdgeParameters(ctx context.Context, db gorp.SqlExecutor, es ...*actionEdge) error {
	edgeIDs := actionEdgesToIDs(es)

	ps := []actionEdgeParameter{}
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_edge_parameter WHERE action_edge_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(edgeIDs))
	if err := gorpmapping.GetAll(ctx, db, query, &ps); err != nil {
		return sdk.WrapError(err, "cannot get action edge parameters for edge ids %d", edgeIDs)
	}

	m := make(map[int64][]actionEdgeParameter, len(ps))
	for i := range ps {
		if _, ok := m[ps[i].ActionEdgeID]; !ok {
			m[ps[i].ActionEdgeID] = make([]actionEdgeParameter, 0)
		}
		m[ps[i].ActionEdgeID] = append(m[ps[i].ActionEdgeID], ps[i])
	}
	for i := range es {
		if ps, ok := m[es[i].ID]; ok {
			es[i].Parameters = ps
		}
	}

	return nil
}

func loadEdgeChildren(ctx context.Context, db gorp.SqlExecutor, es ...*actionEdge) error {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE action.id = ANY(string_to_array($1, ',')::int[])",
	).Args(gorpmapping.IDsToQueryString(actionEdgesToChildIDs(es)))

	children, err := getAll(ctx, db, query,
		loadParameters,
		loadRequirements,
		loadGroup,
		loadChildren,
	)
	if err != nil {
		return err
	}

	m := make(map[int64]sdk.Action, len(children))
	for i := range children {
		m[children[i].ID] = children[i]
	}
	for i := range es {
		if child, ok := m[es[i].ChildID]; ok {
			es[i].Child = &child
		}
	}

	return nil
}
