package rbac

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func LoadRbacByName(ctx context.Context, db gorp.SqlExecutor, name string, opts ...LoadOptionFunc) (sdk.RBAC, error) {
	query := `SELECT * FROM rbac WHERE name = $1`
	return get(ctx, db, gorpmapping.NewQuery(query).Args(name), opts...)
}

// Insert a RBAC permission in database
func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rb *sdk.RBAC) error {
	if err := sdk.IsValidRbac(rb); err != nil {
		return err
	}
	if rb.UUID == "" {
		rb.UUID = sdk.UUID()
	}
	if rb.Created.IsZero() {
		rb.Created = time.Now()
	}
	rb.LastModified = time.Now()
	dbRb := rbac{RBAC: *rb}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbRb); err != nil {
		return err
	}

	for i := range rb.Globals {
		dbRbGlobal := rbacGlobal{
			RbacUUID:   dbRb.UUID,
			RBACGlobal: rb.Globals[i],
		}
		if err := insertRbacGlobal(ctx, db, &dbRbGlobal); err != nil {
			return err
		}
	}
	for i := range rb.Projects {
		dbRbProject := rbacProject{
			RbacUUID:    dbRb.UUID,
			RBACProject: rb.Projects[i],
		}
		if err := insertRbacProject(ctx, db, &dbRbProject); err != nil {
			return err
		}
	}
	*rb = dbRb.RBAC
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rb *sdk.RBAC) error {
	if err := Delete(ctx, db, *rb); err != nil {
		return err
	}
	return Insert(ctx, db, rb)
}

func Delete(_ context.Context, db gorpmapper.SqlExecutorWithTx, rb sdk.RBAC) error {
	dbRb := rbac{RBAC: rb}
	if err := gorpmapping.Delete(db, &dbRb); err != nil {
		return err
	}
	return nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (sdk.RBAC, error) {
	var r sdk.RBAC
	var rbacDB rbac
	found, err := gorpmapping.Get(ctx, db, q, &rbacDB)
	if err != nil {
		return r, err
	}
	if !found {
		return r, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(rbacDB, rbacDB.Signature)
	if err != nil {
		return r, sdk.WrapError(err, "error when checking signature for rbac %s", rbacDB.UUID)
	}
	if !isValid {
		log.Error(ctx, "rbac.get> rbac %s (%s) data corrupted", rbacDB.Name, rbacDB.UUID)
	}
	for _, f := range opts {
		if err := f(ctx, db, &rbacDB); err != nil {
			return r, err
		}
	}
	r = rbacDB.RBAC
	return r, nil
}
