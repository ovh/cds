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

// LoadEntriesByUserID returns, gathered in a single RBAC, only the entries that name the given user,
// directly or through one of their groups. Entries granted to all users are not included.
func LoadEntriesByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) (sdk.RBAC, error) {
	var ids entryIDs

	globalUsers, err := loadRBACGlobalUsersByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range globalUsers {
		ids.global = append(ids.global, x.RbacGlobalID)
	}
	globalGroups, err := loadRBACGlobalGroupsByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range globalGroups {
		ids.global = append(ids.global, x.RbacGlobalID)
	}

	projectUsers, err := loadRBACProjectUsersByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range projectUsers {
		ids.projects = append(ids.projects, x.RbacProjectID)
	}
	projectGroups, err := loadRBACProjectGroupsByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range projectGroups {
		ids.projects = append(ids.projects, x.RbacProjectID)
	}

	regionUsers, err := loadRBACRegionUsersByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range regionUsers {
		ids.regions = append(ids.regions, x.RbacRegionID)
	}
	regionGroups, err := loadRBACRegionGroupsByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range regionGroups {
		ids.regions = append(ids.regions, x.RbacRegionID)
	}

	workflowUsers, err := loadRBACWorkflowUsersByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range workflowUsers {
		ids.workflows = append(ids.workflows, x.RbacWorkflowID)
	}
	workflowGroups, err := loadRBACWorkflowGroupsByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range workflowGroups {
		ids.workflows = append(ids.workflows, x.RbacWorkflowID)
	}

	vsUsers, err := loadRBACVariableSetUsersByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range vsUsers {
		ids.variableSets = append(ids.variableSets, x.RbacVariableSetID)
	}
	vsGroups, err := loadRBACVariableSetGroupsByUserID(ctx, db, userID)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range vsGroups {
		ids.variableSets = append(ids.variableSets, x.RbacVariableSetID)
	}

	return loadEntriesByIDs(ctx, db, ids)
}

// LoadEntriesByGroupID returns, gathered in a single RBAC, only the entries that name the given group.
func LoadEntriesByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64) (sdk.RBAC, error) {
	var ids entryIDs
	groupIDs := []int64{groupID}

	globalGroups, err := loadRBACGlobalGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range globalGroups {
		ids.global = append(ids.global, x.RbacGlobalID)
	}

	projectGroups, err := loadRBACProjectGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range projectGroups {
		ids.projects = append(ids.projects, x.RbacProjectID)
	}

	regionGroups, err := loadRBACRegionGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range regionGroups {
		ids.regions = append(ids.regions, x.RbacRegionID)
	}

	workflowGroups, err := loadRBACWorkflowGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range workflowGroups {
		ids.workflows = append(ids.workflows, x.RbacWorkflowID)
	}

	vsGroups, err := loadRBACVariableSetGroupsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return sdk.RBAC{}, err
	}
	for _, x := range vsGroups {
		ids.variableSets = append(ids.variableSets, x.RbacVariableSetID)
	}

	return loadEntriesByIDs(ctx, db, ids)
}

type entryIDs struct {
	global, projects, regions, workflows, variableSets []int64
}

// loadEntriesByIDs loads the given entries without their users and groups.
func loadEntriesByIDs(ctx context.Context, db gorp.SqlExecutor, ids entryIDs) (sdk.RBAC, error) {
	var rb sdk.RBAC

	if len(ids.global) > 0 {
		globals, err := loadRBACGlobalsByIDs(ctx, db, ids.global)
		if err != nil {
			return rb, err
		}
		for _, g := range globals {
			rb.Global = append(rb.Global, g.RBACGlobal)
		}
	}

	if len(ids.projects) > 0 {
		projects, err := loadRBACProjectsByIDs(ctx, db, ids.projects)
		if err != nil {
			return rb, err
		}
		projectKeys, err := loadAllRBACProjectKeys(ctx, db, ids.projects)
		if err != nil {
			return rb, err
		}
		keysByProjectID := make(map[int64][]string)
		for _, k := range projectKeys {
			keysByProjectID[k.RbacProjectID] = append(keysByProjectID[k.RbacProjectID], k.ProjectKey)
		}
		for _, p := range projects {
			p.RBACProjectKeys = keysByProjectID[p.ID]
			rb.Projects = append(rb.Projects, p.RBACProject)
		}
	}

	if len(ids.regions) > 0 {
		regions, err := loadRBACRegionsByIDs(ctx, db, ids.regions)
		if err != nil {
			return rb, err
		}
		rb.Regions = regions
	}

	if len(ids.workflows) > 0 {
		workflows, err := loadRBACWorkflowsByIDs(ctx, db, ids.workflows)
		if err != nil {
			return rb, err
		}
		for _, w := range workflows {
			rb.Workflows = append(rb.Workflows, w.RBACWorkflow)
		}
	}

	if len(ids.variableSets) > 0 {
		variableSets, err := loadRBACVariableSetsByIDs(ctx, db, ids.variableSets)
		if err != nil {
			return rb, err
		}
		for _, vs := range variableSets {
			rb.VariableSets = append(rb.VariableSets, vs.RBACVariableSet)
		}
	}

	return rb, nil
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
