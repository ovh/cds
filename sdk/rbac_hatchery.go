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
