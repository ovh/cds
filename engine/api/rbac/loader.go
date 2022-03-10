package rbac

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc is used as options to loadProject functions
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, *rbac) error

// LoadOptions provides all options on rbac loads functions
var LoadOptions = struct {
	Default         LoadOptionFunc
	LoadRbacGlobal  LoadOptionFunc
	LoadRbacProject LoadOptionFunc
}{
	Default:         loadDefault,
	LoadRbacGlobal:  loadRbacGlobal,
	LoadRbacProject: loadRbacProject,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	if err := loadRbacGlobal(ctx, db, rbac); err != nil {
		return sdk.WithStack(err)
	}
	if err := loadRbacProject(ctx, db, rbac); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func loadRbacProject(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_project WHERE rbac_uuid = $1"
	var rbacPrj []rbacProject
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.UUID), &rbacPrj); err != nil {
		return err
	}
	rbac.Projects = make([]sdk.RbacProject, 0, len(rbacPrj))
	for i := range rbacPrj {
		rp := &rbacPrj[i]
		if err := loadRbacProjectTargeted(ctx, db, rp); err != nil {
			return err
		}
		if !rp.All {
			if err := loadRbacRbacProjectUsersTargeted(ctx, db, rp); err != nil {
				return err
			}
			if err := loadRbacRbacProjectGroupsTargeted(ctx, db, rp); err != nil {
				return err
			}
		}
		rbac.Projects = append(rbac.Projects, rp.RbacProject)
	}
	return nil
}

func loadRbacGlobal(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_global WHERE rbac_uuid = $1"
	var rbacGbl []rbacGlobal
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.UUID), &rbacGbl); err != nil {
		return err
	}
	rbac.Globals = make([]sdk.RbacGlobal, 0, len(rbacGbl))
	for i := range rbacGbl {
		rg := &rbacGbl[i]
		if err := loadRbacRbacGlobalUsersTargeted(ctx, db, rg); err != nil {
			return err
		}
		if err := loadRbacRbacGlobalGroupsTargeted(ctx, db, rg); err != nil {
			return err
		}
		rbac.Globals = append(rbac.Globals, rg.RbacGlobal)
	}
	return nil
}
