package sdk

import "time"

const (
	// Global Role
	GlobalRoleManagePermission   = "manage-permission"
	GlobalRoleManageOrganization = "manage-organization"
	GlobalRoleManageRegion       = "manage-region"
	GlobalRoleManageHatchery     = "manage-hatchery"
	GlobalRoleManageUser         = "manage-user"
	GlobalRoleManageGroup        = "manage-group"
	GlobalRoleProjectCreate      = "create-project"

	// Project Role
	ProjectRoleRead              = "read"
	ProjectRoleManage            = "manage"
	ProjectRoleManageWorkerModel = "manage-worker-model"

	// Hatchery Role
	HatcheryRoleSpawn = "start-worker"

	// Region Role
	RegionRoleList    = "list"
	RegionRoleExecute = "execute"
	RegionRoleManage  = "manage"
)

type RBAC struct {
	ID           string         `json:"id" db:"id"`
	Name         string         `json:"name" db:"name"`
	Created      time.Time      `json:"created" db:"created"`
	LastModified time.Time      `json:"last_modified" db:"last_modified"`
	Globals      []RBACGlobal   `json:"globals,omitempty" db:"-"`
	Projects     []RBACProject  `json:"projects,omitempty" db:"-"`
	Regions      []RBACRegion   `json:"regions,omitempty" db:"-"`
	Hatcheries   []RBACHatchery `json:"hatcheries,omitempty" db:"-"`
}

func IsValidRBAC(rbac *RBAC) error {
	if rbac.Name == "" {
		return WrapError(ErrInvalidData, "missing permission name")
	}
	for _, g := range rbac.Globals {
		if err := isValidRBACGlobal(rbac.Name, g); err != nil {
			return err
		}
	}
	for _, p := range rbac.Projects {
		if err := isValidRBACProject(rbac.Name, p); err != nil {
			return err
		}
	}
	for _, r := range rbac.Regions {
		if err := isValidRBACRegion(rbac.Name, r); err != nil {
			return err
		}
	}
	for _, h := range rbac.Hatcheries {
		if err := isValidRBACHatchery(rbac.Name, h); err != nil {
			return err
		}
	}
	return nil
}
