package migrate

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RefactorWorkerModelCrypto .
func RefactorWorkerModelCrypto(ctx context.Context, db *gorp.DbMap) error {
	query := "SELECT id FROM worker_model WHERE sig IS NULL"
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := refactorWorkerModelCrypto(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorWorkerModelCrypto> unable to migrate worker_model %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorWorkerModelCrypto(ctx context.Context, db *gorp.DbMap, id int64) error {
	log.Info(ctx, "migrate.refactorWorkerModelCrypto> worker_model %d migration begin", id)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	query := `
    SELECT
      id, type, name, image, created_by, group_id, last_registration, need_registration,
      disabled, user_last_modified, last_spawn_err, nb_spawn_err, date_last_spawn_err, description,
      restricted, is_deprecated, registered_os, registered_arch, check_registration, last_spawn_err_log,
      model_docker, model_virtual_machine
    FROM worker_model
    WHERE id = $1
    AND sig IS NULL
    FOR UPDATE SKIP LOCKED
  `

	var m sdk.Model

	if err := tx.QueryRow(query, id).Scan(
		&m.ID, &m.Type, &m.Name, &m.Image, &m.Author, &m.GroupID, &m.LastRegistration, &m.NeedRegistration,
		&m.Disabled, &m.UserLastModified, &m.LastSpawnErr, &m.NbSpawnErr, &m.DateLastSpawnErr, &m.Description,
		&m.Restricted, &m.IsDeprecated, &m.RegisteredOS, &m.RegisteredArch, &m.CheckRegistration, &m.LastSpawnErrLogs,
		&m.ModelDocker, &m.ModelVirtualMachine,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "unable to select and lock application %d", id)
	}

	// Clean useless model data part if exists
	switch m.Type {
	case sdk.Docker:
		m.ModelVirtualMachine = sdk.ModelVirtualMachine{}
	default:
		m.ModelDocker = sdk.ModelDocker{}
	}

	// For docker model with private registry password, we want to move the password to secrets.
	// To do it we will give the clear value in PasswordInput field that will be managed by UpdateDB func.
	if m.Type == sdk.Docker && m.ModelDocker.Private && m.ModelDocker.Password != "" {
		clearPassword, err := secret.DecryptValue(m.ModelDocker.Password)
		if err != nil {
			return sdk.WrapError(err, "cannot decrypt registry password for model with id %d", m.ID)
		}
		m.ModelDocker.PasswordInput = clearPassword
	}

	if err := workermodel.UpdateDB(ctx, tx, &m); err != nil {
		return sdk.WrapError(err, "unable to update worker_model %d", id)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.refactorWorkerModelCrypto> worker_model %d migration end", id)
	return nil
}
