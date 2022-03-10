package rbac

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func LoadRbacByName(ctx context.Context, db gorp.SqlExecutor, name string, opts ...LoadOptionFunc) (sdk.Rbac, error) {
	query := `SELECT * FROM rbac WHERE name = $1`
	var r sdk.Rbac
	var rbacDB rbac
	if _, err := gorpmapping.Get(ctx, db, gorpmapping.NewQuery(query).Args(name), &rbacDB); err != nil {
		return r, err
	}
	for _, f := range opts {
		if err := f(ctx, db, &rbacDB); err != nil {
			return r, err
		}
	}
	r = rbacDB.Rbac
	return r, nil
}

func LoadRbacProjectIDsByUserID(_ context.Context, db gorp.SqlExecutor, role string, userID string) ([]sdk.IDName, error) {
	query := `
		WITH userRbac as (
			SELECT distinct(rpi.project_id) as id
			FROM rbac_project_ids rpi
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.role = $1
			JOIN rbac_project_users rpu ON rpu.rbac_project_id = rp.id
			WHERE rpu.user_id = $2
		),
		groupRbac as (
			SELECT distinct(rpi.project_id) as id
			FROM rbac_project_ids rpi
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.role = $1
			JOIN rbac_project_groups rpg ON rpg.rbac_project_id = rp.id
			JOIN "group" g ON g.id = rpg.group_id
			JOIN group_authentified_user gau ON gau.group_id = g.id
			WHERE gau.authentified_user_id = $2
		),
		userAllRbac as (
			SELECT distinct(p.id) as id
			FROM project p
			JOIN rbac_project_ids rpi ON rpi.project_id = p.id
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.role = $1 AND rp.all = true 
			JOIN rbac_project_users rpu ON rpu.rbac_project_id = rp.id AND rpu.user_id = $2
		),
		groupAllRbac as (
			SELECT distinct(p.id) as id
			FROM project p
			JOIN rbac_project_ids rpi ON rpi.project_id = p.id
			JOIN rbac_project rp ON rp.id = rpi.rbac_project_id AND rp.role = $1 AND rp.all = true 
			JOIN rbac_project_groups rpg ON rpg.rbac_project_id = rp.id
			JOIN "group" g ON g.id = rpg.group_id
			JOIN group_authentified_user gau ON gau.group_id = g.id AND gau.authentified_user_id = $2
		),
		concat as (
			SELECT distinct(id) as id FROM (
				SELECT id FROM userRbac UNION SELECT id FROM groupRbac UNION SELECT id FROM userAllRbac UNION SELECT id FROM groupAllRbac
			) tmp
		)
		SELECT p.id, p.name FROM concat c
		JOIN project p ON p.id = c.id`
	var projectIDNames []sdk.IDName
	if _, err := db.Select(&projectIDNames, query, role, userID); err != nil {
		return nil, err
	}
	return projectIDNames, nil
}

// Insert a RBAC permission in database
func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rb *sdk.Rbac) error {
	if err := sdk.IsValidRbac(rb); err != nil {
		return err
	}
	rb.UUID = sdk.UUID()
	rb.Created = time.Now()
	rb.LastModified = time.Now()
	dbRb := rbac{Rbac: *rb}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbRb); err != nil {
		return err
	}

	for i := range rb.Globals {
		rg := &rb.Globals[i]
		rg.RbacUUID = dbRb.UUID
		if err := insertRbacGlobal(ctx, db, rg); err != nil {
			return err
		}
		dbRb.Rbac.Globals[i] = *rg
	}
	for i := range rb.Projects {
		rp := &rb.Projects[i]
		rp.RbacUUID = dbRb.UUID
		if err := insertRbacProject(ctx, db, rp); err != nil {
			return err
		}
	}
	*rb = dbRb.Rbac
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rb *sdk.Rbac) error {
	if err := Delete(ctx, db, *rb); err != nil {
		return err
	}
	return Insert(ctx, db, rb)
}

func Delete(_ context.Context, db gorpmapper.SqlExecutorWithTx, rb sdk.Rbac) error {
	dbRb := rbac{Rbac: rb}
	if err := gorpmapping.Delete(db, &dbRb); err != nil {
		return err
	}
	return nil
}
