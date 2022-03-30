package rbac

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type rbac struct {
	sdk.RBAC
	gorpmapper.SignedEntity
}

func (r rbac) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{r.ID, r.Name}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}",
	}
}

type rbacGlobal struct {
	ID     int64  `db:"id"`
	RbacID string `db:"rbac_id"`
	sdk.RBACGlobal
	gorpmapper.SignedEntity
}

func (rg rbacGlobal) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{rg.ID, rg.RbacID, rg.Role}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacID}}{{.Role}}",
	}
}

type rbacGlobalUser struct {
	ID               int64  `db:"id"`
	RbacGlobalID     int64  `db:"rbac_global_id"`
	RbacGlobalUserID string `db:"user_id"`
	gorpmapper.SignedEntity
}

func (rgu rbacGlobalUser) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{rgu.ID, rgu.RbacGlobalID, rgu.RbacGlobalUserID}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacGlobalID}}{{.RbacGlobalUserID}}",
	}
}

type rbacGlobalGroup struct {
	ID                int64 `json:"-" db:"id"`
	RbacGlobalID      int64 `db:"rbac_global_id"`
	RbacGlobalGroupID int64 `db:"group_id"`
	gorpmapper.SignedEntity
}

func (rgg rbacGlobalGroup) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{rgg.ID, rgg.RbacGlobalID, rgg.RbacGlobalGroupID}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacGlobalID}}{{.RbacGlobalGroupID}}",
	}
}

type rbacProject struct {
	ID     int64  `json:"-" db:"id"`
	RbacID string `json:"-" db:"rbac_id"`
	sdk.RBACProject
	gorpmapper.SignedEntity
}

func (rp rbacProject) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{rp.ID, rp.RbacID, rp.Role, rp.All}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacID}}{{.Role}}{{.All}}",
	}
}

type rbacProjectKey struct {
	ID            int64  `json:"-" db:"id"`
	RbacProjectID int64  `json:"-" db:"rbac_project_id"`
	ProjectKey    string `json:"-" db:"project_key"`
	gorpmapper.SignedEntity
}

func (rpi rbacProjectKey) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{rpi.ID, rpi.RbacProjectID, rpi.ProjectKey}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacProjectID}}{{.ProjectKey}}",
	}
}

type rbacProjectUser struct {
	ID                int64  `json:"-" db:"id"`
	RbacProjectID     int64  `db:"rbac_project_id"`
	RbacProjectUserID string `db:"user_id"`
	gorpmapper.SignedEntity
}

func (rgu rbacProjectUser) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{rgu.ID, rgu.RbacProjectID, rgu.RbacProjectUserID}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacProjectID}}{{.RbacProjectUserID}}",
	}
}

type rbacProjectGroup struct {
	ID                 int64 `json:"-" db:"id"`
	RbacProjectID      int64 `db:"rbac_project_id"`
	RbacProjectGroupID int64 `db:"group_id"`
	gorpmapper.SignedEntity
}

func (rgg rbacProjectGroup) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{rgg.ID, rgg.RbacProjectID, rgg.RbacProjectGroupID}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.RbacProjectID}}{{.RbacProjectGroupID}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(rbac{}, "rbac", false, "id"))
	gorpmapping.Register(gorpmapping.New(rbacGlobal{}, "rbac_global", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacGlobalUser{}, "rbac_global_users", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacGlobalGroup{}, "rbac_global_groups", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProject{}, "rbac_project", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProjectKey{}, "rbac_project_keys", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProjectUser{}, "rbac_project_users", true, "id"))
	gorpmapping.Register(gorpmapping.New(rbacProjectGroup{}, "rbac_project_groups", true, "id"))
}
