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
	UUID         string        `json:"uuid" db:"uuid"`
	Name         string        `json:"name" db:"name"`
	Created      time.Time     `json:"created" db:"created"`
	LastModified time.Time     `json:"last_modified" db:"last_modified"`
	Globals      []RbacGlobal  `json:"globals" db:"-"`
	Projects     []RbacProject `json:"projects" db:"-"`
}

type AbstractRbac struct {
	ID         int64       `json:"-" db:"id"`
	RbacUUID   string      `json:"-" db:"rbac_uuid"`
	Role       string      `json:"role" db:"role"`
	RbacUsers  []RbacUser  `json:"users" db:"-"`
	RbacGroups []RbacGroup `json:"groups" db:"-"`
}

type RbacUser struct {
	ID     string `json:"-" db:"id"`
	UserID string `json:"-" db:"user_id"`
	Name   string `json:"name" db:"-"`
}

type RbacGroup struct {
	ID      string `json:"-" db:"id"`
	GroupID int64  `json:"-" db:"group_id"`
	Name    string `json:"name" db:"-"`
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
