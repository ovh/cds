package workermodel

import (
	"errors"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	bookRegisterTTLInSeconds = 360
)

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

// UpdateRegistration updates need_registration to false and last_registration time, reset err registration.
func UpdateRegistration(db gorp.SqlExecutor, modelID int64) error {
	query := `UPDATE worker_model SET need_registration=false, check_registration=false, last_registration = $2, nb_spawn_err=0, last_spawn_err=NULL, last_spawn_err_log=NULL WHERE id = $1`
	res, err := db.Exec(query, modelID, time.Now())
	if err != nil {
		return sdk.WithStack(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	log.Debug("UpdateRegistration> %d worker model updated", rows)
	return nil
}

// UpdateOSAndArch updates os and arch for a worker model.
func UpdateOSAndArch(db gorp.SqlExecutor, modelID int64, OS, arch string) error {
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

// KeyBookWorkerModel returns cache key for given model id.
func KeyBookWorkerModel(id int64) string {
	return cache.Key("book", "workermodel", strconv.FormatInt(id, 10))
}

// BookForRegister books a worker model for register, used by hatcheries
func BookForRegister(store cache.Store, id int64, hatchery *sdk.Service) (*sdk.Service, error) {
	k := KeyBookWorkerModel(id)
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
	k := KeyBookWorkerModel(id)
	store.Delete(k)
}
