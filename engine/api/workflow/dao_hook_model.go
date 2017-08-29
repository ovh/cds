package workflow

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Here are the default hooks
var (
	WebHookModel = &sdk.WorkflowHookModel{
		Author:     "CDS",
		Type:       sdk.WorkflowHookModelBuiltin,
		Identifier: sdk.WorkflowHookModelBuiltin,
		Name:       "WebHook",
		Icon:       "",
	}

	GitPollerModel = &sdk.WorkflowHookModel{
		Author:     "CDS",
		Type:       sdk.WorkflowHookModelBuiltin,
		Identifier: sdk.WorkflowHookModelBuiltin,
		Name:       "Git Repository Poller",
		Icon:       "",
	}

	SchedulerModel = &sdk.WorkflowHookModel{
		Author:     "CDS",
		Type:       sdk.WorkflowHookModelBuiltin,
		Identifier: sdk.WorkflowHookModelBuiltin,
		Name:       "Scheduler",
		Icon:       "",
	}

	builtinModels = []*sdk.WorkflowHookModel{
		WebHookModel,
		GitPollerModel,
		SchedulerModel,
	}
)

//PostInsert is a db hook
func (r *NodeHookModel) PostInsert(db gorp.SqlExecutor) error {
	return r.PostUpdate(db)
}

//PostUpdate is a db hook
func (r *NodeHookModel) PostUpdate(db gorp.SqlExecutor) error {
	if r.DefaultConfig == nil {
		r.DefaultConfig = sdk.WorkflowNodeHookConfig{}
	}

	btes, err := json.Marshal(r.DefaultConfig)
	if err != nil {
		return err
	}
	if _, err := db.Exec("update workflow_hook_model set default_config = $2 where id = $1", r.ID, btes); err != nil {
		return err
	}
	return err
}

//PostGet is a db hook
func (r *NodeHookModel) PostGet(db gorp.SqlExecutor) error {
	confStr, err := db.SelectStr("select default_config from workflow_hook_model where id = $1", r.ID)
	if err != nil {
		return err
	}
	conf := sdk.WorkflowNodeHookConfig{}
	if err := json.Unmarshal([]byte(confStr), &conf); err != nil {
		return err
	}
	return nil
}

//CreateBuiltinWorkflowHookModels insert all builtin hook models in database
func CreateBuiltinWorkflowHookModels(db *gorp.DbMap) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("LOCK TABLE workflow_hook_model IN ACCESS EXCLUSIVE MODE"); err != nil {
		return err
	}

	for _, h := range builtinModels {
		ok, err := checkBuiltinWorkflowHookModelExist(tx, h)
		if err != nil {
			return err
		}

		if !ok {
			if err := InsertHookModel(tx, h); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func checkBuiltinWorkflowHookModelExist(db gorp.SqlExecutor, h *sdk.WorkflowHookModel) (bool, error) {
	var count = 0
	if err := db.QueryRow("select count(1), id from workflow_hook_model where name = $1 group by id", h.Name).Scan(&count, &h.ID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}

// LoadHookModels returns all hook models available
func LoadHookModels(db gorp.SqlExecutor) ([]sdk.WorkflowHookModel, error) {
	dbModels := []NodeHookModel{}
	if _, err := db.Select(&dbModels, "select id, name, type, image, command, author, description, identifier, icon from workflow_hook_model"); err != nil {
		return nil, sdk.WrapError(err, "LoadHookModels> Unable to load WorkflowHookModel")
	}
	models := []sdk.WorkflowHookModel{}
	for i := range dbModels {
		m := dbModels[i]
		if err := m.PostGet(db); err != nil {
			return nil, err
		}
		models = append(models, sdk.WorkflowHookModel(m))
	}
	return models, nil
}

// LoadHookModelByName returns a hook model by it's name, if not founc, it returns an error
func LoadHookModelByName(db gorp.SqlExecutor, name string) (*sdk.WorkflowHookModel, error) {
	m := NodeHookModel{}
	query := "select id, name, type, image, command, default_config, author, description, identifier, icon from workflow_hook_model where name = $1"
	if err := db.SelectOne(&m, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound
		}
		return nil, err
	}
	model := sdk.WorkflowHookModel(m)
	return &model, nil
}

// UpdateHookModel updates a hook model in database
func UpdateHookModel(db gorp.SqlExecutor, m *sdk.WorkflowHookModel) error {
	dbm := NodeHookModel(*m)
	if n, err := db.Update(&dbm); err != nil {
		return sdk.WrapError(err, "UpdateHookModel> Unable to update hook model %s", m.Name)
	} else if n == 0 {
		return sdk.WrapError(sdk.ErrNotFound, "UpdateHookModel> Unable to update hook model %s", m.Name)
	}
	return nil
}

// InsertHookModel inserts a hook model in database
func InsertHookModel(db gorp.SqlExecutor, m *sdk.WorkflowHookModel) error {
	dbm := NodeHookModel(*m)
	if err := db.Insert(&dbm); err != nil {
		return sdk.WrapError(err, "UpdateHookModel> Unable to insert hook model %s", m.Name)
	}
	*m = sdk.WorkflowHookModel(dbm)
	return nil
}
