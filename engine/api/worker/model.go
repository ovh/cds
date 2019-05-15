package worker

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
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
	worker_model.last_spawn_err_log,
	worker_model.nb_spawn_err,
	worker_model.date_last_spawn_err,
	worker_model.is_deprecated,
	"group".name as groupname`

const (
	bookRegisterTTLInSeconds = 360
	modelsCacheTTLInSeconds  = 30
)

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

// StateLoadOption represent load options to load worker model
type StateLoadOption string

func (s StateLoadOption) String() string {
	return string(s)
}

// IsValid returns an error if the state value is not valid.
func (s StateLoadOption) IsValid() error {
	switch s {
	case StateDisabled, StateOfficial, StateError, StateRegister, StateDeprecated, StateActive:
		return nil
	default:
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given state value")
	}
}

// List of const for state load option
const (
	StateError      StateLoadOption = "error"
	StateDisabled   StateLoadOption = "disabled"
	StateRegister   StateLoadOption = "register"
	StateDeprecated StateLoadOption = "deprecated"
	StateActive     StateLoadOption = "active"
	StateOfficial   StateLoadOption = "official"
)

// InsertWorkerModel insert a new worker model in database
func InsertWorkerModel(db gorp.SqlExecutor, model *sdk.Model) error {
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

// UpdateWorkerModel update a worker model. If worker model have SpawnErr -> clear them
func UpdateWorkerModel(db gorp.SqlExecutor, model *sdk.Model) error {
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

// UpdateWorkerModelWithoutRegistration update a worker model. If worker model have SpawnErr -> clear them
func UpdateWorkerModelWithoutRegistration(db gorp.SqlExecutor, model sdk.Model) error {
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

// LoadWorkerModels retrieves models from database.
func LoadWorkerModels(db gorp.SqlExecutor) ([]sdk.Model, error) {
	query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id order by worker_model.name`, modelColumns)
	return loadWorkerModels(db, false, query)
}

// LoadWorkerModelsNotSharedInfra retrieves models not shared infra from database.
func LoadWorkerModelsNotSharedInfra(db gorp.SqlExecutor) ([]sdk.Model, error) {
	query := fmt.Sprintf(`SELECT %s FROM worker_model JOIN "group" ON worker_model.group_id = "group".id WHERE worker_model.group_id != $1 ORDER BY worker_model.name`, modelColumns)
	return loadWorkerModels(db, false, query, group.SharedInfraGroup.ID)
}

// loadWorkerModels retrieves a list of worker model in database.
func loadWorkerModels(db gorp.SqlExecutor, withPassword bool, query string, args ...interface{}) ([]sdk.Model, error) {
	wms := []dbResultWMS{}
	if _, err := db.Select(&wms, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNoWorkerModel)
		}
		return nil, sdk.WithStack(err)
	}
	if len(wms) == 0 {
		return []sdk.Model{}, nil
	}
	r, err := scanWorkerModels(db, wms, withPassword)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// loadWorkerModel retrieves a specific worker model in database.
func loadWorkerModel(db gorp.SqlExecutor, withPassword bool, query string, args ...interface{}) (*sdk.Model, error) {
	wms := []dbResultWMS{}
	if _, err := db.Select(&wms, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNoWorkerModel)
		}
		return nil, err
	}
	if len(wms) == 0 {
		return nil, sdk.WithStack(sdk.ErrNoWorkerModel)
	}
	r, err := scanWorkerModels(db, wms, withPassword)
	if err != nil {
		return nil, err
	}
	if len(r) != 1 {
		return nil, sdk.WithStack(fmt.Errorf("worker model not unique"))
	}
	return &r[0], nil
}

// LoadWorkerModelsByNameAndGroupIDs retrieves all worker model with given name for group ids in database.
func LoadWorkerModelsByNameAndGroupIDs(db gorp.SqlExecutor, name string, groupIDs []int64) ([]sdk.Model, error) {
	query := `
    SELECT worker_model.*, "group".name as groupname
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    WHERE worker_model.name = $1
    AND worker_model.group_id = ANY(string_to_array($2, ',')::int[])
  `
	return loadWorkerModels(db, false, query, name, gorpmapping.IDsToQueryString(groupIDs))
}

// LoadWorkerModelByID retrieves a specific worker model in database.
func LoadWorkerModelByID(db gorp.SqlExecutor, ID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id and worker_model.id = $1`, modelColumns)
	return loadWorkerModel(db, false, query, ID)
}

// LoadWorkerModelByNameAndGroupIDWithClearPassword retrieves a specific worker model in database by name and group id.
func LoadWorkerModelByNameAndGroupIDWithClearPassword(db gorp.SqlExecutor, name string, groupID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`SELECT %s FROM worker_model JOIN "group" ON worker_model.group_id = "group".id AND worker_model.name = $1 AND worker_model.group_id = $2`, modelColumns)
	return loadWorkerModel(db, true, query, name, groupID)
}

// LoadWorkerModelByNameAndGroupID retrieves a specific worker model in database by name and group id.
func LoadWorkerModelByNameAndGroupID(db gorp.SqlExecutor, name string, groupID int64) (*sdk.Model, error) {
	query := fmt.Sprintf(`SELECT %s FROM worker_model JOIN "group" ON worker_model.group_id = "group".id AND worker_model.name = $1 AND worker_model.group_id = $2`, modelColumns)
	return loadWorkerModel(db, false, query, name, groupID)
}

// LoadWorkerModelsActiveAndNotDeprecatedForGroupIDs retrieves models for given group ids.
func LoadWorkerModelsActiveAndNotDeprecatedForGroupIDs(db gorp.SqlExecutor, groupIDs []int64) ([]sdk.Model, error) {
	query := `
    SELECT worker_model.*, "group".name as groupname
    FROM worker_model
    JOIN "group" ON worker_model.group_id = "group".id
    WHERE worker_model.group_id = ANY(string_to_array($1, ',')::int[])
    AND worker_model.is_deprecated = false
    AND worker_model.disabled = false
  `
	return loadWorkerModels(db, false, query, gorpmapping.IDsToQueryString(groupIDs))
}

// LoadWorkerModelsByUser returns worker models list according to user's groups
func LoadWorkerModelsByUser(db gorp.SqlExecutor, store cache.Store, user *sdk.User, opts *StateLoadOption) ([]sdk.Model, error) {
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
		query = fmt.Sprintf(`select %s from worker_model JOIN "group" on worker_model.group_id = "group".id`, modelColumns)
		if len(additionalFilters) > 0 {
			query += fmt.Sprintf(" WHERE %s", strings.Join(additionalFilters, " AND "))
		}
	} else {
		query = fmt.Sprintf(`select %s
					from worker_model
					JOIN "group" on worker_model.group_id = "group".id
					where group_id in (select group_id from group_user where user_id = $1)
					union
					select %s from worker_model
					JOIN "group" on worker_model.group_id = "group".id
					where group_id = $2`, modelColumns, modelColumns)
		if len(additionalFilters) > 0 {
			query += fmt.Sprintf(" AND %s", strings.Join(additionalFilters, " AND "))
		}

		args = []interface{}{user.ID, group.SharedInfraGroup.ID}
	}
	models, err := loadWorkerModels(db, false, query, args...)
	if err != nil {
		return nil, err
	}

	store.SetWithTTL(key, models, modelsCacheTTLInSeconds)
	return models, nil
}

// LoadWorkerModelsUsableOnGroupWithClearPassword returns worker models for a group.
func LoadWorkerModelsUsableOnGroupWithClearPassword(db gorp.SqlExecutor, store cache.Store, groupID int64) ([]sdk.Model, error) {
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
	models, err := loadWorkerModels(db, false, query, groupID)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	store.SetWithTTL(key, models, modelsCacheTTLInSeconds)

	return models, nil
}

// LoadWorkerModelsByUserAndBinary returns worker models list according to user's groups and binary capability
func LoadWorkerModelsByUserAndBinary(db gorp.SqlExecutor, user *sdk.User, binary string) ([]sdk.Model, error) {
	if user.Admin {
		query := fmt.Sprintf(`
			SELECT %s
				FROM worker_model
					JOIN "group" ON worker_model.group_id = "group".id
					JOIN worker_capability ON worker_model.id = worker_capability.worker_model_id
					WHERE worker_capability.type = 'binary' AND worker_capability.argument = $1
    `, modelColumns)
		return loadWorkerModels(db, false, query, binary)
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
	return loadWorkerModels(db, false, query, user.ID, group.SharedInfraGroup.ID, binary)
}

func scanWorkerModels(db gorp.SqlExecutor, rows []dbResultWMS, withPassword bool) ([]sdk.Model, error) {
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

// DeleteWorkerModel removes from database worker model informations and all its capabilities
func DeleteWorkerModel(db gorp.SqlExecutor, ID int64) error {
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

// LoadWorkerModelCapabilities retrieves capabilities of given worker model
func LoadWorkerModelCapabilities(db gorp.SqlExecutor, workerID int64) (sdk.RequirementList, error) {
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
				return updateAllToCheckRegistration(db)
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

// updateAllToCheckRegistration is like need_registration but without exclusive mode
func updateAllToCheckRegistration(db gorp.SqlExecutor) error {
	query := `UPDATE worker_model SET check_registration = $1`
	res, err := db.Exec(query, true)
	if err != nil {
		return sdk.WithStack(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	log.Debug("updateAllToCheckRegistration> %d worker model(s) check registration", rows)
	return nil
}

// UpdateSpawnErrorWorkerModel updates worker model error registration
func UpdateSpawnErrorWorkerModel(db gorp.SqlExecutor, modelID int64, spawnError sdk.SpawnErrorForm) error {
	// some times when the docker container fails to start, the docker logs is not empty but only contains utf8 null char
	if spawnError.Error == string([]byte{0x00}) {
		spawnError.Error = ""
	}

	query := `UPDATE worker_model SET nb_spawn_err=nb_spawn_err+1, last_spawn_err=$3, last_spawn_err_log=$4, date_last_spawn_err=$2 WHERE id = $1`
	res, err := db.Exec(query, modelID, time.Now(), spawnError.Error, spawnError.Logs)
	if err != nil {
		return sdk.WithStack(err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	if n == 0 {
		return sdk.WithStack(sdk.ErrNoWorkerModel)
	}
	return nil
}

// updateRegistration updates need_registration to false and last_registration time, reset err registration
func updateRegistration(db gorp.SqlExecutor, modelID int64) error {
	query := `UPDATE worker_model SET need_registration=false, check_registration=false, last_registration = $2, nb_spawn_err=0, last_spawn_err=NULL, last_spawn_err_log=NULL WHERE id = $1`
	res, err := db.Exec(query, modelID, time.Now())
	if err != nil {
		return sdk.WithStack(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	log.Debug("updateRegistration> %d worker model updated", rows)
	return nil
}

// updateOSAndArch updates os and arch for a worker model
func updateOSAndArch(db gorp.SqlExecutor, modelID int64, OS, arch string) error {
	query := `UPDATE worker_model SET registered_os=$1, registered_arch = $2 WHERE id = $3`
	res, err := db.Exec(query, OS, arch, modelID)
	if err != nil {
		return sdk.WithStack(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	log.Debug("updateOSAndArch> %d worker model updated", rows)
	return nil
}

func keyBookWorkerModel(id int64) string {
	return cache.Key("book", "workermodel", strconv.FormatInt(id, 10))
}

// BookForRegister books a worker model for register, used by hatcheries
func BookForRegister(store cache.Store, id int64, hatchery *sdk.Service) (*sdk.Service, error) {
	k := keyBookWorkerModel(id)
	h := sdk.Service{}
	if !store.Get(k, &h) {
		// worker model not already booked, book it for 6 min
		store.SetWithTTL(k, hatchery, bookRegisterTTLInSeconds)
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrWorkerModelAlreadyBooked, "worker model %d already booked by %s (%d)", id, h.Name, h.ID)
}

// UnbookForRegister release the book
func UnbookForRegister(store cache.Store, id int64) {
	k := keyBookWorkerModel(id)
	store.Delete(k)
}

func MergeModelEnvsWithDefaultEnvs(envs map[string]string) map[string]string {
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

func getAdditionalSQLFilters(opts *StateLoadOption) []string {
	var additionalFilters []string
	if opts != nil {
		switch {
		case *opts == StateError:
			additionalFilters = append(additionalFilters, "worker_model.nb_spawn_err > 0")
		case *opts == StateDisabled:
			additionalFilters = append(additionalFilters, "worker_model.disabled = true")
		case *opts == StateRegister:
			additionalFilters = append(additionalFilters, "worker_model.need_registration = true")
		case *opts == StateDeprecated:
			additionalFilters = append(additionalFilters, "worker_model.is_deprecated = true")
		case *opts == StateActive:
			additionalFilters = append(additionalFilters, "worker_model.is_deprecated = false")
		case *opts == StateOfficial:
			additionalFilters = append(additionalFilters, fmt.Sprintf("worker_model.group_id = %d", group.SharedInfraGroup.ID))
		}
	}
	return additionalFilters
}
