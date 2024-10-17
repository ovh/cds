package sdk

var (
	RegionProjectRoles = StringSlice{RegionRoleExecute}
)

type RBACRegionProject struct {
	Role            string   `json:"role" db:"role"`
	RegionID        string   `json:"region_id" db:"region_id"`
	AllProjects     bool     `json:"all_projects,omitempty" db:"all_projects"`
	RBACProjectKeys []string `json:"projects,omitempty" db:"-"`
	RegionName      string   `json:"region,omitempty"  db:"-"`
}
