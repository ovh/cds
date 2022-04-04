package sdk

import "time"

const (
	RoleRead             = "read"
	RoleManage           = "manage"
	RoleDelete           = "delete"
	RoleCreateProject    = "create-project"
	RoleManagePermission = "manage-permission"
)

type RBAC struct {
	ID           string        `json:"id" db:"id"`
	Name         string        `json:"name" db:"name"`
	Created      time.Time     `json:"created" db:"created"`
	LastModified time.Time     `json:"last_modified" db:"last_modified"`
	Globals      []RBACGlobal  `json:"globals" db:"-"`
	Projects     []RBACProject `json:"projects" db:"-"`
}

func IsValidRbac(rbac *RBAC) error {
	if rbac.Name == "" {
		return WrapError(ErrInvalidData, "missing permission name")
	}
	for _, g := range rbac.Globals {
		if err := isValidRbacGlobal(rbac.Name, g); err != nil {
			return err
		}
	}
	for _, p := range rbac.Projects {
		if err := isValidRbacProject(rbac.Name, p); err != nil {
			return err
		}
	}
	return nil
}
