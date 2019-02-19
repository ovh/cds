package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadAllTypeDefault actions from database.
func LoadAllTypeDefault(db gorp.SqlExecutor) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = $1 ORDER BY name",
	).Args(sdk.DefaultAction)
	return getAll(db, query, FullView)
}

// LoadAllTypeDefaultByGroupIDs actions from database.
func LoadAllTypeDefaultByGroupIDs(db gorp.SqlExecutor, groupIDs []int64) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM action
    WHERE type = $1 AND group_id = ANY(string_to_array($2, ',')::int[])
    ORDER BY name
  `).Args(sdk.DefaultAction, gorpmapping.IDsToQueryString(groupIDs))
	return getAll(db, query, FullView)
}

// LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs actions from database.
func LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs(db gorp.SqlExecutor, groupIDs []int64) ([]sdk.Action, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM action
		WHERE 
			type = $1
			OR type = $2
			OR (type = $3 AND group_id = ANY(string_to_array($4, ',')::int[]))
	`).Args(
		sdk.BuiltinAction,
		sdk.PluginAction,
		sdk.DefaultAction,
		gorpmapping.IDsToQueryString(groupIDs),
	)
	return getAll(db, query, LiteView)
}

// LoadTypeBuiltInByName returns a action from database for given name.
func LoadTypeBuiltInByName(db gorp.SqlExecutor, name string) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = $1 AND lower(action.name) = lower($3)",
	).Args(sdk.BuiltinAction, name)
	return get(db, query, FullView)
}

// LoadTypePluginByName returns a action from database for given name.
func LoadTypePluginByName(db gorp.SqlExecutor, name string) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = $1 AND lower(action.name) = lower($2)",
	).Args(sdk.PluginAction, name)
	return get(db, query, FullView)
}

// LoadTypeDefaultByNameAndGroupID returns an action from database with given name and group id.
func LoadTypeDefaultByNameAndGroupID(db gorp.SqlExecutor, name string, groupID int64) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE type = $1 AND lower(name) = lower($2) AND group_id = $3",
	).Args(sdk.DefaultAction, name, groupID)
	return get(db, query, LiteView)
}

// LoadTypeBuiltInOrPluginByName returns an action from database with given name.
func LoadTypeBuiltInOrPluginByName(db gorp.SqlExecutor, name string) (*sdk.Action, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action WHERE (type = $1 OR type = $2) AND lower(name) = lower($3)",
	).Args(sdk.BuiltinAction, sdk.PluginAction, name)
	return get(db, query, LiteView)
}

// LoadByID retrieves in database the action with given id.
func LoadByID(db gorp.SqlExecutor, id int64) (*sdk.Action, error) {
	query := gorpmapping.NewQuery("SELECT * FROM action WHERE action.id = $1").Args(id)
	return get(db, query, FullView)
}

// loadEdgesByParentIDs retrieves in database all action edges for given parent ids.
func loadEdgesByParentIDs(db gorp.SqlExecutor, parentIDs []int64) ([]actionEdge, error) {
	query := gorpmapping.NewQuery(
		"SELECT * FROM action_edge WHERE parent_id = ANY(string_to_array($1, ',')::int[]) ORDER BY exec_order ASC",
	).Args(gorpmapping.IDsToQueryString(parentIDs))
	return getEdges(db, query,
		aggregateEdgeParameters,
		aggregateEdgeChildren,
	)
}
