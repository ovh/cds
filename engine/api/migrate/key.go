package migrate

import (
	"strings"

	"github.com/go-gorp/gorp"

	"fmt"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func KeyMigration(store cache.Store, DBFunc func() *gorp.DbMap, u *sdk.User) {
	db := DBFunc()

	// Project migration
	ids, errQ := loadProjectIDs(db)
	if errQ != nil {
		return
	}
	for _, id := range ids {
		if err := migrateProject(db, id, store, u); err != nil {
			continue
		}
	}

	// Application migration
	appIds, errA := loadApplicationIDs(db)
	if errA != nil {
		return
	}
	for _, id := range appIds {
		if err := migrateApplication(db, id, store, u); err != nil {
			continue
		}
	}

	// Environment migration
	envIds, errE := loadEnvironmentIDs(db)
	if errE != nil {
		return
	}
	for _, id := range envIds {
		if err := migrateEnvironment(db, id, u); err != nil {
			continue
		}
	}
}

func loadProjectIDs(db gorp.SqlExecutor) ([]int64, error) {
	query := `SELECT project_id from project_variable where var_type = 'key'`
	rows, errQ := db.Query(query)
	if errQ != nil {
		log.Warning("loadProjectIDs> Cannot load project ids: %s", errQ)
		return nil, errQ
	}
	var ids []int64
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Warning("loadProjectIDs> Cannot key next id: %s", err)
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func migrateProject(db *gorp.DbMap, projID int64, store cache.Store, u *sdk.User) error {
	tx, errT := db.Begin()
	if errT != nil {
		log.Warning("migrateProject> Cannot start transaction: %s", errT)
		return errT
	}
	defer tx.Rollback()

	proj, errP := project.LoadByID(tx, store, projID, u, project.LoadOptions.WithLockNoWait, project.LoadOptions.WithVariablesWithClearPassword)
	if errP != nil {
		log.Warning("migrateProject> Cannot get project %d: %s", projID, errP)
		return errP
	}

	// find private key
	keys := findKeyPair(proj.Variable)
	for _, k := range keys {
		projectKey := sdk.ProjectKey{
			ProjectID: projID,
			Builtin:   false,
			Key: sdk.Key{
				Public:  k.public.Value,
				Private: k.private.Value,
				Name:    fmt.Sprintf("proj-%s", k.private.Name),
				Type:    sdk.KeySSHParameter,
			},
		}
		if errK := project.InsertKey(tx, &projectKey); errK != nil {
			log.Warning("migrateProject> Cannot insert project key %s: %s", k.private.Name, errK)
			return errK
		}

		if errD := project.DeleteVariable(tx, proj, &k.private, u); errD != nil {
			log.Warning("migrateProject> Unable to delete private key variable %s for project %d: %s", k.private.Name, projID, errD)
			return errD
		}
		if errPub := project.DeleteVariable(tx, proj, &k.public, u); errPub != nil {
			log.Warning("migrateProject> Unable to delete public key variable %s for project %d: %s", k.public.Name, projID, errPub)
			return errPub
		}
	}

	// TODO Update git clone action on pipeline

	return tx.Commit()
}

func loadApplicationIDs(db gorp.SqlExecutor) ([]int64, error) {
	query := `SELECT application_id from application_variable where var_type = 'key'`
	rows, errQ := db.Query(query)
	if errQ != nil {
		log.Warning("loadApplicationIDs> Cannot load application ids: %s", errQ)
		return nil, errQ
	}
	var ids []int64
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Warning("loadApplicationIDs> Cannot key next id: %s", err)
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func migrateApplication(db *gorp.DbMap, appID int64, store cache.Store, u *sdk.User) error {
	tx, errT := db.Begin()
	if errT != nil {
		log.Warning("migrateApplication> Cannot start transaction: %s", errT)
		return errT
	}

	// TODO add with lock application
	app, errA := application.LoadByID(tx, store, appID, u, application.LoadOptions.WithVariables)
	if errA != nil {
		log.Warning("migrateApplication> Cannot load application %d: %s", appID, errA)
	}
	keys := findKeyPair(app.Variable)
	for _, k := range keys {
		appKey := sdk.ApplicationKey{
			ApplicationID: appID,
			Key: sdk.Key{
				Public:  k.public.Value,
				Private: k.private.Value,
				Name:    fmt.Sprintf("app-%s", k.private.Name),
				Type:    sdk.KeySSHParameter,
			},
		}
		if errK := application.InsertKey(tx, &appKey); errK != nil {
			log.Warning("migrateApplication> Cannot insert application key %s: %s", k.private.Name, errK)
			return errK
		}

		if errD := application.DeleteVariable(tx, store, app, &k.private, u); errD != nil {
			log.Warning("migrateApplication> Unable to delete private key variable %s for application %d: %s", k.private.Name, appID, errD)
			return errD
		}
		if errPub := application.DeleteVariable(tx, store, app, &k.public, u); errPub != nil {
			log.Warning("migrateProject> Unable to delete public key variable %s for application %d: %s", k.public.Name, appID, errPub)
			return errPub
		}

	}

	// TODO Update pipelines

	return tx.Commit()
}

func loadEnvironmentIDs(db gorp.SqlExecutor) ([]int64, error) {
	query := `SELECT environment_id from environment_variable where type = 'key'`
	rows, errQ := db.Query(query)
	if errQ != nil {
		log.Warning("loadEnvironmentIDs> Cannot load environment ids: %s", errQ)
		return nil, errQ
	}
	var ids []int64
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Warning("loadEnvironmentIDs> Cannot key next id: %s", err)
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func migrateEnvironment(db *gorp.DbMap, envID int64, u *sdk.User) error {
	tx, errT := db.Begin()
	if errT != nil {
		log.Warning("migrateEnvironment> Cannot start transaction: %s", errT)
		return errT
	}

	if errL := environment.LockByID(tx, envID); errL != nil {
		log.Warning("migrateEnvironment> Cannot lock environment %d: %s", envID, errL)
		return errL
	}
	env, errR := environment.LoadEnvironmentByID(tx, envID)
	if errR != nil {
		log.Warning("migrateEnvironment> Cannot load environment %d: %s", envID, errR)
		return errR
	}
	keys := findKeyPair(env.Variable)
	for _, k := range keys {
		envKey := sdk.EnvironmentKey{
			EnvironmentID: envID,
			Key: sdk.Key{
				Public:  k.public.Value,
				Private: k.private.Value,
				Name:    fmt.Sprintf("env-%s", k.private.Name),
				Type:    sdk.KeySSHParameter,
			},
		}
		if errK := environment.InsertKey(tx, &envKey); errK != nil {
			log.Warning("migrateEnvironment> Cannot insert environment key %s: %s", k.private.Name, errK)
			return errK
		}

		if errD := environment.DeleteVariable(tx, envID, &k.private, u); errD != nil {
			log.Warning("migrateEnvironment> Unable to delete private key variable %s for environment %d: %s", k.private.Name, envID, errD)
			return errD
		}
		if errPub := environment.DeleteVariable(tx, envID, &k.public, u); errPub != nil {
			log.Warning("migrateEnvironment> Unable to delete public key variable %s for environment %d: %s", k.public.Name, envID, errPub)
			return errPub
		}
	}
	// TODO pipeline keys

	return tx.Commit()
}

type keyPair struct {
	private sdk.Variable
	public  sdk.Variable
}

func findKeyPair(vs []sdk.Variable) []keyPair {
	kps := []keyPair{}
	for _, v := range vs {
		if v.Type == sdk.KeyVariable {
			for _, vp := range vs {
				if strings.HasPrefix(vp.Name, v.Name) {
					kp := keyPair{
						private: v,
						public:  vp,
					}
					kps = append(kps, kp)
					break
				}
			}
		}
	}
	return kps
}
