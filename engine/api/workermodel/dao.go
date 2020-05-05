package workermodel

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	pms := []*WorkerModel{}

	if err := gorpmapping.GetAll(ctx, db, q, &pms); err != nil {
		return nil, sdk.WrapError(err, "cannot get worker models")
	}

	// Temporary hide password and exec post select
	for i := range pms {
		if err := pms[i].PostSelect(db); err != nil {
			return nil, sdk.WithStack(err)
		}
		if pms[i].ModelDocker.Password != "" {
			pms[i].ModelDocker.Password = sdk.PasswordPlaceholder
		}
	}

	var pres = make([]*sdk.Model, len(pms))
	for i := range pms {
		wm := sdk.Model(*pms[i])
		pres[i] = &wm
	}

	if len(pres) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pres...); err != nil {
				return nil, err
			}
		}
	}

	var res = make([]sdk.Model, len(pms))
	for i := range pres {
		res[i] = *pres[i]
	}

	return res, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Model, error) {
	var dbModel WorkerModel

	found, err := gorpmapping.Get(ctx, db, q, &dbModel)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get worker model")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	// Temporary hide password and exec post select
	if err := dbModel.PostSelect(db); err != nil {
		return nil, sdk.WithStack(err)
	}
	if dbModel.ModelDocker.Password != "" {
		dbModel.ModelDocker.Password = sdk.PasswordPlaceholder
	}

	model := sdk.Model(dbModel)

	for i := range opts {
		if err := opts[i](ctx, db, &model); err != nil {
			return nil, err
		}
	}

	return &model, nil
}

// LoadAll retrieves worker models from database.
func LoadAll(ctx context.Context, db gorp.SqlExecutor, filter *LoadFilter, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	var query gorpmapping.Query

	if filter == nil {
		query = gorpmapping.NewQuery("SELECT * FROM worker_model ORDER BY name")
	} else {
		query = gorpmapping.NewQuery(`
      SELECT distinct worker_model.*
      FROM worker_model
      LEFT JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
      WHERE ` + filter.SQL() + `
      ORDER BY worker_model.name
    `).Args(filter.Args())
	}

	return getAll(ctx, db, query, opts...)
}

// LoadAllByGroupIDs returns worker models list for given group ids.
func LoadAllByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, filter *LoadFilter, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	var query gorpmapping.Query

	if filter == nil {
		query = gorpmapping.NewQuery(`
      SELECT *
      FROM worker_model
      WHERE group_id = ANY(string_to_array($1, ',')::int[])
      ORDER BY name
    `).Args(gorpmapping.IDsToQueryString(groupIDs))
	} else {
		query = gorpmapping.NewQuery(`
      SELECT distinct worker_model.*
      FROM worker_model
      LEFT JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
      WHERE worker_model.group_id = ANY(string_to_array(:groupIDs, ',')::int[])
      AND ` + filter.SQL() + `
      ORDER BY worker_model.name
    `).Args(filter.Args().Merge(gorpmapping.ArgsMap{
			"groupIDs": gorpmapping.IDsToQueryString(groupIDs),
		}))
	}

	return getAll(ctx, db, query, opts...)
}

// LoadAllByNameAndGroupIDs retrieves all worker model with given name for group ids in database.
func LoadAllByNameAndGroupIDs(ctx context.Context, db gorp.SqlExecutor, name string, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE name = $1
    AND group_id = ANY(string_to_array($2, ',')::int[])
    ORDER BY name
  `).Args(name, gorpmapping.IDsToQueryString(groupIDs))
	return getAll(ctx, db, query, opts...)
}

// LoadAllActiveAndNotDeprecatedForGroupIDs retrieves models for given group ids.
func LoadAllActiveAndNotDeprecatedForGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE group_id = ANY(string_to_array($1, ',')::int[])
    AND is_deprecated = false
    AND disabled = false
    ORDER BY name
  `).Args(gorpmapping.IDsToQueryString(groupIDs))
	return getAll(ctx, db, query, opts...)
}

// LoadByID retrieves a specific worker model in database.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE id = $1
  `).Args(id)
	return get(ctx, db, query, opts...)
}

// LoadByNameAndGroupID retrieves a specific worker model in database by name and group id.
func LoadByNameAndGroupID(ctx context.Context, db gorp.SqlExecutor, name string, groupID int64, opts ...LoadOptionFunc) (*sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE name = $1 AND group_id = $2
  `).Args(name, groupID)
	return get(ctx, db, query, opts...)
}

// loadAll retrieves a list of worker model in database.
func loadAll(db gorp.SqlExecutor, withPassword bool, query string, args ...interface{}) ([]sdk.Model, error) {
	wms := []dbResultWMS{}
	if _, err := db.Select(&wms, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, sdk.WithStack(err)
	}
	if len(wms) == 0 {
		return []sdk.Model{}, nil
	}
	r, err := scanAll(db, wms, withPassword)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// load retrieves a specific worker model in database.
func load(db gorp.SqlExecutor, withPassword bool, query string, args ...interface{}) (*sdk.Model, error) {
	wms := []dbResultWMS{}
	if _, err := db.Select(&wms, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}
		return nil, err
	}
	if len(wms) == 0 {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	r, err := scanAll(db, wms, withPassword)
	if err != nil {
		return nil, err
	}
	if len(r) != 1 {
		return nil, sdk.WithStack(fmt.Errorf("worker model not unique"))
	}
	return &r[0], nil
}

func scanAll(db gorp.SqlExecutor, rows []dbResultWMS, withPassword bool) ([]sdk.Model, error) {
	models := []sdk.Model{}
	for _, row := range rows {
		m := row.WorkerModel
		m.Group = &sdk.Group{ID: m.GroupID, Name: row.GroupName}
		if err := m.PostSelect(db); err != nil {
			return nil, sdk.WithStack(err)
		}
		if m.ModelDocker.Password != "" && !withPassword {
			m.ModelDocker.Password = sdk.PasswordPlaceholder
		}
		models = append(models, sdk.Model(m))
	}
	// as we can't use order by name with sql union without alias, sort models here
	sort.Slice(models, func(i, j int) bool {
		return models[i].Name <= models[j].Name
	})
	return models, nil
}

const modelColumns = `
	DISTINCT worker_model.id,
	worker_model.type,
	worker_model.name,
	worker_model.image,
	worker_model.description,
	worker_model.group_id,
	worker_model.last_registration,
	worker_model.need_registration,
	worker_model.disabled,
	worker_model.template,
	worker_model.communication,
	worker_model.run_script,
	worker_model.restricted,
	worker_model.user_last_modified,
	worker_model.last_spawn_err,
	worker_model.last_spawn_err_log,
	worker_model.nb_spawn_err,
	worker_model.date_last_spawn_err,
	worker_model.is_deprecated,
	"group".name as groupname`

// LoadByIDWithClearPassword retrieves a specific worker model in database.
func LoadByIDWithClearPassword(db gorp.SqlExecutor, id int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" on worker_model.group_id = "group".id
    WHERE worker_model.id = $1
  `, modelColumns)
	model, err := load(db, true, query, id)
	if err != nil {
		return nil, err
	}

	if model.ModelDocker.Private && model.ModelDocker.Password != "" {
		var err error
		model.ModelDocker.Password, err = secret.DecryptValue(model.ModelDocker.Password)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot decrypt value for model %s", fmt.Sprintf("%s/%s", model.Group.Name, model.Name))
		}
	}

	return model, nil
}

// LoadByNameAndGroupIDWithClearPassword retrieves a specific worker model in database by name and group id.
func LoadByNameAndGroupIDWithClearPassword(db gorp.SqlExecutor, name string, groupID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    WHERE worker_model.name = $1 AND worker_model.group_id = $2
  `, modelColumns)
	return load(db, true, query, name, groupID)
}

// LoadAllUsableWithClearPasswordByGroupIDs returns usable worker models for given group ids.
func LoadAllUsableWithClearPasswordByGroupIDs(db gorp.SqlExecutor, groupIDs []int64) ([]sdk.Model, error) {
	// note about restricted field on worker model:
	// if restricted = true, worker model can be launched by a group hatchery only
	// so, a 'shared.infra' hatchery need all its worker models and all others with restricted = false

	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    WHERE (
      worker_model.group_id = ANY(string_to_array($1, ',')::int[])
      OR (
        $2 = ANY(string_to_array($1, ',')::int[])
        AND worker_model.restricted = false
      )
    ) AND worker_model.disabled = false
    ORDER BY name
  `, modelColumns)
	models, err := loadAll(db, true, query, gorpmapping.IDsToQueryString(groupIDs), group.SharedInfraGroup.ID)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	for i := range models {
		if models[i].ModelDocker.Private && models[i].ModelDocker.Password != "" {
			var err error
			models[i].ModelDocker.Password, err = secret.DecryptValue(models[i].ModelDocker.Password)
			if err != nil {
				return nil, sdk.WrapError(err, "cannot decrypt value for model %s", fmt.Sprintf("%s/%s", models[i].Group.Name, models[i].Name))
			}
		}
	}

	return models, nil
}

// Insert a new worker model in database.
func Insert(db gorp.SqlExecutor, model *sdk.Model) error {
	dbmodel := WorkerModel(*model)
	dbmodel.NeedRegistration = true
	model.UserLastModified = time.Now()
	if model.ModelDocker.Password == sdk.PasswordPlaceholder {
		return sdk.WithStack(sdk.ErrInvalidPassword)
	}
	if err := db.Insert(&dbmodel); err != nil {
		return sdk.WithStack(err)
	}
	*model = sdk.Model(dbmodel)
	if model.ModelDocker.Password != "" {
		model.ModelDocker.Password = sdk.PasswordPlaceholder
	}
	return nil
}

// UpdateDB a worker model
// if the worker model have SpawnErr -> clear them.
func UpdateDB(db gorp.SqlExecutor, model *sdk.Model) error {
	model.UserLastModified = time.Now()
	model.NeedRegistration = true
	model.NbSpawnErr = 0
	model.LastSpawnErr = ""
	model.LastSpawnErrLogs = nil
	dbmodel := WorkerModel(*model)
	if _, err := db.Update(&dbmodel); err != nil {
		return sdk.WithStack(err)
	}
	*model = sdk.Model(dbmodel)
	if model.ModelDocker.Password != "" {
		model.ModelDocker.Password = sdk.PasswordPlaceholder
	}
	return nil
}

// Delete a worker model from database and all its capabilities.
func Delete(db gorp.SqlExecutor, ID int64) error {
	m := WorkerModel(sdk.Model{ID: ID})
	count, err := db.Delete(&m)
	if err != nil {
		return sdk.WithStack(err)
	}
	if count == 0 {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	return nil
}

// LoadCapabilities retrieves capabilities of given worker model.
func LoadCapabilities(db gorp.SqlExecutor, workerID int64) (sdk.RequirementList, error) {
	query := `
    SELECT name, type, argument
    FROM worker_capability
    WHERE worker_model_id = $1
    ORDER BY name
  `

	rows, err := db.Query(query, workerID)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer rows.Close()

	var capas sdk.RequirementList
	for rows.Next() {
		var c sdk.Requirement
		if err := rows.Scan(&c.Name, &c.Type, &c.Value); err != nil {
			return nil, sdk.WithStack(err)
		}
		capas = append(capas, c)
	}
	return capas, nil
}
