package workermodel

import (
	"fmt"
	"sort"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

func getAll(db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	pms := []*sdk.Model{}

	if err := gorpmapping.GetAll(db, q, &pms); err != nil {
		return nil, sdk.WrapError(err, "cannot get worker models")
	}
	if len(pms) > 0 {
		for i := range opts {
			if err := opts[i](db, pms...); err != nil {
				return nil, err
			}
		}
	}

	// TODO refactor data model to remove post select
	for i := range pms {
		wm := WorkerModel(*pms[i])
		if err := wm.PostSelect(db); err != nil {
			return nil, sdk.WithStack(err)
		}
		if wm.ModelDocker.Password != "" {
			wm.ModelDocker.Password = sdk.PasswordPlaceholder
		}
		*pms[i] = sdk.Model(wm)
	}

	ms := make([]sdk.Model, len(pms))
	for i := range ms {
		ms[i] = *pms[i]
	}

	return ms, nil
}

func get(db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Model, error) {
	var m sdk.Model

	found, err := gorpmapping.Get(db, q, &m)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get worker model")
	}
	if !found {
		return nil, nil
	}

	for i := range opts {
		if err := opts[i](db, &m); err != nil {
			return nil, err
		}
	}

	return &m, nil
}

// LoadAll retrieves worker models from database.
func LoadAll(db gorp.SqlExecutor, filter *LoadFilter, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	var query gorpmapping.Query

	if filter == nil {
		query = gorpmapping.NewQuery("SELECT * FROM worker_model ORDER BY name")
	} else {
		query = gorpmapping.NewQuery(`
      SELECT worker_model.*
      FROM worker_model
      LEFT JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
      WHERE ` + filter.SQL() + `
      ORDER BY worker_model.name
    `).Args(filter.Args())
	}

	return getAll(db, query, opts...)
}

// LoadAllByGroupIDs returns worker models list for given group ids.
func LoadAllByGroupIDs(db gorp.SqlExecutor, groupIDs []int64, filter *LoadFilter, opts ...LoadOptionFunc) ([]sdk.Model, error) {
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
      SELECT worker_model.*
      FROM worker_model
      LEFT JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
      WHERE worker_model.group_id = ANY(string_to_array(:groupIDs, ',')::int[])
      AND ` + filter.SQL() + `
      ORDER BY worker_model.name
    `).Args(filter.Args().Merge(gorpmapping.ArgsMap{
			"groupIDs": gorpmapping.IDsToQueryString(groupIDs),
		}))
	}

	return getAll(db, query, opts...)
}

// LoadAllNotSharedInfra retrieves models not shared infra from database.
func LoadAllNotSharedInfra(db gorp.SqlExecutor, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE group_id != $1
    ORDER BY name
  `).Args(group.SharedInfraGroup.ID)
	return getAll(db, query, opts...)
}

// LoadAllByNameAndGroupIDs retrieves all worker model with given name for group ids in database.
func LoadAllByNameAndGroupIDs(db gorp.SqlExecutor, name string, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE name = $1
    AND group_id = ANY(string_to_array($2, ',')::int[])
    ORDER BY name
  `).Args(name, gorpmapping.IDsToQueryString(groupIDs))
	return getAll(db, query, opts...)
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
	worker_model.provision,
	worker_model.restricted,
	worker_model.user_last_modified,
	worker_model.last_spawn_err,
	worker_model.last_spawn_err_log,
	worker_model.nb_spawn_err,
	worker_model.date_last_spawn_err,
	worker_model.is_deprecated,
	"group".name as groupname`

// LoadByID retrieves a specific worker model in database.
func LoadByID(db gorp.SqlExecutor, id int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" on worker_model.group_id = "group".id
    WHERE worker_model.id = $1
  `, modelColumns)
	return load(db, false, query, id)
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

// LoadByNameAndGroupID retrieves a specific worker model in database by name and group id.
func LoadByNameAndGroupID(db gorp.SqlExecutor, name string, groupID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    WHERE worker_model.name = $1 AND worker_model.group_id = $2
  `, modelColumns)
	return load(db, false, query, name, groupID)
}

// LoadAllActiveAndNotDeprecatedForGroupIDs retrieves models for given group ids.
func LoadAllActiveAndNotDeprecatedForGroupIDs(db gorp.SqlExecutor, groupIDs []int64) ([]sdk.Model, error) {
	query := `
    SELECT worker_model.*, "group".name as groupname
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    WHERE worker_model.group_id = ANY(string_to_array($1, ',')::int[])
    AND worker_model.is_deprecated = false
    AND worker_model.disabled = false
    ORDER BY worker_model.name
  `
	return loadAll(db, false, query, gorpmapping.IDsToQueryString(groupIDs))
}

// LoadAllByBinaryAndGroupIDs returns worker models list with given binary capability for group ids.
func LoadAllByBinaryAndGroupIDs(db gorp.SqlExecutor, binary string, groupIDs []int64) ([]sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
    WHERE worker_model.group_id = ANY(string_to_array($1, ',')::int[])
    AND worker_capability.type = 'binary'
    AND worker_capability.argument = $2
  `, modelColumns)
	return loadAll(db, false, query, gorpmapping.IDsToQueryString(groupIDs), binary)
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
        $2 = ANY(string_to_array($1, ',')::int[]
        AND worker_model.restricted = false
      )
    )
    AND worker_model.disabled = false
    ORDER by name
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

// LoadAllByBinary returns worker models list with given binary capability.
func LoadAllByBinary(db gorp.SqlExecutor, binary string) ([]sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
    WHERE worker_capability.type = 'binary'
    AND worker_capability.argument = $1
  `, modelColumns)
	return loadAll(db, false, query, binary)
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

// UpdateWithoutRegistration update a worker model
// if the worker model have SpawnErr -> clear them.
func UpdateWithoutRegistration(db gorp.SqlExecutor, model sdk.Model) error {
	model.UserLastModified = time.Now()
	model.NeedRegistration = false
	model.NbSpawnErr = 0
	model.LastSpawnErr = ""
	model.LastSpawnErrLogs = nil
	dbmodel := WorkerModel(model)
	if _, err := db.Update(&dbmodel); err != nil {
		return sdk.WithStack(err)
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
		return sdk.WithStack(sdk.ErrNoWorkerModel)
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
