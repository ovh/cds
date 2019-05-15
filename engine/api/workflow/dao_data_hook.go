package workflow

import (
	"database/sql"
	"encoding/json"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// CountHooksByApplication count hooks by application id
func CountHooksByApplication(db gorp.SqlExecutor, appID int64) (int64, error) {
	query := `
    SELECT count(w_node_hook.*)
    FROM w_node_hook
    JOIN w_node_context ON w_node_context.node_id = w_node_hook.node_id
    WHERE w_node_context.application_id = $1;
  `
	count, err := db.SelectInt(query, appID)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return count, nil
}

// LoadHookByUUID load a hook by his uuid
func LoadHookByUUID(db gorp.SqlExecutor, uuid string) (sdk.NodeHook, error) {
	var hook sdk.NodeHook
	var res dbNodeHookData
	if err := db.SelectOne(&res, "select * from w_node_hook where uuid = $1", uuid); err != nil {
		if err == sql.ErrNoRows {
			return hook, sdk.WithStack(sdk.ErrNotFound)
		}
		return hook, sdk.WithStack(err)
	}

	model, err := LoadHookModelByID(db, res.HookModelID)
	if err != nil {
		return hook, sdk.WithStack(err)
	}
	res.HookModelName = model.Name
	return sdk.NodeHook(res), nil
}

// LoadAllHooks returns all hooks
func LoadAllHooks(db gorp.SqlExecutor) ([]sdk.NodeHook, error) {
	models, err := LoadHookModels(db)
	if err != nil {
		return nil, err
	}

	var res []dbNodeHookData
	if _, err := db.Select(&res, "select * from w_node_hook"); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	nodes := make([]sdk.NodeHook, 0, len(res))
	for i := range res {
		if err := res[i].PostGet(db); err != nil {
			return nil, err
		}
		for _, m := range models {
			if res[i].HookModelID == m.ID {
				res[i].HookModelName = m.Name
				break
			}
		}
		nodes = append(nodes, sdk.NodeHook(res[i]))
	}

	return nodes, nil
}

func (h *dbNodeHookData) PostGet(db gorp.SqlExecutor) error {
	var config sdk.WorkflowNodeHookConfig
	var configS string
	if _, err := db.Select(&configS, "select config from w_node_hook where id = $1", h.ID); err != nil {
		return sdk.WithStack(err)
	}
	if err := json.Unmarshal([]byte(configS), &config); err != nil {
		return sdk.WithStack(err)
	}
	h.Config = config
	return nil
}

func insertNodeHookData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.Hooks == nil || len(n.Hooks) == 0 {
		return nil
	}

	for i := range n.Hooks {
		h := &n.Hooks[i]
		h.NodeID = n.ID

		dbHook := dbNodeHookData(*h)
		if err := db.Insert(&dbHook); err != nil {
			return sdk.WrapError(err, "insertNodeHookData> Unable to insert workflow node hook")
		}
		h.ID = dbHook.ID
	}
	return nil
}

// PostInsert is a db hook
func (h *dbNodeHookData) PostInsert(db gorp.SqlExecutor) error {
	return h.PostUpdate(db)
}

// PostUpdate is a db hook
func (h *dbNodeHookData) PostUpdate(db gorp.SqlExecutor) error {
	config, errC := gorpmapping.JSONToNullString(h.Config)
	if errC != nil {
		return sdk.WrapError(errC, "dbNodeHookData.PostUpdate> Unable to marshall config")
	}

	if _, err := db.Exec("UPDATE w_node_hook SET config = $1 WHERE id = $2", config, h.ID); err != nil {
		return sdk.WrapError(err, "dbNodeHookData.PostUpdate> Unable to update config")
	}
	return nil
}
