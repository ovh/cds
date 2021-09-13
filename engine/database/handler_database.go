package database

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/database/dbmigrate"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

type DBFunc func() *gorp.DbMap

func AdminDeleteDatabaseMigration(db DBFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) == 0 {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Id is mandatory. Check id from table gorp_migrations")
		}

		return dbmigrate.DeleteMigrate(db().Db, id)
	}
}

func AdminDatabaseMigrationUnlocked(db DBFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) == 0 {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Id is mandatory. Check id from table gorp_migrations_lock")
		}

		return dbmigrate.UnlockMigrate(db().Db, id, gorp.PostgresDialect{})
	}
}

func AdminGetDatabaseMigration(db DBFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		a, err := dbmigrate.List(db().Db)
		if err != nil {
			return sdk.WrapError(err, "Cannot load database migration list %d", err)
		}
		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func AdminDatabaseSignatureResume(db DBFunc, mapper *gorpmapper.Mapper) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var entities = mapper.ListSignedEntities()
		var resume = make(sdk.CanonicalFormUsageResume, len(entities))

		for _, e := range entities {
			data, err := mapper.ListCanonicalFormsByEntity(db(), e)
			if err != nil {
				return err
			}
			resume[e] = data
		}

		return service.WriteJSON(w, resume, http.StatusOK)
	}
}

func AdminDatabaseSignatureTuplesBySigner(db DBFunc, mapper *gorpmapper.Mapper) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]
		signer := vars["signer"]

		pks, err := mapper.ListTupleByCanonicalForm(db(), entity, signer)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, pks, http.StatusOK)
	}
}

func AdminDatabaseSignatureRollEntityByPrimaryKey(db DBFunc, mapper *gorpmapper.Mapper) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]
		pk := vars["pk"]

		tx, err := db().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		if err := mapper.RollSignedTupleByPrimaryKey(ctx, tx, entity, pk); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func AdminDatabaseEncryptedEntities(db DBFunc, mapper *gorpmapper.Mapper) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, mapper.ListEncryptedEntities(), http.StatusOK)
	}
}

func AdminDatabaseEncryptedTuplesByEntity(db DBFunc, mapper *gorpmapper.Mapper) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]

		pks, err := mapper.ListTuplesByEntity(db(), entity)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, pks, http.StatusOK)
	}
}

func AdminDatabaseRollEncryptedEntityByPrimaryKey(db DBFunc, mapper *gorpmapper.Mapper) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]
		pk := vars["pk"]

		if err := mapper.RollEncryptedTupleByPrimaryKey(db(), entity, pk); err != nil {
			return err
		}

		return nil
	}
}
