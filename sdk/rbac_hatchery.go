package sdk

var (
	HatcheryRoles = StringSlice{HatcheryRoleSpawn}
)

type RBACHatchery struct {
	Role       string `json:"role" db:"role"`
	HatcheryID string `json:"hatchery_id" db:"hatchery_id"`
	RegionID   string `json:"region_id" db:"region_id"`

	RegionName   string `json:"region" db:"-"`
	HatcheryName string `json:"hatchery" db:"-"`
}

func isValidRBACHatchery(rbacName string, rbacHatchery RBACHatchery) error {
	// Check empty hatchery
	if rbacHatchery.HatcheryID == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: missing hatchery", rbacName)
	}
	if rbacHatchery.RegionID == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: missing region", rbacName)
	}

	// Check role
	if rbacHatchery.Role == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role for hatchery permission cannot be empty", rbacName)
	}
	if !HatcheryRoles.Contains(rbacHatchery.Role) {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role %s is not allowed on a hatchery permission", rbacName, rbacHatchery.Role)
	}
	return nil
}
