package workflow

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// UpdateHook Update a workflow node hook
func UpdateHook(db gorp.SqlExecutor, h *sdk.WorkflowNodeHook) error {
	dbhook := NodeHook(*h)
	if _, err := db.Update(&dbhook); err != nil {
		return sdk.WrapError(err, "updateHook> Cannot update hook")
	}
	if err := dbhook.PostInsert(db); err != nil {
		return sdk.WrapError(err, "updateHook> Cannot post update hook")
	}
	return nil
}

// insertHook inserts a hook
func insertHook(db gorp.SqlExecutor, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeHook) error {
	hook.WorkflowNodeID = node.ID
	if hook.WorkflowHookModelID == 0 {
		hook.WorkflowHookModelID = hook.WorkflowHookModel.ID
	}

	if hook.WorkflowHookModelID != 0 {
		model, errm := LoadHookModelByID(db, hook.WorkflowHookModelID)
		if errm != nil {
			return sdk.WrapError(errm, "insertHook> Unable to load model %d", hook.WorkflowHookModelID)
		}
		hook.WorkflowHookModel = *model
	} else {
		model, errm := LoadHookModelByName(db, hook.WorkflowHookModel.Name)
		if errm != nil {
			return sdk.WrapError(errm, "insertHook> Unable to load model %s", hook.WorkflowHookModel.Name)
		}
		hook.WorkflowHookModel = *model
	}
	hook.WorkflowHookModelID = hook.WorkflowHookModel.ID

	errmu := sdk.MultiError{}
	// Check configuration of the hook vs the model
	for k := range hook.WorkflowHookModel.DefaultConfig {
		if _, ok := hook.Config[k]; !ok {
			errmu = append(errmu, fmt.Errorf("Missing configuration key: %s", k))
		}
	}
	if len(errmu) > 0 {
		return sdk.WrapError(&errmu, "insertHook> Invalid hook configuration")
	}

	//Keep the uuid if provided
	if hook.UUID == "" {
		uuid, erruuid := sessionstore.NewSessionKey()
		if erruuid != nil {
			return sdk.WrapError(erruuid, "insertHook> Unable to load model %d", hook.WorkflowHookModelID)
		}
		hook.UUID = string(uuid)
	}

	dbhook := NodeHook(*hook)
	if err := db.Insert(&dbhook); err != nil {
		return sdk.WrapError(err, "insertHook> Unable to insert hook")
	}
	*hook = sdk.WorkflowNodeHook(dbhook)
	return nil
}

//PostInsert is a db hook
func (r *NodeHook) PostInsert(db gorp.SqlExecutor) error {
	sConfig, errgo := gorpmapping.JSONToNullString(r.Config)
	if errgo != nil {
		return errgo
	}

	if _, err := db.Exec("update workflow_node_hook set config = $2 where id = $1", r.ID, sConfig); err != nil {
		return err
	}
	return nil
}

//PostGet is a db hook
func (r *NodeHook) PostGet(db gorp.SqlExecutor) error {
	var res = struct {
		Config sql.NullString `db:"config"`
	}{}
	if err := db.SelectOne(&res, "select config from workflow_node_hook where id = $1", r.ID); err != nil {
		return err
	}

	conf := sdk.WorkflowNodeHookConfig{}

	if err := gorpmapping.JSONNullString(res.Config, &conf); err != nil {
		return err
	}

	r.Config = conf

	//Load the model
	model, err := LoadHookModelByID(db, r.WorkflowHookModelID)
	if err != nil {
		return err
	}

	r.WorkflowHookModel = *model

	return nil
}

// LoadAllHooks returns all hooks
func LoadAllHooks(db gorp.SqlExecutor) ([]sdk.WorkflowNodeHook, error) {
	res := []NodeHook{}
	if _, err := db.Select(&res, "select id, uuid, workflow_hook_model_id, workflow_node_id from workflow_node_hook"); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadAllHooks")
	}

	nodes := []sdk.WorkflowNodeHook{}
	for i := range res {
		if err := res[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadAllHooks")
		}
		nodes = append(nodes, sdk.WorkflowNodeHook(res[i]))
	}

	log.Debug("LoadAllHooks> %+v", nodes)

	return nodes, nil
}

func loadHooks(db gorp.SqlExecutor, node *sdk.WorkflowNode) ([]sdk.WorkflowNodeHook, error) {
	res := []NodeHook{}
	if _, err := db.Select(&res, "select id, uuid, workflow_hook_model_id, workflow_node_id from workflow_node_hook where workflow_node_id = $1", node.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadHooks")
	}

	nodes := []sdk.WorkflowNodeHook{}
	for i := range res {
		if err := res[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "loadHooks")
		}
		res[i].WorkflowNodeID = node.ID
		nodes = append(nodes, sdk.WorkflowNodeHook(res[i]))
	}
	return nodes, nil
}
