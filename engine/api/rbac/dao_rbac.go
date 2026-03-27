package rbac

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func LoadAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.RBAC, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM rbac`)
	return getAll(ctx, db, query)
}

func LoadRBACByName(ctx context.Context, db gorp.SqlExecutor, name string, opts ...LoadOptionFunc) (*sdk.RBAC, error) {
	query := `SELECT * FROM rbac WHERE name = $1`
	return get(ctx, db, gorpmapping.NewQuery(query).Args(name), opts...)
}

func LoadRBACByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadOptionFunc) (*sdk.RBAC, error) {
	query := `SELECT * FROM rbac WHERE id = $1`
	return get(ctx, db, gorpmapping.NewQuery(query).Args(id), opts...)
}

func LoadRBACByIDs(ctx context.Context, db gorp.SqlExecutor, IDs sdk.StringSlice, opts ...LoadOptionFunc) ([]sdk.RBAC, error) {
	query := `SELECT * FROM rbac WHERE id = ANY ($1)`
	return getAll(ctx, db, gorpmapping.NewQuery(query).Args(pq.StringArray(IDs)), opts...)
}

// Insert a RBAC permission in database
func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rb *sdk.RBAC) error {
	if err := IsValidRBAC(ctx, db, rb); err != nil {
		return err
	}
	if rb.ID == "" {
		rb.ID = sdk.UUID()
	}
	if rb.Created.IsZero() {
		rb.Created = time.Now()
	}
	rb.LastModified = time.Now()
	dbRb := rbac{RBAC: *rb}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbRb); err != nil {
		return err
	}

	for i := range rb.Global {
		dbRbGlobal := rbacGlobal{
			RbacID:     dbRb.ID,
			RBACGlobal: rb.Global[i],
		}
		if err := insertRBACGlobal(ctx, db, &dbRbGlobal); err != nil {
			return err
		}
	}
	for i := range rb.Projects {
		dbRbProject := rbacProject{
			RbacID:      dbRb.ID,
			RBACProject: rb.Projects[i],
		}
		if err := insertRBACProject(ctx, db, &dbRbProject); err != nil {
			return err
		}
	}
	for i := range rb.Workflows {
		dbRbWorkflow := rbacWorkflow{
			RbacID:       dbRb.ID,
			RBACWorkflow: rb.Workflows[i],
		}
		if err := insertRBACWorkflow(ctx, db, &dbRbWorkflow); err != nil {
			return err
		}
	}
	for i := range rb.Regions {
		rbRegion := &rb.Regions[i]
		rbRegion.RbacID = dbRb.ID
		dbRbRegion := rbacRegion{RBACRegion: *rbRegion}
		if err := insertRBACRegion(ctx, db, &dbRbRegion); err != nil {
			return err
		}
	}
	for i := range rb.Hatcheries {
		dbRbHatchery := rbacHatchery{
			RbacID:       dbRb.ID,
			RBACHatchery: rb.Hatcheries[i],
		}
		if err := insertRBACHatchery(ctx, db, &dbRbHatchery); err != nil {
			return err
		}
	}
	for i := range rb.VariableSets {
		dbRbVariableSet := rbacVariableSet{
			RbacID:          dbRb.ID,
			RBACVariableSet: rb.VariableSets[i],
		}
		if err := insertRBACVariableSet(ctx, db, &dbRbVariableSet); err != nil {
			return err
		}
	}
	for i := range rb.RegionProjects {
		dbRbRegionProject := rbacRegionProject{
			RbacID:            dbRb.ID,
			RBACRegionProject: rb.RegionProjects[i],
		}
		if err := insertRBACRegionProject(ctx, db, &dbRbRegionProject); err != nil {
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

// LoadAllRBACByUserID returns all RBAC rules where the given user is referenced,
// either directly or through one of their groups, across all scopes
// (global, project, region, workflow, variableset).
func LoadAllRBACByUserID(ctx context.Context, db gorp.SqlExecutor, userID string, opts ...LoadOptionFunc) ([]sdk.RBAC, error) {
	rbacIDs := make(sdk.StringSlice, 0)

	// --- Global ---
	rbacGlobalUsers, err := loadRBACGlobalUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacGlobalGroups, err := loadRBACGlobalGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	globalIDs := make(sdk.Int64Slice, 0)
	for _, x := range rbacGlobalUsers {
		globalIDs = append(globalIDs, x.RbacGlobalID)
	}
	for _, x := range rbacGlobalGroups {
		globalIDs = append(globalIDs, x.RbacGlobalID)
	}
	globalIDs.Unique()
	if len(globalIDs) > 0 {
		globals, err := loadRBACGlobalsByIDs(ctx, db, globalIDs)
		if err != nil {
			return nil, err
		}
		for _, g := range globals {
			rbacIDs = append(rbacIDs, g.RbacID)
		}
	}

	// --- Project ---
	rbacProjectUsers, err := loadRBACProjectUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacProjectGroups, err := loadRBACProjectGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacProjectIDs := make(sdk.Int64Slice, 0)
	for _, x := range rbacProjectUsers {
		rbacProjectIDs = append(rbacProjectIDs, x.RbacProjectID)
	}
	for _, x := range rbacProjectGroups {
		rbacProjectIDs = append(rbacProjectIDs, x.RbacProjectID)
	}
	rbacProjectIDs.Unique()
	if len(rbacProjectIDs) > 0 {
		projects, err := loadRBACProjectsByIDs(ctx, db, rbacProjectIDs)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			rbacIDs = append(rbacIDs, p.RbacID)
		}
	}

	// --- Region ---
	rbacRegionUsers, err := loadRBACRegionUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacRegionGroups, err := loadRBACRegionGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacRegionIDs := make(sdk.Int64Slice, 0)
	for _, x := range rbacRegionUsers {
		rbacRegionIDs = append(rbacRegionIDs, x.RbacRegionID)
	}
	for _, x := range rbacRegionGroups {
		rbacRegionIDs = append(rbacRegionIDs, x.RbacRegionID)
	}
	rbacRegionIDs.Unique()
	if len(rbacRegionIDs) > 0 {
		regions, err := loadRBACRegionsByIDs(ctx, db, rbacRegionIDs)
		if err != nil {
			return nil, err
		}
		for _, r := range regions {
			rbacIDs = append(rbacIDs, r.RbacID)
		}
	}

	// --- Workflow ---
	rbacWorkflowUsers, err := loadRBACWorkflowUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacWorkflowGroups, err := loadRBACWorkflowGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacWorkflowIDs := make(sdk.Int64Slice, 0)
	for _, x := range rbacWorkflowUsers {
		rbacWorkflowIDs = append(rbacWorkflowIDs, x.RbacWorkflowID)
	}
	for _, x := range rbacWorkflowGroups {
		rbacWorkflowIDs = append(rbacWorkflowIDs, x.RbacWorkflowID)
	}
	rbacWorkflowIDs.Unique()
	if len(rbacWorkflowIDs) > 0 {
		workflows, err := loadRBACWorkflowsByIDs(ctx, db, rbacWorkflowIDs)
		if err != nil {
			return nil, err
		}
		for _, w := range workflows {
			rbacIDs = append(rbacIDs, w.RbacID)
		}
	}

	// --- VariableSet ---
	rbacVSUsers, err := loadRBACVariableSetUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacVSGroups, err := loadRBACVariableSetGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	rbacVSIDs := make(sdk.Int64Slice, 0)
	for _, x := range rbacVSUsers {
		rbacVSIDs = append(rbacVSIDs, x.RbacVariableSetID)
	}
	for _, x := range rbacVSGroups {
		rbacVSIDs = append(rbacVSIDs, x.RbacVariableSetID)
	}
	rbacVSIDs.Unique()
	if len(rbacVSIDs) > 0 {
		variableSets, err := loadRBACVariableSetsByIDs(ctx, db, rbacVSIDs)
		if err != nil {
			return nil, err
		}
		for _, vs := range variableSets {
			rbacIDs = append(rbacIDs, vs.RbacID)
		}
	}

	rbacIDs.Unique()
	if len(rbacIDs) == 0 {
		return nil, nil
	}
	return LoadRBACByIDs(ctx, db, rbacIDs, opts...)
}

// LoadAllRBACByGroupID returns all RBAC rules where the given group is referenced
// across all scopes (global, project, region, workflow, variableset).
func LoadAllRBACByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64, opts ...LoadOptionFunc) ([]sdk.RBAC, error) {
	groupIDs := []int64{groupID}
	rbacIDs := make(sdk.StringSlice, 0)

	// --- Global ---
	rbacGlobalGroups, err := loadRBACGlobalGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return nil, err
	}
	rbacGlobalIDs := make(sdk.Int64Slice, 0, len(rbacGlobalGroups))
	for _, x := range rbacGlobalGroups {
		rbacGlobalIDs = append(rbacGlobalIDs, x.RbacGlobalID)
	}
	rbacGlobalIDs.Unique()
	if len(rbacGlobalIDs) > 0 {
		globals, err := loadRBACGlobalsByIDs(ctx, db, rbacGlobalIDs)
		if err != nil {
			return nil, err
		}
		for _, g := range globals {
			rbacIDs = append(rbacIDs, g.RbacID)
		}
	}

	// --- Project ---
	rbacProjectGroups, err := loadRBACProjectGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return nil, err
	}
	rbacProjectIDs := make(sdk.Int64Slice, 0, len(rbacProjectGroups))
	for _, x := range rbacProjectGroups {
		rbacProjectIDs = append(rbacProjectIDs, x.RbacProjectID)
	}
	rbacProjectIDs.Unique()
	if len(rbacProjectIDs) > 0 {
		projects, err := loadRBACProjectsByIDs(ctx, db, rbacProjectIDs)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			rbacIDs = append(rbacIDs, p.RbacID)
		}
	}

	// --- Region ---
	rbacRegionGroups, err := loadRBACRegionGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return nil, err
	}
	rbacRegionIDs := make(sdk.Int64Slice, 0, len(rbacRegionGroups))
	for _, x := range rbacRegionGroups {
		rbacRegionIDs = append(rbacRegionIDs, x.RbacRegionID)
	}
	rbacRegionIDs.Unique()
	if len(rbacRegionIDs) > 0 {
		regions, err := loadRBACRegionsByIDs(ctx, db, rbacRegionIDs)
		if err != nil {
			return nil, err
		}
		for _, r := range regions {
			rbacIDs = append(rbacIDs, r.RbacID)
		}
	}

	// --- Workflow ---
	rbacWorkflowGroups, err := loadRBACWorkflowGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return nil, err
	}
	rbacWorflowIDs := make(sdk.Int64Slice, 0, len(rbacWorkflowGroups))
	for _, x := range rbacWorkflowGroups {
		rbacWorflowIDs = append(rbacWorflowIDs, x.RbacWorkflowID)
	}
	rbacWorflowIDs.Unique()
	if len(rbacWorflowIDs) > 0 {
		workflows, err := loadRBACWorkflowsByIDs(ctx, db, rbacWorflowIDs)
		if err != nil {
			return nil, err
		}
		for _, w := range workflows {
			rbacIDs = append(rbacIDs, w.RbacID)
		}
	}

	// --- VariableSet ---
	rbacVSGroups, err := loadRBACVariableSetGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return nil, err
	}
	rbacVSIDs := make(sdk.Int64Slice, 0, len(rbacVSGroups))
	for _, x := range rbacVSGroups {
		rbacVSIDs = append(rbacVSIDs, x.RbacVariableSetID)
	}
	rbacVSIDs.Unique()
	if len(rbacVSIDs) > 0 {
		variableSets, err := loadRBACVariableSetsByIDs(ctx, db, rbacVSIDs)
		if err != nil {
			return nil, err
		}
		for _, vs := range variableSets {
			rbacIDs = append(rbacIDs, vs.RbacID)
		}
	}

	rbacIDs.Unique()
	if len(rbacIDs) == 0 {
		return nil, nil
	}
	return LoadRBACByIDs(ctx, db, rbacIDs, opts...)
}

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.RBAC, error) {
	var rs []sdk.RBAC
	var rbacsDB []rbac
	if err := gorpmapping.GetAll(ctx, db, q, &rbacsDB); err != nil {
		return nil, err
	}

	for _, rbac := range rbacsDB {
		isValid, err := gorpmapping.CheckSignature(rbac, rbac.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac %s", rbac.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.get> rbac %s (%s) data corrupted", rbac.Name, rbac.ID)
			continue
		}
		for _, f := range opts {
			if err := f(ctx, db, &rbac); err != nil {
				return nil, err
			}
		}
		rs = append(rs, rbac.RBAC)
	}
	return rs, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.RBAC, error) {
	var r sdk.RBAC
	var rbacDB rbac
	found, err := gorpmapping.Get(ctx, db, q, &rbacDB)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(rbacDB, rbacDB.Signature)
	if err != nil {
		return nil, sdk.WrapError(err, "error when checking signature for rbac %s", rbacDB.ID)
	}
	if !isValid {
		log.Error(ctx, "rbac.get> rbac %s (%s) data corrupted", rbacDB.Name, rbacDB.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	for _, f := range opts {
		if err := f(ctx, db, &rbacDB); err != nil {
			return nil, err
		}
	}
	r = rbacDB.RBAC
	return &r, nil
}
