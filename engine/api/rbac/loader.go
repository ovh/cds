package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

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
		return err
	}
	if err := loadRbacProject(ctx, db, rbac); err != nil {
		return err
	}
	return nil
}

func loadRbacProject(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_project WHERE rbac_id = $1"
	var rbacProjects []rbacProject
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacProjects); err != nil {
		return err
	}
	rbac.Projects = make([]sdk.RBACProject, 0, len(rbacProjects))
	for i := range rbacProjects {
		rbacProject := &rbacProjects[i]
		isValid, err := gorpmapping.CheckSignature(rbacProject, rbacProject.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_project %d", rbacProject.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac_project.get> rbac_project %d data corrupted", rbacProject.ID)
			continue
		}
		if err := loadRBACProjectKeys(ctx, db, rbacProject); err != nil {
			return err
		}
		if !rbacProject.All {
			if err := loadRBACProjectUsers(ctx, db, rbacProject); err != nil {
				return err
			}
			if err := loadRBACProjectGroups(ctx, db, rbacProject); err != nil {
				return err
			}
		}
		rbac.Projects = append(rbac.Projects, rbacProject.RBACProject)
	}
	return nil
}

func loadRbacGlobal(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_global WHERE rbac_id = $1"
	var rbacGbl []rbacGlobal
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacGbl); err != nil {
		return err
	}
	rbac.Globals = make([]sdk.RBACGlobal, 0, len(rbacGbl))
	for i := range rbacGbl {
		rg := &rbacGbl[i]
		isValid, err := gorpmapping.CheckSignature(rg, rg.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_global %d", rg.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.loadRbacGlobal> rbac_global %d data corrupted", rg.ID)
			continue
		}
		if err := loadRBACGlobalUsers(ctx, db, rg); err != nil {
			return err
		}
		if err := loadRBACGlobalGroups(ctx, db, rg); err != nil {
			return err
		}
		rbac.Globals = append(rbac.Globals, rg.RBACGlobal)
	}
	return nil
}
