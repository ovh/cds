package worker

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
	worker_model.nb_spawn_err,
	worker_model.date_last_spawn_err,
	worker_model.is_deprecated,
	"group".name as groupname`

const bookRegisterTTLInSeconds = 360

var defaultEnvs = map[string]string{
	"CDS_SINGLE_USE":          "1",
	"CDS_TTL":                 "{{.TTL}}",
	"CDS_GRAYLOG_HOST":        "{{.GraylogHost}}",
	"CDS_GRAYLOG_PORT":        "{{.GraylogPort}}",
	"CDS_GRAYLOG_EXTRA_KEY":   "{{.GraylogExtraKey}}",
	"CDS_GRAYLOG_EXTRA_VALUE": "{{.GraylogExtraValue}}",
}

type dbResultWMS struct {
	WorkerModel
	GroupName string `db:"groupname"`
}

// InsertWorkerModel insert a new worker model in database
func InsertWorkerModel(db gorp.SqlExecutor, model *sdk.Model) error {
	dbmodel := WorkerModel(*model)
	if err := db.Insert(&dbmodel); err != nil {
		return err
	}
	*model = sdk.Model(dbmodel)
	return nil
}

// UpdateWorkerModel update a worker model. If worker model have SpawnErr -> clear them
func UpdateWorkerModel(db gorp.SqlExecutor, model *sdk.Model) error {
	model.UserLastModified = time.Now()
	model.NeedRegistration = true
	model.NbSpawnErr = 0
	model.LastSpawnErr = ""
	dbmodel := WorkerModel(*model)
	if _, err := db.Update(&dbmodel); err != nil {
		return err
	}
	*model = sdk.Model(dbmodel)
	return nil
}

// UpdateWorkerModelWithoutRegistration update a worker model. If worker model have SpawnErr -> clear them
func UpdateWorkerModelWithoutRegistration(db gorp.SqlExecutor, model sdk.Model) error {
	model.UserLastModified = time.Now()
	model.NeedRegistration = false
	model.NbSpawnErr = 0
	model.LastSpawnErr = ""
	dbmodel := WorkerModel(model)
	if _, err := db.Update(&dbmodel); err != nil {
		return err
	}
	return nil
}

// LoadWorkerModels retrieves models from database
func LoadWorkerModels(db gorp.SqlExecutor) ([]sdk.Model, error) {
	wms := []dbResultWMS{}
	query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id order by worker_model.name`, modelColumns)
	if _, err := db.Select(&wms, query); err != nil {
		return nil, sdk.WrapError(err, "LoadAllWorkerModels> ")
	}
	return scanWorkerModels(db, wms)
}

// loadWorkerModel retrieves a specific worker model in database
func loadWorkerModel(db gorp.SqlExecutor, query string, args ...interface{}) (*sdk.Model, error) {
	wms := []dbResultWMS{}
	if _, err := db.Select(&wms, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoWorkerModel
		}
		return nil, err
	}
	if len(wms) == 0 {
		return nil, sdk.ErrNoWorkerModel
	}
	r, err := scanWorkerModels(db, wms)
	if err != nil {
		return nil, err
	}
	if len(r) != 1 {
		return nil, fmt.Errorf("worker model not unique")
	}
	return &r[0], nil
}

// LoadWorkerModelByName retrieves a specific worker model in database
func LoadWorkerModelByName(db gorp.SqlExecutor, name string) (*sdk.Model, error) {
	query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id and worker_model.name = $1`, modelColumns)
	return loadWorkerModel(db, query, name)
}

// LoadWorkerModelByID retrieves a specific worker model in database
func LoadWorkerModelByID(db gorp.SqlExecutor, ID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id and worker_model.id = $1`, modelColumns)
	return loadWorkerModel(db, query, ID)
}

// LoadAndLockWorkerModelByID retrieves a specific worker model in database
func LoadAndLockWorkerModelByID(db gorp.SqlExecutor, ID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id and worker_model.id = $1 FOR UPDATE NOWAIT`, modelColumns)
	return loadWorkerModel(db, query, ID)
}

// LoadWorkerModelsByUser returns worker models list according to user's groups
func LoadWorkerModelsByUser(db gorp.SqlExecutor, user *sdk.User) ([]sdk.Model, error) {
	wms := []dbResultWMS{}
	if user.Admin {
		query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id`, modelColumns)
		if _, err := db.Select(&wms, query); err != nil {
			return nil, sdk.WrapError(err, "LoadWorkerModelsByUser> for admin")
		}
	} else {
		query := fmt.Sprintf(`select %s
					from worker_model
					JOIN "group" on worker_model.group_id = "group".id
					where group_id in (select group_id from group_user where user_id = $1)
					union
					select %s from worker_model
					JOIN "group" on worker_model.group_id = "group".id
					where group_id = $2`, modelColumns, modelColumns)
		if _, err := db.Select(&wms, query, user.ID, group.SharedInfraGroup.ID); err != nil {
			return nil, sdk.WrapError(err, "LoadWorkerModelsByUser> for user")
		}
	}
	return scanWorkerModels(db, wms)
}

// LoadWorkerModelsByUserAndBinary returns worker models list according to user's groups and binary capability
func LoadWorkerModelsByUserAndBinary(db gorp.SqlExecutor, user *sdk.User, binary string) ([]sdk.Model, error) {
	wms := []dbResultWMS{}
	if user.Admin {
		query := fmt.Sprintf(`
			SELECT %s
				FROM worker_model
					JOIN "group" ON worker_model.group_id = "group".id
					JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
					WHERE worker_capability.type = 'binary' AND worker_capability.argument = $1
		`, modelColumns)
		if _, err := db.Select(&wms, query, binary); err != nil {
			return nil, sdk.WrapError(err, "LoadWorkerModelsByUserAndBinary> for admin")
		}
	} else {
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
		if _, err := db.Select(&wms, query, user.ID, group.SharedInfraGroup.ID, binary); err != nil {
			return nil, sdk.WrapError(err, "LoadWorkerModelsByUserAndBinary> for user")
		}
	}
	return scanWorkerModels(db, wms)
}

func scanWorkerModels(db gorp.SqlExecutor, rows []dbResultWMS) ([]sdk.Model, error) {
	models := []sdk.Model{}
	for _, row := range rows {
		m := row.WorkerModel
		m.Group = sdk.Group{ID: m.GroupID, Name: row.GroupName}
		if err := m.PostSelect(db); err != nil {
			return nil, err
		}
		models = append(models, sdk.Model(m))
	}
	// as we can't use order by name with sql union without alias, sort models here
	sort.Slice(models, func(i, j int) bool {
		return models[i].Name <= models[j].Name
	})
	return models, nil
}

// DeleteWorkerModel removes from database worker model informations and all its capabilities
func DeleteWorkerModel(db gorp.SqlExecutor, ID int64) error {
	m := WorkerModel(sdk.Model{ID: ID})
	count, err := db.Delete(&m)
	if err != nil {
		return err
	}
	if count == 0 {
		return sdk.ErrNoWorkerModel
	}
	return nil
}

// LoadWorkerModelCapabilities retrieves capabilities of given worker model
func LoadWorkerModelCapabilities(db gorp.SqlExecutor, workerID int64) (sdk.RequirementList, error) {
	query := `SELECT name, type, argument FROM worker_capability WHERE worker_model_id = $1 ORDER BY name`

	rows, err := db.Query(query, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var capas sdk.RequirementList
	for rows.Next() {
		var c sdk.Requirement
		if err := rows.Scan(&c.Name, &c.Type, &c.Value); err != nil {
			return nil, err
		}
		capas = append(capas, c)
	}
	return capas, nil
}

// ComputeRegistrationNeeds checks if worker models need to be register
// if requirements contains "binary" type: all workers model need to be registered again by
// setting flag need_registration to true in DB.
func ComputeRegistrationNeeds(db gorp.SqlExecutor, allBinaryReqs sdk.RequirementList, reqs sdk.RequirementList) error {
	log.Debug("ComputeRegistrationNeeds>")
	var nbModelReq int
	var nbOSArchReq int
	var nbHostnameReq int

	for _, r := range reqs {
		switch r.Type {
		case sdk.BinaryRequirement:
			exist := false
			for _, e := range allBinaryReqs {
				if e.Value == r.Value {
					exist = true
					break
				}
			}
			if !exist {
				return updateAllToNeedRegistration(db)
			}
		case sdk.OSArchRequirement:
			nbOSArchReq++
		case sdk.ModelRequirement:
			nbModelReq++
		case sdk.HostnameRequirement:
			nbHostnameReq++
		}
	}

	if nbOSArchReq > 1 {
		return sdk.NewError(sdk.ErrWrongRequest, errors.New("invalid os-architecture requirement usage"))
	}
	if nbModelReq > 1 {
		return sdk.NewError(sdk.ErrWrongRequest, errors.New("invalid model requirement usage"))
	}
	if nbHostnameReq > 1 {
		return sdk.NewError(sdk.ErrWrongRequest, errors.New("invalid hostname requirement usage"))
	}

	return nil
}

func updateAllToNeedRegistration(db gorp.SqlExecutor) error {
	query := `UPDATE worker_model SET need_registration = $1`
	res, err := db.Exec(query, true)
	if err != nil {
		return sdk.WrapError(err, "updateAllToNeedRegistration>")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "updateAllToNeedRegistration>")
	}
	log.Debug("updateAllToNeedRegistration> %d worker model(s) need registration", rows)
	return nil
}

// UpdateSpawnErrorWorkerModel updates worker model error registration
func UpdateSpawnErrorWorkerModel(db gorp.SqlExecutor, modelID int64, info string) error {
	query := `UPDATE worker_model SET nb_spawn_err=nb_spawn_err+1, last_spawn_err=$1, date_last_spawn_err=$2 WHERE id = $3`
	res, err := db.Exec(query, info, time.Now(), modelID)
	if err != nil {
		return sdk.WrapError(err, "UpdateSpawnErrorWorkerModel>")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "UpdateSpawnErrorWorkerModel>")
	}
	log.Debug("UpdateSpawnErrorWorkerModel> %d worker model updated", rows)
	return nil
}

// updateRegistration updates need_registration to false and last_registration time, reset err registration
func updateRegistration(db gorp.SqlExecutor, modelID int64) error {
	query := `UPDATE worker_model SET need_registration=$1, last_registration = $2, nb_spawn_err=$3, last_spawn_err=$4 WHERE id = $5`
	res, err := db.Exec(query, false, time.Now(), 0, "", modelID)
	if err != nil {
		return sdk.WrapError(err, "updateRegistration>")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "updateRegistration>")
	}
	log.Debug("updateRegistration> %d worker model updated", rows)
	return nil
}

// updateOSAndArch updates os and arch for a worker model
func updateOSAndArch(db gorp.SqlExecutor, modelID int64, OS, arch string) error {
	query := `UPDATE worker_model SET registered_os=$1, registered_arch = $2 WHERE id = $3`
	res, err := db.Exec(query, OS, arch, modelID)
	if err != nil {
		return sdk.WrapError(err, "updateOSAndArch>")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "updateOSAndArch>")
	}
	log.Debug("updateOSAndArch> %d worker model updated", rows)
	return nil
}

func keyBookWorkerModel(id int64) string {
	return cache.Key("book", "workermodel", strconv.FormatInt(id, 10))
}

// BookForRegister books a worker model for register, used by hatcheries
func BookForRegister(store cache.Store, id int64, hatchery *sdk.Hatchery) (*sdk.Hatchery, error) {
	k := keyBookWorkerModel(id)
	h := sdk.Hatchery{}
	if !store.Get(k, &h) {
		// worker model not already booked, book it for 6 min
		store.SetWithTTL(k, hatchery, bookRegisterTTLInSeconds)
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrWorkerModelAlreadyBooked, "BookForRegister> worker model %d already booked by %s (%d)", id, h.Name, h.ID)
}

func mergeWithDefaultEnvs(envs map[string]string) map[string]string {
	if envs == nil {
		return defaultEnvs
	}
	for envName := range defaultEnvs {
		if _, ok := envs[envName]; !ok {
			envs[envName] = defaultEnvs[envName]
		}
	}

	return envs
}
