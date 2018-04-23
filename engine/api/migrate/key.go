package migrate

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func KeyMigration(store cache.Store, DBFunc func(context.Context) *gorp.DbMap, u *sdk.User) {
	db := DBFunc(context.Background())

	// Project migration
	ids, errQ := loadProjectIDs(db)
	if errQ != nil {
		return
	}

	for _, id := range ids {
		migrateProject(db, id, store, u)
	}

	// Application migration
	appIds, errA := loadApplicationIDs(db)
	if errA != nil {
		return
	}
	for _, id := range appIds {
		migrateApplication(db, id, store, u)
	}

	// Environment migration
	envIds, errE := loadEnvironmentIDs(db)
	if errE != nil {
		return
	}
	for _, id := range envIds {
		migrateEnvironment(db, id)
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

	proj, errP := project.LoadByID(tx, store, projID, u, project.LoadOptions.WithLockNoWait, project.LoadOptions.WithVariablesWithClearPassword, project.LoadOptions.WithKeys)
	if errP != nil {
		log.Warning("migrateProject> Cannot get project %d: %s", projID, errP)
		return errP
	}

	// find private key
	keys := findKeyPair(proj.Variable)
	for _, k := range keys {
		keyName := fmt.Sprintf("proj-%s", k.private.Name)
		found := false
		for _, k := range proj.Keys {
			if k.Name == keyName {
				found = true
				break
			}
		}

		if !found {
			projectKey := sdk.ProjectKey{
				ProjectID: projID,
				Builtin:   false,
				Key: sdk.Key{
					Public:  k.public.Value,
					Private: k.private.Value,
					Name:    keyName,
					Type:    sdk.KeyTypeSSH,
				},
			}
			if errK := project.InsertKey(tx, &projectKey); errK != nil {
				log.Warning("migrateProject> Cannot insert project key %s: %s", k.private.Name, errK)
				return errK
			}
		}
	}

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

	app, errA := application.LoadAndLockByID(tx, store, appID, u, application.LoadOptions.WithVariablesWithClearPassword, application.LoadOptions.WithKeys)
	if errA != nil {
		log.Warning("migrateApplication> Cannot load application %d: %s", appID, errA)
		return errA
	}
	keys := findKeyPair(app.Variable)
	for _, k := range keys {
		keyName := fmt.Sprintf("app-%s", k.private.Name)

		found := false
		for _, k := range app.Keys {
			if k.Name == keyName {
				found = true
				break
			}
		}

		if !found {
			appKey := sdk.ApplicationKey{
				ApplicationID: appID,
				Key: sdk.Key{
					Public:  k.public.Value,
					Private: k.private.Value,
					Name:    keyName,
					Type:    sdk.KeyTypeSSH,
				},
			}
			if errK := application.InsertKey(tx, &appKey); errK != nil {
				log.Warning("migrateApplication> Cannot insert application key %s: %s", k.private.Name, errK)
				return errK
			}
		}
	}
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

func migrateEnvironment(db *gorp.DbMap, envID int64) error {
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
	envVars, errV := environment.GetAllVariableByID(tx, envID, environment.WithClearPassword())
	if errV != nil {
		log.Warning("migrateEnvironment> Cannot load clear password")
	}
	env.Variable = envVars
	keys := findKeyPair(env.Variable)
	for _, k := range keys {
		keyName := fmt.Sprintf("env-%s", k.private.Name)

		found := false
		for _, k := range env.Keys {
			if k.Name == keyName {
				found = true
				break
			}
		}

		if !found {
			envKey := sdk.EnvironmentKey{
				EnvironmentID: envID,
				Key: sdk.Key{
					Public:  k.public.Value,
					Private: k.private.Value,
					Name:    keyName,
					Type:    sdk.KeyTypeSSH,
				},
			}
			if errK := environment.InsertKey(tx, &envKey); errK != nil {
				log.Warning("migrateEnvironment> Cannot insert environment key %s: %s", k.private.Name, errK)
				return errK
			}
		}
	}
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
				if strings.HasPrefix(vp.Name, v.Name) && strings.HasSuffix(vp.Name, ".pub") {
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
