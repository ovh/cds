package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// UpdateHook Update a workflow node hook
func UpdateHook(db gorp.SqlExecutor, h *sdk.WorkflowNodeHook) error {
	dbhook := NodeHook(*h)
	if _, err := db.Update(&dbhook); err != nil {
		return sdk.WrapError(err, "Cannot update hook")
	}
	if err := dbhook.PostInsert(db); err != nil {
		return sdk.WrapError(err, "Cannot post update hook")
	}
	return nil
}

// DeleteHook Delete a workflow node hook
func DeleteHook(db gorp.SqlExecutor, h *sdk.WorkflowNodeHook) error {
	dbhook := NodeHook(*h)
	if _, err := db.Delete(&dbhook); err != nil {
		return sdk.WrapError(err, "Cannot update hook")
	}
	return nil
}

// insertHook inserts a hook
func insertHook(db gorp.SqlExecutor, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeHook) error {
	hook.WorkflowNodeID = node.ID
	if hook.WorkflowHookModelID == 0 {
		hook.WorkflowHookModelID = hook.WorkflowHookModel.ID
	}

	var icon string
	if hook.WorkflowHookModelID != 0 {
		model, errm := LoadHookModelByID(db, hook.WorkflowHookModelID)
		if errm != nil {
			return sdk.WrapError(errm, "insertHook> Unable to load model %d", hook.WorkflowHookModelID)
		}
		hook.WorkflowHookModel = *model
		icon = hook.WorkflowHookModel.Icon
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
			errmu = append(errmu, fmt.Errorf("Missing %s configuration key: %s", hook.WorkflowHookModel.Name, k))
		}
	}
	if len(errmu) > 0 {
		return sdk.WrapError(&errmu, "insertHook> Invalid hook configuration")
	}

	// if it's a new hook
	if hook.UUID == "" {
		hook.UUID = sdk.UUID()
		if hook.Ref == "" {
			hook.Ref = fmt.Sprintf("%d", time.Now().Unix())
		}

		hook.Config["hookIcon"] = sdk.WorkflowNodeHookConfigValue{
			Value:        icon,
			Configurable: false,
		}
	}

	dbhook := NodeHook(*hook)
	if err := db.Insert(&dbhook); err != nil {
		return sdk.WrapError(err, "Unable to insert hook")
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
	if _, err := db.Select(&res, "select id, uuid, ref, workflow_hook_model_id, workflow_node_id from workflow_node_hook"); err != nil {
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

	return nodes, nil
}

func loadHooks(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode) ([]sdk.WorkflowNodeHook, error) {
	res := []NodeHook{}
	if _, err := db.Select(&res, "select id, uuid, ref, workflow_hook_model_id, workflow_node_id from workflow_node_hook where workflow_node_id = $1", node.ID); err != nil {
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
		w.HookModels[res[i].WorkflowHookModelID] = res[i].WorkflowHookModel
		nodes = append(nodes, sdk.WorkflowNodeHook(res[i]))
	}
	return nodes, nil
}

func loadOutgoingHooks(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User, opts LoadOptions) ([]sdk.WorkflowNodeOutgoingHook, error) {
	res := []nodeOutgoingHook{}
	if _, err := db.Select(&res, "select id, name, workflow_hook_model_id from workflow_node_outgoing_hook where workflow_node_id = $1", node.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadOutgoingHooks")
	}

	hooks := make([]sdk.WorkflowNodeOutgoingHook, len(res))
	for i := range res {
		if err := res[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "loadOutgoingHooks")
		}
		res[i].WorkflowNodeID = node.ID
		hooks[i] = sdk.WorkflowNodeOutgoingHook(res[i])
		w.OutGoingHookModels[hooks[i].WorkflowHookModelID] = hooks[i].WorkflowHookModel

		//Select triggers id
		var triggerIDs []int64
		if _, err := db.Select(&triggerIDs, "select id from workflow_node_outgoing_hook_trigger where  workflow_node_outgoing_hook_id = $1", hooks[i].ID); err != nil {
			if err == sql.ErrNoRows {
				return nil, sdk.WrapError(err, "Unable to load hook triggers id for hook %d", hooks[i].ID)
			}
			return nil, sdk.WrapError(err, "Unable to load hook triggers id for hook %d", hooks[i].ID)
		}

		//Load triggers
		for _, t := range triggerIDs {
			jt, err := loadHookTrigger(ctx, db, store, proj, w, &hooks[i], t, u, opts)
			if err != nil {
				if sdk.Cause(err) == sql.ErrNoRows {
					log.Info("nodeOutgoingHook.PostGet> trigger %d not found", t)
					continue
				}
				return nil, sdk.WrapError(err, "Unable to load hook trigger %d", t)
			}

			hooks[i].Triggers = append(hooks[i].Triggers, jt)
		}
	}

	return hooks, nil
}

// LoadHookByUUID loads a single hook
func LoadHookByUUID(db gorp.SqlExecutor, uuid string) (*sdk.WorkflowNodeHook, error) {
	query := `
		SELECT id, uuid, ref, workflow_hook_model_id, workflow_node_id
			FROM workflow_node_hook
		WHERE uuid = $1`

	res := NodeHook{}
	if err := db.SelectOne(&res, query, uuid); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	if err := res.PostGet(db); err != nil {
		return nil, sdk.WrapError(err, "cannot load postget")
	}
	wNodeHook := sdk.WorkflowNodeHook(res)

	return &wNodeHook, nil
}

// LoadHooksByNodeID loads hooks linked to a nodeID
func LoadHooksByNodeID(db gorp.SqlExecutor, nodeID int64) ([]sdk.WorkflowNodeHook, error) {
	query := `
		SELECT id, uuid, ref, workflow_hook_model_id, workflow_node_id
			FROM workflow_node_hook
		WHERE workflow_node_id = $1`

	res := []NodeHook{}
	if _, err := db.Select(&res, query, nodeID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	nodeHooks := make([]sdk.WorkflowNodeHook, len(res))
	for i, nh := range res {
		if err := nh.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "cannot load postget")
		}
		nodeHooks[i] = sdk.WorkflowNodeHook(nh)
	}

	return nodeHooks, nil
}

// insertOutgoingHook inserts a outgoing hook
func insertOutgoingHook(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, node *sdk.WorkflowNode, hook *sdk.WorkflowNodeOutgoingHook, u *sdk.User) error {
	hook.WorkflowNodeID = node.ID
	if hook.WorkflowHookModelID == 0 {
		hook.WorkflowHookModelID = hook.WorkflowHookModel.ID
	}

	var icon string
	if hook.WorkflowHookModelID != 0 {
		icon = hook.WorkflowHookModel.Icon
		model, errm := LoadOutgoingHookModelByID(db, hook.WorkflowHookModelID)
		if errm != nil {
			return sdk.WrapError(errm, "insertHook> Unable to load model %d", hook.WorkflowHookModelID)
		}
		hook.WorkflowHookModel = *model
	} else {
		model, errm := LoadOutgoingHookModelByName(db, hook.WorkflowHookModel.Name)
		if errm != nil {
			return sdk.WrapError(errm, "insertHook> Unable to load model %s", hook.WorkflowHookModel.Name)
		}
		hook.WorkflowHookModel = *model
		icon = model.Icon
	}
	hook.WorkflowHookModelID = hook.WorkflowHookModel.ID

	hook.Config["hookIcon"] = sdk.WorkflowNodeHookConfigValue{
		Value:        icon,
		Configurable: false,
	}
	hook.Config[sdk.HookConfigProject] = sdk.WorkflowNodeHookConfigValue{Value: w.ProjectKey}
	hook.Config[sdk.HookConfigWorkflow] = sdk.WorkflowNodeHookConfigValue{Value: w.Name}

	//Checks minimal configuration upon its model
	for k := range hook.WorkflowHookModel.DefaultConfig {
		if configuredValue, has := hook.Config[k]; !has {
			return sdk.NewError(sdk.ErrInvalidHookConfiguration, fmt.Errorf("hook %s invalid configuration. %s is missing", hook.Name, k))
		} else if configuredValue.Value == "" {
			return sdk.NewError(sdk.ErrInvalidHookConfiguration, fmt.Errorf("hook %s invalid configuration. %s is missing", hook.Name, k))
		}
	}

	dbhook := nodeOutgoingHook(*hook)
	if err := db.Insert(&dbhook); err != nil {
		return sdk.WrapError(err, "Unable to insert hook")
	}
	*hook = sdk.WorkflowNodeOutgoingHook(dbhook)

	//Setup destination triggers
	for i := range hook.Triggers {
		t := &hook.Triggers[i]
		if errJT := insertOutgoingTrigger(db, store, w, *hook, t, u); errJT != nil {
			return sdk.WrapError(errJT, "insertOutgoingHook> Unable to insert or update trigger")
		}
	}

	return nil
}

// PostInsert is a db hook
func (h *nodeOutgoingHook) PostInsert(db gorp.SqlExecutor) error {
	sConfig, errgo := gorpmapping.JSONToNullString(h.Config)
	if errgo != nil {
		return errgo
	}

	if _, err := db.Exec("update workflow_node_outgoing_hook set config = $2 where id = $1", h.ID, sConfig); err != nil {
		return err
	}
	return nil
}

// PostGet is a db hook
func (h *nodeOutgoingHook) PostGet(db gorp.SqlExecutor) error {
	resConfig, err := db.SelectNullStr("select config from workflow_node_outgoing_hook where id = $1", h.ID)
	if err != nil {
		return err
	}

	conf := sdk.WorkflowNodeHookConfig{}

	if err := gorpmapping.JSONNullString(resConfig, &conf); err != nil {
		return err
	}

	h.Config = conf
	//Load the model
	model, err := LoadOutgoingHookModelByID(db, h.WorkflowHookModelID)
	if err != nil {
		return err
	}

	h.WorkflowHookModel = *model
	h.Ref = fmt.Sprintf("%d", h.ID)

	return nil
}

func insertOutgoingTrigger(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, h sdk.WorkflowNodeOutgoingHook, trigger *sdk.WorkflowNodeOutgoingHookTrigger, u *sdk.User) error {
	trigger.WorkflowNodeOutgoingHookID = h.ID
	trigger.ID = 0

	//Setup destination node
	if errN := insertNode(db, store, w, &trigger.WorkflowDestNode, u, false); errN != nil {
		return sdk.WrapError(errN, "insertOutgoingTrigger> Unable to setup destination node")
	}
	trigger.WorkflowDestNodeID = trigger.WorkflowDestNode.ID

	//Insert trigger
	dbt := outgoingHookTrigger(*trigger)
	if err := db.Insert(&dbt); err != nil {
		return sdk.WrapError(err, "Unable to insert trigger")
	}
	trigger.ID = dbt.ID
	trigger.WorkflowDestNode.TriggerHookSrcID = trigger.ID

	// Update node trigger ID
	if err := updateWorkflowTriggerHookSrc(db, &trigger.WorkflowDestNode); err != nil { //FIX
		return sdk.WrapError(err, "Unable to update node %d for trigger %d", trigger.WorkflowDestNode.ID, trigger.ID)
	}

	return nil
}

func loadHookTrigger(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, hook *sdk.WorkflowNodeOutgoingHook, id int64, u *sdk.User, opts LoadOptions) (sdk.WorkflowNodeOutgoingHookTrigger, error) {
	var t sdk.WorkflowNodeOutgoingHookTrigger

	dbtrigger := outgoingHookTrigger{}
	//Load the trigger
	if err := db.SelectOne(&dbtrigger, "select * from workflow_node_outgoing_hook_trigger where workflow_node_outgoing_hook_id = $1 and id = $2", hook.ID, id); err != nil {
		if err == sql.ErrNoRows {
			return t, nil
		}
		return t, sdk.WrapError(err, "Unable to load trigger %d", id)
	}

	t = sdk.WorkflowNodeOutgoingHookTrigger(dbtrigger)
	//Load node destination
	if t.WorkflowDestNodeID != 0 {
		dest, err := loadNode(ctx, db, store, proj, w, t.WorkflowDestNodeID, u, opts)
		if err != nil {
			return t, sdk.WrapError(err, "Unable to load destination node %d", t.WorkflowDestNodeID)
		}
		t.WorkflowDestNode = *dest
	}

	return t, nil
}
