package action

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

// LoadOptionFunc for action.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.Action) error

// LoadOptions provides all options on action loads functions.
var LoadOptions = struct {
	Default          LoadOptionFunc
	WithRequirements LoadOptionFunc
	WithParameters   LoadOptionFunc
	WithChildren     LoadOptionFunc
	WithFlatChildren LoadOptionFunc
	WithAudits       LoadOptionFunc
	WithGroup        LoadOptionFunc
	WithEdge         LoadOptionFunc
}{
	Default:          loadDefault,
	WithRequirements: loadRequirements,
	WithParameters:   loadParameters,
	WithChildren:     loadChildrenRecursively,
	WithFlatChildren: loadFlatChildren,
	WithAudits:       loadAudits,
	WithGroup:        loadGroup,
	WithEdge:         loadEdge,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	if err := loadRequirements(ctx, db, as...); err != nil {
		return err
	}

	if err := loadParameters(ctx, db, as...); err != nil {
		return err
	}

	if err := loadChildrenRecursively(ctx, db, as...); err != nil {
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

func loadEdge(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	// don't try to load children if action is builtin
	actionsNotBuiltIn := sdk.ActionsFilterNotTypes(as, sdk.BuiltinAction)
	if len(actionsNotBuiltIn) == 0 {
		return nil
	}
	actionEdge, err := loadEdgesByParentIDs(ctx, db, sdk.ActionsToIDs(actionsNotBuiltIn), loadEdgeParameters)
	if err != nil {
		return nil
	}

	for i := range as {
		a := as[i]
		a.Actions = make([]sdk.Action, 0)
		for j := range actionEdge {
			ae := actionEdge[j]
			if ae.ParentID != a.ID {
				continue
			}
			params := make([]sdk.Parameter, len(ae.Parameters))
			for i, p := range ae.Parameters {
				params[i] = sdk.Parameter{
					ID:          p.ID,
					Name:        p.Name,
					Type:        p.Type,
					Value:       p.Value,
					Description: p.Description,
					Advanced:    p.Advanced,
				}
			}
			a.Actions = append(a.Actions, sdk.Action{ID: ae.ChildID, Parameters: params, ActionEdgeID: ae.ID})
		}
	}
	return nil
}

func loadFlatChildren(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	// don't try to load children if action is builtin
	actionsNotBuiltIn := sdk.ActionsFilterNotTypes(as, sdk.BuiltinAction)
	if len(actionsNotBuiltIn) == 0 {
		return nil
	}
	// get edges for all actions, then init a map of edges for all actions
	edges, err := loadEdgesByParentIDs(ctx, db, sdk.ActionsToIDs(actionsNotBuiltIn), loadEdgeParameters)
	if err != nil {
		return err
	}

	mapActions := make(map[int64]*sdk.Action)
	for i := range as {
		mapActions[as[i].ID] = as[i]
	}
	mapChilds := make(map[int64][]*sdk.Action)

	for _, e := range edges {
		parent := mapActions[e.ParentID]
		child := mapActions[e.ChildID]

		// Fill child with step data
		child.StepName = e.StepName
		child.Optional = e.Optional
		child.AlwaysExecuted = e.AlwaysExecuted
		child.Enabled = e.Enabled

		child.Parameters = make([]sdk.Parameter, len(e.Parameters))
		for i, ep := range e.Parameters {
			child.Parameters[i] = sdk.Parameter{
				Advanced:    ep.Advanced,
				Description: ep.Description,
				Value:       ep.Value,
				Type:        ep.Type,
				Name:        ep.Name,
				ID:          ep.ID,
			}
		}

		// add child to temp parent
		if _, ok := mapChilds[parent.ID]; !ok {
			mapChilds[parent.ID] = make([]*sdk.Action, 0)
		}
		mapChilds[parent.ID] = append(mapChilds[parent.ID], child)
	}

	for i := range as {
		if children, has := mapChilds[as[i].ID]; has {
			child := make([]sdk.Action, len(children))
			for i, c := range children {
				child[i] = *c
			}
			as[i].Actions = child
			as[i].Requirements = as[i].FlattenRequirementsRecursively()
		}
	}

	return nil
}

func loadChildrenRecursively(ctx context.Context, db gorp.SqlExecutor, as ...*sdk.Action) error {
	// don't try to load children if action is builtin
	actionsNotBuiltIn := sdk.ActionsFilterNotTypes(as, sdk.BuiltinAction)
	if len(actionsNotBuiltIn) == 0 {
		return nil
	}
	// get edges for all actions, then init a map of edges for all actions
	edges, err := loadEdgesByParentIDs(ctx, db, sdk.ActionsToIDs(actionsNotBuiltIn), loadEdgeParameters, loadEdgeChildren)
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
	gs, err := group.LoadAllByIDs(ctx, db, sdk.ActionsToGroupIDs(as))
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
		loadChildrenRecursively,
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
