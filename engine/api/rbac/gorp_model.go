package rbac

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type rbac struct {
	sdk.Rbac
	gorpmapper.SignedEntity
}

func (r rbac) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.UUID}}{{.Name}}",
	}
}

type rbacGlobal struct {
	sdk.RbacGlobal
	gorpmapper.SignedEntity
}

func (rg rbacGlobal) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacUUID}}{{.Role}}",
	}
}

type rbacGlobalUser struct {
	sdk.RbacUser
	RbacGlobalID int64 `db:"rbac_global_id"`
	gorpmapper.SignedEntity
}

func (rgu rbacGlobalUser) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacGlobalID}}{{.UserID}}",
	}
}

type rbacGlobalGroup struct {
	sdk.RbacGroup
	RbacGlobalID int64 `db:"rbac_global_id"`
	gorpmapper.SignedEntity
}

func (rgg rbacGlobalGroup) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacGlobalID}}{{.GroupID}}",
	}
}

type rbacProject struct {
	sdk.RbacProject
	gorpmapper.SignedEntity
}

func (rp rbacProject) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacUUID}}{{.Role}}{{.All}}",
	}
}

type rbacProjectID struct {
	sdk.RbacProjectIdentifiers
	gorpmapper.SignedEntity
}

func (rpi rbacProjectID) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacProjectID}}{{.ProjectID}}",
	}
}

type rbacProjectUser struct {
	sdk.RbacUser
	RbacProjectID int64 `db:"rbac_project_id"`
	gorpmapper.SignedEntity
}

func (rgu rbacProjectUser) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacProjectID}}{{.UserID}}",
	}
}

type rbacProjectGroup struct {
	sdk.RbacGroup
	RbacProjectID int64 `db:"rbac_project_id"`
	gorpmapper.SignedEntity
}

func (rgg rbacProjectGroup) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacProjectID}}{{.GroupID}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(rbac{}, "rbac", false, "uuid"))
	gorpmapping.Register(gorpmapping.New(rbacGlobal{}, "rbac_global", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacGlobalUser{}, "rbac_global_users", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacGlobalGroup{}, "rbac_global_groups", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProject{}, "rbac_project", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProjectID{}, "rbac_project_ids", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProjectUser{}, "rbac_project_users", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProjectGroup{}, "rbac_project_groups", true, "id"))
}
