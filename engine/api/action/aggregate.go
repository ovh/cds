package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

type view []aggregator

func (v view) Exec(db gorp.SqlExecutor, as ...*sdk.Action) error {
	if len(as) > 0 {
		for i := range v {
			if err := v[i](db, as...); err != nil {
				return err
			}
		}
	}
	return nil
}

type aggregator func(gorp.SqlExecutor, ...*sdk.Action) error

// FullView for action of type default.
var FullView = view{
	aggregateRequirements,
	aggregateParameters,
	aggregateChildren,
	aggregateAudits,
	group.AggregateOnAction,
}

// LiteView for action of type default.
var LiteView = view{
	aggregateRequirements,
	aggregateParameters,
	group.AggregateOnAction,
}

func aggregateRequirements(db gorp.SqlExecutor, as ...*sdk.Action) error {
	actionIDs := sdk.ActionsToIDs(as)

	var rs []sdk.Requirement
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_requirement WHERE action_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(actionIDs))
	if err := gorpmapping.GetAll(db, query, &rs); err != nil {
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

func aggregateParameters(db gorp.SqlExecutor, as ...*sdk.Action) error {
	actionIDs := sdk.ActionsToIDs(as)

	var ps []actionParameter
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_parameter WHERE action_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(actionIDs))
	if err := gorpmapping.GetAll(db, query, &ps); err != nil {
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

func aggregateAudits(db gorp.SqlExecutor, as ...*sdk.Action) error {
	for i := range as {
		latestAudit, err := GetAuditLatestByActionID(db, as[i].ID)
		if err != nil {
			return err
		}

		oldestAudit, err := GetAuditOldestByActionID(db, as[i].ID)
		if err != nil {
			return err
		}

		as[i].FirstAudit = oldestAudit
		as[i].LastAudit = latestAudit
	}

	return nil
}

func aggregateChildren(db gorp.SqlExecutor, as ...*sdk.Action) error {
	// don't try to load children if action is builtin
	actionsNotBuiltIn := sdk.ActionsFilterNotTypes(as, sdk.BuiltinAction)
	if len(actionsNotBuiltIn) == 0 {
		return nil
	}

	// get edges for all actions, then init a map of edges for all actions
	edges, err := loadEdgesByParentIDs(db, sdk.ActionsToIDs(actionsNotBuiltIn))
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

type edgeAggregator func(gorp.SqlExecutor, ...*actionEdge) error

func aggregateEdgeParameters(db gorp.SqlExecutor, es ...*actionEdge) error {
	edgeIDs := actionEdgesToIDs(es)

	ps := []actionEdgeParameter{}
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_edge_parameter WHERE action_edge_id = ANY(string_to_array($1, ',')::int[]) ORDER BY name",
	).Args(gorpmapping.IDsToQueryString(edgeIDs))
	if err := gorpmapping.GetAll(db, query, &ps); err != nil {
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

func aggregateEdgeChildren(db gorp.SqlExecutor, es ...*actionEdge) error {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE action.id = ANY(string_to_array($1, ',')::int[])",
	).Args(gorpmapping.IDsToQueryString(actionEdgesToChildIDs(es)))

	children, err := getAll(db, query, view{
		aggregateParameters,
		aggregateRequirements,
		aggregateChildren,
		group.AggregateOnAction,
	})
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
