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
type MapperFunc func() *gorpmapper.Mapper

func AdminGetDatabaseMigration(db DBFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		a, err := dbmigrate.List(db().Db)
		if err != nil {
			return sdk.WrapError(err, "cannot load database migration list %d", err)
		}
		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func AdminDeleteDatabaseMigration(db DBFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) == 0 {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Migration id is mandatory. Check id from table gorp_migrations")
		}

		return dbmigrate.DeleteMigrate(db().Db, id)
	}
}

func AdminPostDatabaseMigrationUnlock(db DBFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) == 0 {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Migration id is mandatory. Check id from table gorp_migrations_lock")
		}

		return dbmigrate.UnlockMigrate(db().Db, id, gorp.PostgresDialect{})
	}
}

func AdminGetDatabaseEntityList(db DBFunc, mapper MapperFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var res []sdk.DatabaseEntity
		for k, v := range mapper().Mapping {
			if !v.SignedEntity && !v.EncryptedEntity {
				continue
			}
			e := sdk.DatabaseEntity{
				Name:      k,
				Signed:    v.SignedEntity,
				Encrypted: v.EncryptedEntity,
			}
			if v.SignedEntity {
				data, err := mapper().ListCanonicalFormsByEntity(db(), k)
				if err != nil {
					return err
				}
				e.CanonicalForms = data
			}
			res = append(res, e)
		}
		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func AdminGetDatabaseEntity(db DBFunc, mapper MapperFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]

		signer := r.FormValue("signer")
		if signer != "" {
			res, err := mapper().ListTuplesByCanonicalForm(db(), entity, signer)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, res, http.StatusOK)
		}

		res, err := mapper().ListTuplesByEntity(db(), entity)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func AdminPostDatabaseEntityInfo(db DBFunc, mapper MapperFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]

		var pks []string
		if err := service.UnmarshalBody(r, &pks); err != nil {
			return err
		}

		var res []sdk.DatabaseEntityInfo
		for _, pk := range pks {
			i, err := mapper().InfoTupleByPrimaryKey(ctx, db(), entity, pk)
			if err != nil {
				return err
			}
			if i != nil {
				res = append(res, *i)
			}
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func AdminPostDatabaseEntityRoll(db DBFunc, mapper MapperFunc) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]

		ignoreMissing := r.FormValue("ignoreMissing") == sdk.TrueString

		var pks []string
		if err := service.UnmarshalBody(r, &pks); err != nil {
			return err
		}
		if len(pks) == 0 {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "no primary key was given")
		}

		var res []sdk.DatabaseEntityInfo
		for _, pk := range pks {
			tx, err := db().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}

			i, err := mapper().RollTupleByPrimaryKey(ctx, tx, entity, pk)
			if err != nil {
				tx.Rollback() // nolint
				return err
			}
			if i == nil {
				tx.Rollback() //nolint
				if ignoreMissing {
					continue
				}
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "no tuple found for primary key %s", pk)
			}

			res = append(res, *i)
			if err := tx.Commit(); err != nil {
				tx.Rollback() // nolint
				return sdk.WithStack(err)
			}
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}
