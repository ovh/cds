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
	Default               LoadOptionFunc
	LoadRBACGlobal        LoadOptionFunc
	LoadRBACProject       LoadOptionFunc
	LoadRBACHatchery      LoadOptionFunc
	LoadRBACRegion        LoadOptionFunc
	LoadRBACWorkflow      LoadOptionFunc
	LoadRBACVariableSet   LoadOptionFunc
	LoadRbacRegionProject LoadOptionFunc
	All                   LoadOptionFunc
}{
	Default:               loadDefault,
	LoadRBACGlobal:        loadRBACGlobal,
	LoadRBACProject:       loadRBACProject,
	LoadRBACHatchery:      loadRBACHatchery,
	LoadRBACRegion:        loadRBACRegion,
	LoadRBACWorkflow:      loadRBACWorkflow,
	LoadRBACVariableSet:   loadRBACVariableSet,
	LoadRbacRegionProject: loadRBACRegionProject,
	All:                   loadAll,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	if err := loadRBACGlobal(ctx, db, rbac); err != nil {
		return err
	}
	if err := loadRBACProject(ctx, db, rbac); err != nil {
		return err
	}
	return nil
}

func loadAll(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	if err := loadRBACGlobal(ctx, db, rbac); err != nil {
		return err
	}
	if err := loadRBACProject(ctx, db, rbac); err != nil {
		return err
	}
	if err := loadRBACRegion(ctx, db, rbac); err != nil {
		return err
	}
	if err := loadRBACHatchery(ctx, db, rbac); err != nil {
		return err
	}
	if err := loadRBACWorkflow(ctx, db, rbac); err != nil {
		return err
	}
	if err := loadRBACVariableSet(ctx, db, rbac); err != nil {
		return err
	}
	if err := loadRBACRegionProject(ctx, db, rbac); err != nil {
		return err
	}
	return nil
}

func loadRBACVariableSet(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_variableset WHERE rbac_id = $1"
	var rbacVariableSets []rbacVariableSet
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacVariableSets); err != nil {
		return err
	}
	rbac.VariableSets = make([]sdk.RBACVariableSet, 0, len(rbacVariableSets))
	for i := range rbacVariableSets {
		rbacVS := &rbacVariableSets[i]
		isValid, err := gorpmapping.CheckSignature(rbacVS, rbacVS.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_variableset %d", rbacVS.ID)
		}
		if !isValid {
			log.Error(ctx, "loadRBACVariableSet> rbac_variableset %d data corrupted", rbacVS.ID)
			continue
		}
		if !rbacVS.AllUsers {
			if err := loadRBACVariableSetUsers(ctx, db, rbacVS); err != nil {
				return err
			}
			if err := loadRBACVariableSetGroups(ctx, db, rbacVS); err != nil {
				return err
			}
		}
		rbac.VariableSets = append(rbac.VariableSets, rbacVS.RBACVariableSet)
	}
	return nil
}

func loadRBACWorkflow(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_workflow WHERE rbac_id = $1"
	var rbacWorkflows []rbacWorkflow
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacWorkflows); err != nil {
		return err
	}
	rbac.Workflows = make([]sdk.RBACWorkflow, 0, len(rbacWorkflows))
	for i := range rbacWorkflows {
		rbacWf := &rbacWorkflows[i]
		isValid, err := gorpmapping.CheckSignature(rbacWf, rbacWf.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_workflow %d", rbacWf.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac_workflow.get> rbac_workflow %d data corrupted", rbacWf.ID)
			continue
		}
		if !rbacWf.AllUsers {
			if err := loadRBACWorkflowUsers(ctx, db, rbacWf); err != nil {
				return err
			}
			if err := loadRBACWorkflowGroups(ctx, db, rbacWf); err != nil {
				return err
			}
		}
		rbac.Workflows = append(rbac.Workflows, rbacWf.RBACWorkflow)
	}
	return nil
}

func loadRBACProject(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
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
		if !rbacProject.AllUsers {
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

func loadRBACGlobal(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_global WHERE rbac_id = $1"
	var rbacGbl []rbacGlobal
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacGbl); err != nil {
		return err
	}
	rbac.Global = make([]sdk.RBACGlobal, 0, len(rbacGbl))
	for i := range rbacGbl {
		rg := &rbacGbl[i]
		isValid, err := gorpmapping.CheckSignature(rg, rg.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_global %d", rg.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.loadRBACGlobal> rbac_global %d data corrupted", rg.ID)
			continue
		}
		if err := loadRBACGlobalUsers(ctx, db, rg); err != nil {
			return err
		}
		if err := loadRBACGlobalGroups(ctx, db, rg); err != nil {
			return err
		}
		rbac.Global = append(rbac.Global, rg.RBACGlobal)
	}
	return nil
}

func loadRBACRegionProject(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_region_project WHERE rbac_id = $1"
	var rbacRegionProjects []rbacRegionProject
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacRegionProjects); err != nil {
		return err
	}
	rbac.RegionProjects = make([]sdk.RBACRegionProject, 0, len(rbacRegionProjects))
	for i := range rbacRegionProjects {
		rbacRegionProject := &rbacRegionProjects[i]
		isValid, err := gorpmapping.CheckSignature(rbacRegionProject, rbacRegionProject.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_region_project %d", rbacRegionProject.ID)
		}
		if !isValid {
			log.Error(ctx, "loadRBACRegionProject> rbac_region_project %d data corrupted", rbacRegionProject.ID)
			continue
		}
		if !rbacRegionProject.AllProjects {
			if err := loadRBACRegionProjectKeys(ctx, db, rbacRegionProject); err != nil {
				return err
			}
		}
		rbac.RegionProjects = append(rbac.RegionProjects, rbacRegionProject.RBACRegionProject)
	}
	return nil
}

func loadRBACRegion(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_region WHERE rbac_id = $1"
	var rbacRegions []rbacRegion
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacRegions); err != nil {
		return err
	}
	rbac.Regions = make([]sdk.RBACRegion, 0, len(rbacRegions))
	for i := range rbacRegions {
		rbacReg := &rbacRegions[i]
		isValid, err := gorpmapping.CheckSignature(rbacReg, rbacReg.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_region %d", rbacReg.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac_region.get> rbac_region %d data corrupted", rbacReg.ID)
			continue
		}
		if err := LoadRBACRegionOrganizations(ctx, db, &rbacReg.RBACRegion); err != nil {
			return err
		}
		if err := loadRBACRegionUsers(ctx, db, &rbacReg.RBACRegion); err != nil {
			return err
		}
		if err := loadRBACRegionGroups(ctx, db, &rbacReg.RBACRegion); err != nil {
			return err
		}

		rbac.Regions = append(rbac.Regions, rbacReg.RBACRegion)
	}
	return nil
}

func loadRBACHatchery(ctx context.Context, db gorp.SqlExecutor, rbac *rbac) error {
	query := "SELECT * FROM rbac_hatchery WHERE rbac_id = $1"
	var rbacHatcheries []rbacHatchery
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbac.ID), &rbacHatcheries); err != nil {
		return err
	}
	rbac.Hatcheries = make([]sdk.RBACHatchery, 0, len(rbacHatcheries))
	for i := range rbacHatcheries {
		rbacHatch := &rbacHatcheries[i]
		isValid, err := gorpmapping.CheckSignature(rbacHatch, rbacHatch.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_hatchery %d", rbacHatch.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac_region.get> rbac_hatchery %d data corrupted", rbacHatch.ID)
			continue
		}
		rbac.Hatcheries = append(rbac.Hatcheries, rbacHatch.RBACHatchery)
	}
	return nil
}
