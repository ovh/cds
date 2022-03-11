package sdk

import "time"

const (
	RoleRead             = "read"
	RoleManage           = "manage"
	RoleDelete           = "delete"
	RoleCreateProject    = "createProject"
	RoleManagePermission = "managePermission"
)

type Rbac struct {
	UUID         string        `json:"uuid" db:"uuid" yaml:"-"`
	Name         string        `json:"name" db:"name" yaml:"name"`
	Created      time.Time     `json:"created" db:"created" yaml:"-"`
	LastModified time.Time     `json:"last_modified" db:"last_modified" yaml:"-"`
	Globals      []RbacGlobal  `json:"globals" db:"-" yaml:"globals"`
	Projects     []RbacProject `json:"projects" db:"-" yaml:"projects"`
}

func IsValidRbac(rbac *Rbac) error {
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
