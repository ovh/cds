package workermodel

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

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

func getAllWithClearPassword(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	pms := []*WorkerModel{}

	if err := gorpmapping.GetAll(ctx, db, q, &pms); err != nil {
		return nil, sdk.WrapError(err, "cannot get worker models")
	}

	// Temporary decrypt password and exec post select
	for i := range pms {
		if pms[i].ModelDocker.Private && pms[i].ModelDocker.Password != "" {
			var err error
			pms[i].ModelDocker.Password, err = secret.DecryptValue(pms[i].ModelDocker.Password)
			if err != nil {
				return nil, sdk.WrapError(err, "cannot decrypt value for model with id %d", pms[i].ID)
			}
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

func getWithClearPassword(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Model, error) {
	var dbModel WorkerModel

	found, err := gorpmapping.Get(ctx, db, q, &dbModel)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get worker model")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	// Temporary decrypt password and exec post select
	if dbModel.ModelDocker.Private && dbModel.ModelDocker.Password != "" {
		dbModel.ModelDocker.Password, err = secret.DecryptValue(dbModel.ModelDocker.Password)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot decrypt value for model with id %d", dbModel.ID)
		}
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
      WHERE group_id = ANY($1)
      ORDER BY name
    `).Args(pq.Int64Array(groupIDs))
	} else {
		query = gorpmapping.NewQuery(`
      SELECT distinct worker_model.*
      FROM worker_model
      LEFT JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
      WHERE worker_model.group_id = ANY(:groupIDs)
      AND ` + filter.SQL() + `
      ORDER BY worker_model.name
    `).Args(filter.Args().Merge(gorpmapping.ArgsMap{
			"groupIDs": pq.Int64Array(groupIDs),
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
    AND group_id = ANY($2)
    ORDER BY name
  `).Args(name, pq.Int64Array(groupIDs))
	return getAll(ctx, db, query, opts...)
}

// LoadAllActiveAndNotDeprecatedForGroupIDs retrieves models for given group ids.
func LoadAllActiveAndNotDeprecatedForGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE group_id = ANY($1)
    AND is_deprecated = false
    AND disabled = false
    ORDER BY name
  `).Args(pq.Int64Array(groupIDs))
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

// LoadByIDWithClearPassword retrieves a specific worker model in database.
func LoadByIDWithClearPassword(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE id = $1
  `).Args(id)
	return getWithClearPassword(ctx, db, query, opts...)
}

// LoadByNameAndGroupIDWithClearPassword retrieves a specific worker model in database by name and group id.
func LoadByNameAndGroupIDWithClearPassword(ctx context.Context, db gorp.SqlExecutor, name string, groupID int64, opts ...LoadOptionFunc) (*sdk.Model, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE name = $1 AND group_id = $2
  `).Args(name, groupID)
	return getWithClearPassword(ctx, db, query, opts...)
}

// LoadAllUsableWithClearPasswordByGroupIDs returns usable worker models for given group ids.
func LoadAllUsableWithClearPasswordByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.Model, error) {
	// note about restricted field on worker model:
	// if restricted = true, worker model can be launched by a group hatchery only
	// so, a 'shared.infra' hatchery need all its worker models and all others with restricted = false

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model
    WHERE (
      group_id = ANY($1)
      OR (
        $2 = ANY($1)
        AND restricted = false
      )
    ) AND disabled = false
    ORDER BY name
  `).Args(pq.Int64Array(groupIDs), group.SharedInfraGroup.ID)

	return getAllWithClearPassword(ctx, db, query, opts...)
}

// Insert a new worker model in database.
func Insert(db gorp.SqlExecutor, model *sdk.Model) error {
	dbmodel := WorkerModel(*model)

	dbmodel.UserLastModified = time.Now()
	dbmodel.NeedRegistration = true

	if dbmodel.Type == sdk.Docker {
		dbmodel.ModelDocker.Envs = MergeModelEnvsWithDefaultEnvs(dbmodel.ModelDocker.Envs)
		if dbmodel.ModelDocker.Password == sdk.PasswordPlaceholder {
			return sdk.WithStack(sdk.ErrInvalidPassword)
		}
		if dbmodel.ModelDocker.Private {
			if dbmodel.ModelDocker.Password != "" {
				var err error
				dbmodel.ModelDocker.Password, err = secret.EncryptValue(dbmodel.ModelDocker.Password)
				if err != nil {
					return sdk.WrapError(err, "cannot encrypt docker password")
				}
			}
		} else {
			dbmodel.ModelDocker.Username = ""
			dbmodel.ModelDocker.Password = ""
			dbmodel.ModelDocker.Registry = ""
		}
	}

	if err := db.Insert(&dbmodel); err != nil {
		return sdk.WithStack(err)
	}

	for _, r := range dbmodel.RegisteredCapabilities {
		if err := InsertCapabilityForModelID(db, dbmodel.ID, &r); err != nil {
			return err
		}
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
	dbmodel := WorkerModel(*model)

	if err := DeleteCapabilitiesByModelID(db, dbmodel.ID); err != nil {
		return err
	}

	dbmodel.UserLastModified = time.Now()
	dbmodel.NeedRegistration = true
	dbmodel.NbSpawnErr = 0
	dbmodel.LastSpawnErr = nil
	dbmodel.LastSpawnErrLogs = nil

	if dbmodel.ModelDocker.Password == sdk.PasswordPlaceholder {
		return sdk.WithStack(sdk.ErrInvalidPassword)
	}
	if dbmodel.ModelDocker.Private {
		if dbmodel.ModelDocker.Password != "" {
			var err error
			dbmodel.ModelDocker.Password, err = secret.EncryptValue(dbmodel.ModelDocker.Password)
			if err != nil {
				return sdk.WrapError(err, "cannot encrypt docker password")
			}
		}
	} else {
		dbmodel.ModelDocker.Username = ""
		dbmodel.ModelDocker.Password = ""
		dbmodel.ModelDocker.Registry = ""
	}

	if _, err := db.Update(&dbmodel); err != nil {
		return sdk.WithStack(err)
	}

	for _, r := range dbmodel.RegisteredCapabilities {
		if err := InsertCapabilityForModelID(db, dbmodel.ID, &r); err != nil {
			return err
		}
	}

	*model = sdk.Model(dbmodel)
	if model.ModelDocker.Password != "" {
		model.ModelDocker.Password = sdk.PasswordPlaceholder
	}

	return nil
}

// DeleteByID a worker model from database and all its capabilities.
func DeleteByID(db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec("DELETE FROM worker_model WHERE id = $1", id)
	return sdk.WrapError(err, "unable to remove worker model with id %d", id)
}
