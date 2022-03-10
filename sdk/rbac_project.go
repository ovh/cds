package sdk

var (
	ProjectRoles = []string{RoleRead, RoleManage, RoleDelete}
)

type RbacProject struct {
	AbstractRbac
	All      bool                     `json:"all" db:"all"`
	Projects []RbacProjectIdentifiers `json:"projects" db:"-"`
}

type RbacProjectIdentifiers struct {
	ID            int64  `json:"-" db:"id"`
	RbacProjectID int64  `json:"-" db:"rbac_project_id"`
	ProjectID     int64  `json:"-" db:"project_id"`
	ProjectKey    string `json:"project_key" db:"-"`
	ProjectName   string `json:"project_name" db:"-"`
}

func isValidRbacProject(rbacName string, rp RbacProject) error {
	// Check empty group and users
	if len(rp.RbacGroups) == 0 && len(rp.RbacUsers) == 0 {
		return WrapError(ErrInvalidData, "Rbac %s: missing groups or users on project permission", rbacName)
	}

	// Check role
	if rp.Role == "" {
		return WrapError(ErrInvalidData, "Rbac %s: role for project permission cannot be empty", rbacName)
	}
	roleFound := false
	for _, r := range ProjectRoles {
		if r == rp.Role {
			roleFound = true
			break
		}
	}
	if !roleFound {
		return WrapError(ErrInvalidData, "Rbac %s: role %s is not allowed on a project permission", rbacName, rp.Role)
	}

	// Check project_ids and all flag
	if len(rp.Projects) == 0 && !rp.All {
		return WrapError(ErrInvalidData, "Rbac %s: must have at least 1 project on a project permission", rbacName)
	}
	if len(rp.Projects) > 0 && rp.All {
		return WrapError(ErrInvalidData, "Rbac %s: you can't have a list of project and the all flag checked on a project permission", rbacName)
	}
	return nil
}
