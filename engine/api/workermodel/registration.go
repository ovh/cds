package workermodel

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
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
	var nbModelReq, nbOSArchReq, nbHostnameReq, nbRegionReq int

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
		case sdk.RegionRequirement:
			nbRegionReq++
		}
	}

	if nbOSArchReq > 1 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid os-architecture requirement usage")
	}
	if nbModelReq > 1 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid model requirement usage")
	}
	if nbHostnameReq > 1 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid hostname requirement usage")
	}
	if nbRegionReq > 1 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid region requirement usage")
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
	spawnError.Error = sdk.RemoveNotPrintableChar(spawnError.Error)
	spawnError.Logs = []byte(sdk.RemoveNotPrintableChar(string(spawnError.Logs)))

	query := `UPDATE worker_model SET nb_spawn_err=nb_spawn_err+1, last_spawn_err=$3, last_spawn_err_log=$4, date_last_spawn_err=$2 WHERE id = $1`
	res, err := db.Exec(query, modelID, time.Now(), spawnError.Error, string(spawnError.Logs))
	if err != nil {
		return sdk.WithStack(err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	if n == 0 {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	return nil
}

// UpdateRegistration updates need_registration to false and last_registration time, reset err registration.
func UpdateRegistration(ctx context.Context, db gorp.SqlExecutor, store cache.Store, modelID int64) error {
	query := `UPDATE worker_model SET need_registration=false, check_registration=false, last_registration=$2, nb_spawn_err=0, last_spawn_err=NULL, last_spawn_err_log=NULL WHERE id = $1`
	res, err := db.Exec(query, modelID, time.Now())
	if err != nil {
		return sdk.WithStack(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	log.Debug("UpdateRegistration> %d worker model updated", rows)
	UnbookForRegister(ctx, store, modelID)
	return nil
}

// UpdateOSAndArch updates os and arch for a worker model.
func UpdateOSAndArch(db gorp.SqlExecutor, modelID int64, OS, arch string) error {
	query := `UPDATE worker_model SET registered_os=$1, registered_arch=$2 WHERE id = $3`
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
func BookForRegister(store cache.Store, id int64, serviceID int64) error {
	k := KeyBookWorkerModel(id)
	var bookedByServiceID int64
	if ok, _ := store.Get(k, &bookedByServiceID); !ok {
		// worker model not already booked, book it for 6 min
		if err := store.SetWithTTL(k, serviceID, bookRegisterTTLInSeconds); err != nil {
			return err
		}
		return nil
	}
	return sdk.WrapError(sdk.ErrWorkerModelAlreadyBooked, "worker model %d already booked by service %d", id, bookedByServiceID)
}

// UnbookForRegister release the book
func UnbookForRegister(ctx context.Context, store cache.Store, id int64) {
	k := KeyBookWorkerModel(id)
	if err := store.Delete(k); err != nil {
		log.Error(ctx, "unable to delete cache key %v: %v", k, err)
	}
}

// UpdateCapabilities .
func UpdateCapabilities(ctx context.Context, db gorp.SqlExecutor, modelID int64, registrationForm sdk.WorkerRegistrationForm) error {
	existingCapas, err := LoadCapabilitiesByModelID(ctx, db, modelID)
	if err != nil {
		return sdk.WithStack(err)
	}

	var newCapas []string
	for _, b := range registrationForm.BinaryCapabilities {
		var found bool
		for _, c := range existingCapas {
			if b == c.Value {
				found = true
				break
			}
		}
		if !found {
			newCapas = append(newCapas, b)
		}
	}
	if len(newCapas) > 0 {
		log.Debug("Updating model %d binary capabilities with %d capabilities", modelID, len(newCapas))
		for _, b := range newCapas {
			if err := InsertCapabilityForModelID(db, modelID, &sdk.Requirement{
				Type:  sdk.BinaryRequirement,
				Name:  b,
				Value: b,
			}); err != nil {
				return err
			}
		}
	}

	var capaToDelete []string
	for _, existingCapa := range existingCapas {
		var found bool
		for _, currentCapa := range registrationForm.BinaryCapabilities {
			if existingCapa.Value == currentCapa {
				found = true
				break
			}
		}
		if !found {
			capaToDelete = append(capaToDelete, existingCapa.Value)
		}
	}

	if len(capaToDelete) > 0 {
		log.Debug("Updating model %d binary capabilities with %d capabilities to delete", modelID, len(capaToDelete))
		query := `DELETE FROM worker_capability WHERE worker_model_id=$1 AND name=ANY(string_to_array($2, ',')::text[]) AND type=$3`
		if _, err := db.Exec(query, modelID, strings.Join(capaToDelete, ","), string(sdk.BinaryRequirement)); err != nil {
			//Ignore errors because we let the database to check constraints...
			log.Warning(ctx, "registerWorker> Cannot delete from worker_capability: %v", err)
			return sdk.WithStack(err)
		}
	}

	if registrationForm.OS != "" && registrationForm.Arch != "" {
		if err := UpdateOSAndArch(db, modelID, registrationForm.OS, registrationForm.Arch); err != nil {
			log.Warning(ctx, "registerWorker> Cannot update os and arch for worker model %d : %s", modelID, err)
			return sdk.WithStack(err)
		}
	}

	return nil
}
