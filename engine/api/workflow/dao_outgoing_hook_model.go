package workflow

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

//PostInsert is a db hook
func (r *outgoingHookModel) PostInsert(db gorp.SqlExecutor) error {
	return r.PostUpdate(db)
}

//PostUpdate is a db hook
func (r *outgoingHookModel) PostUpdate(db gorp.SqlExecutor) error {
	if r.DefaultConfig == nil {
		r.DefaultConfig = sdk.WorkflowNodeHookConfig{}
	}

	btes, errm := json.Marshal(r.DefaultConfig)
	if errm != nil {
		return errm
	}
	if _, err := db.Exec("update workflow_outgoing_hook_model set default_config = $2 where id = $1", r.ID, btes); err != nil {
		return err
	}
	return nil
}

//PostGet is a db hook
func (r *outgoingHookModel) PostGet(db gorp.SqlExecutor) error {
	confStr, err := db.SelectStr("select default_config from workflow_outgoing_hook_model where id = $1", r.ID)
	if err != nil {
		return err
	}
	conf := sdk.WorkflowNodeHookConfig{}
	if err := sdk.JSONUnmarshal([]byte(confStr), &conf); err != nil {
		return err
	}
	r.DefaultConfig = conf
	return nil
}

//CreateBuiltinWorkflowOutgoingHookModels insert all builtin hook models in database
func CreateBuiltinWorkflowOutgoingHookModels(db *gorp.DbMap) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "Unable to start transaction")
	}
	defer tx.Rollback() // nolint

	if _, err := tx.Exec("LOCK TABLE workflow_outgoing_hook_model IN ACCESS EXCLUSIVE MODE"); err != nil {
		return sdk.WrapError(err, "Unable to lock table")
	}

	for _, h := range sdk.BuiltinOutgoingHookModels {
		ok, err := checkBuiltinWorkflowOutgoingHookModelExist(tx, h)
		if err != nil {
			return sdk.WrapError(err, "CreateBuiltinWorkflowOutgoingHookModels")
		}

		if !ok {
			log.Debug(context.Background(), "CreateBuiltinWorkflowOutgoingHookModels> inserting hooks config: %s", h.Name)
			if err := InsertOutgoingHookModel(tx, h); err != nil {
				return sdk.WrapError(err, "CreateBuiltinWorkflowOutgoingHookModels error on insert")
			}
		} else {
			log.Debug(context.Background(), "CreateBuiltinWorkflowOutgoingHookModels> updating hooks config: %s", h.Name)
			// update default values
			if err := UpdateOutgoingHookModel(tx, h); err != nil {
				return sdk.WrapError(err, "CreateBuiltinWorkflowOutgoingHookModels  error on update")
			}
		}
	}
	return tx.Commit()
}

func checkBuiltinWorkflowOutgoingHookModelExist(db gorp.SqlExecutor, h *sdk.WorkflowHookModel) (bool, error) {
	var count = 0
	if err := db.QueryRow("select count(1), id from workflow_outgoing_hook_model where name = $1 group by id", h.Name).Scan(&count, &h.ID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sdk.WrapError(err, "checkBuiltinWorkflowOutgoingHookModelExist")
	}
	return count > 0, nil
}

// LoadOutgoingHookModels returns all hook models available
func LoadOutgoingHookModels(db gorp.SqlExecutor) ([]sdk.WorkflowHookModel, error) {
	dbModels := []outgoingHookModel{}
	if _, err := db.Select(&dbModels, "select id, name, type, command, author, description, identifier, icon from workflow_outgoing_hook_model"); err != nil {
		return nil, sdk.WrapError(err, "Unable to load WorkflowHookModel")
	}

	models := make([]sdk.WorkflowHookModel, len(dbModels))
	for i := range dbModels {
		m := dbModels[i]
		if err := m.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "Unable to load WorkflowHookModel")
		}
		models[i] = sdk.WorkflowHookModel(m)
	}
	return models, nil
}

// LoadOutgoingHookModelByID returns a hook model by it's id, if not found, it returns an error
func LoadOutgoingHookModelByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowHookModel, error) {
	m := outgoingHookModel{}
	query := "select id, name, type, command, default_config, author, description, identifier, icon from workflow_outgoing_hook_model where id = $1"
	if err := db.SelectOne(&m, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrNotFound, "workflow_outgoing_hook_model with id %d not found", id)
		}
		return nil, sdk.WrapError(err, "Unable to load WorkflowHookModel")
	}
	model := sdk.WorkflowHookModel(m)
	return &model, nil
}

// LoadOutgoingHookModelByName returns a hook model by it's name, if not found, it returns an error
func LoadOutgoingHookModelByName(db gorp.SqlExecutor, name string) (*sdk.WorkflowHookModel, error) {
	m := outgoingHookModel{}
	query := "select id, name, type, command, default_config, author, description, identifier, icon from workflow_outgoing_hook_model where name = $1"
	if err := db.SelectOne(&m, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrNotFound, "LoadOutgoingHookModelByName> Unable to load WorkflowHookModel")
		}
		return nil, sdk.WrapError(err, "Unable to load WorkflowHookModel")
	}
	model := sdk.WorkflowHookModel(m)
	return &model, nil
}

// UpdateOutgoingHookModel updates a hook model in database
func UpdateOutgoingHookModel(db gorp.SqlExecutor, m *sdk.WorkflowHookModel) error {
	dbm := outgoingHookModel(*m)
	if n, err := db.Update(&dbm); err != nil {
		return sdk.WrapError(err, "Unable to update hook model %s", m.Name)

	} else if n == 0 {
		return sdk.WrapError(sdk.ErrNotFound, "UpdateOutgoingHookModel> Unable to update hook model %s", m.Name)
	}
	return nil
}

// InsertOutgoingHookModel inserts a hook model in database
func InsertOutgoingHookModel(db gorp.SqlExecutor, m *sdk.WorkflowHookModel) error {
	dbm := outgoingHookModel(*m)
	if err := db.Insert(&dbm); err != nil {
		return sdk.WrapError(err, "Unable to insert hook model %s", m.Name)
	}
	*m = sdk.WorkflowHookModel(dbm)
	return nil
}
