package workermodel

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

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

// LoadAll retrieves worker models from database.
func LoadAll(db gorp.SqlExecutor) ([]sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    ORDER BY worker_model.name
  `, modelColumns)
	return loadAll(db, false, query)
}

// LoadAllNotSharedInfra retrieves models not shared infra from database.
func LoadAllNotSharedInfra(db gorp.SqlExecutor) ([]sdk.Model, error) {
	query := fmt.Sprintf(`
    SELECT %s
    FROM worker_model JOIN "group" ON worker_model.group_id = "group".id
    WHERE worker_model.group_id != $1
    ORDER BY worker_model.name
  `, modelColumns)
	return loadAll(db, false, query, group.SharedInfraGroup.ID)
}

// LoadAllByNameAndGroupIDs retrieves all worker model with given name for group ids in database.
func LoadAllByNameAndGroupIDs(db gorp.SqlExecutor, name string, groupIDs []int64) ([]sdk.Model, error) {
	query := `
    SELECT worker_model.*, "group".name as groupname
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    WHERE worker_model.name = $1
    AND worker_model.group_id = ANY(string_to_array($2, ',')::int[])
  `
	return loadAll(db, false, query, name, gorpmapping.IDsToQueryString(groupIDs))
}

// LoadByID retrieves a specific worker model in database.
func LoadByID(db gorp.SqlExecutor, ID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id and worker_model.id = $1`, modelColumns)
	return load(db, false, query, ID)
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
  `
	return loadAll(db, false, query, gorpmapping.IDsToQueryString(groupIDs))
}

// LoadAllByUser returns worker models list according to user's groups
func LoadAllByUser(db gorp.SqlExecutor, store cache.Store, user *sdk.User, opts *StateLoadOption) ([]sdk.Model, error) {
	prefixKey := "api:workermodels"

	if opts != nil {
		prefixKey += fmt.Sprintf(":%v", *opts)
	}
	key := cache.Key(prefixKey, user.Username)
	models := []sdk.Model{}
	if store.Get(key, &models) {
		return models, nil
	}

	additionalFilters := getAdditionalSQLFilters(opts)
	var query string
	var args []interface{}
	if user.Admin {
		query = fmt.Sprintf(`
      SELECT %s
      FROM worker_model
      JOIN "group" ON worker_model.group_id = "group".id
    `, modelColumns)
		if len(additionalFilters) > 0 {
			query += fmt.Sprintf(" WHERE %s", strings.Join(additionalFilters, " AND "))
		}
	} else {
		query = fmt.Sprintf(`
      SELECT %s
			  FROM worker_model
			  JOIN "group" ON worker_model.group_id = "group".id
			  WHERE group_id IN (
          SELECT group_id
          FROM group_user
          WHERE user_id = $1
        )
			UNION
      SELECT %s
        FROM worker_model
			  JOIN "group" on worker_model.group_id = "group".id
        WHERE group_id = $2
    `, modelColumns, modelColumns)
		if len(additionalFilters) > 0 {
			query += fmt.Sprintf(" AND %s", strings.Join(additionalFilters, " AND "))
		}

		args = []interface{}{user.ID, group.SharedInfraGroup.ID}
	}
	models, err := loadAll(db, false, query, args...)
	if err != nil {
		return nil, err
	}

	store.SetWithTTL(key, models, CacheTTLInSeconds)
	return models, nil
}

// LoadAllUsableOnGroupWithClearPassword returns worker models for a group.
func LoadAllUsableOnGroupWithClearPassword(db gorp.SqlExecutor, store cache.Store, groupID int64) ([]sdk.Model, error) {
	key := cache.Key("api:workermodels:bygroup", fmt.Sprintf("%d", groupID))

	models := make([]sdk.Model, 0)
	if store.Get(key, &models) {
		return models, nil
	}

	// note about restricted field on worker model:
	// if restricted = true, worker model can be launched by a user hatchery only
	// so, a 'shared.infra' hatchery need all its worker models and all others with restricted = false

	var query string
	if groupID == group.SharedInfraGroup.ID { // shared infra, return all models, excepts restricted
		query = fmt.Sprintf(`
      SELECT %s
      FROM worker_model
      JOIN "group" ON worker_model.group_id = "group".id
      WHERE (worker_model.restricted = false OR worker_model.group_id = $1)
      AND worker_model.disabled = false
      ORDER by name
    `, modelColumns)
	} else { // not shared infra, returns only selected worker models
		query = fmt.Sprintf(`
      SELECT %s
      FROM worker_model
      JOIN "group" ON worker_model.group_id = "group".id
      WHERE worker_model.group_id = $1
      AND worker_model.disabled = false
      ORDER by name
    `, modelColumns)
	}
	models, err := loadAll(db, true, query, groupID)
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

	store.SetWithTTL(key, models, CacheTTLInSeconds)

	return models, nil
}

// LoadAllByUserAndBinary returns worker models list according to user's groups and binary capability
func LoadAllByUserAndBinary(db gorp.SqlExecutor, user *sdk.User, binary string) ([]sdk.Model, error) {
	if user.Admin {
		query := fmt.Sprintf(`
			SELECT %s
				FROM worker_model
					JOIN "group" ON worker_model.group_id = "group".id
					JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
					WHERE worker_capability.type = 'binary' AND worker_capability.argument = $1
    `, modelColumns)
		return loadAll(db, false, query, binary)
	}

	query := fmt.Sprintf(`
    SELECT %s
      FROM worker_model
        JOIN "group" ON worker_model.group_id = "group".id
        JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
      WHERE group_id IN (SELECT group_id FROM group_user WHERE user_id = $1) AND worker_capability.type = 'binary' AND worker_capability.argument = $3
    UNION
    SELECT %s
      FROM worker_model
        JOIN "group" ON worker_model.group_id = "group".id
        JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
      WHERE group_id = $2 AND worker_capability.type = 'binary' AND worker_capability.argument = $3
  `, modelColumns, modelColumns)
	return loadAll(db, false, query, user.ID, group.SharedInfraGroup.ID, binary)
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
	query := `SELECT name, type, argument FROM worker_capability WHERE worker_model_id = $1 ORDER BY name`

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
