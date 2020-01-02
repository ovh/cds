package group

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LinkGroupUser struct for database entity of group_user table.
type LinkGroupUser struct {
	ID      int64 `db:"id"`
	GroupID int64 `db:"group_id"`
	UserID  int64 `db:"user_id"`
	Admin   bool  `db:"group_admin"`
}

// LinksGroupUser struct.
type LinksGroupUser []LinkGroupUser

// ToUserIDs returns user ids for given links.
func (l LinksGroupUser) ToUserIDs() []int64 {
	ids := make([]int64, len(l))
	for i := range l {
		ids[i] = l[i].UserID
	}
	return ids
}

// ToGroupIDs returns group ids for given links.
func (l LinksGroupUser) ToGroupIDs() []int64 {
	ids := make([]int64, len(l))
	for i := range l {
		ids[i] = l[i].GroupID
	}
	return ids
}

// LinkGroupProject struct for database entity of project_group table.
type LinkGroupProject struct {
	ID        int64 `db:"id"`
	GroupID   int64 `db:"group_id"`
	ProjectID int64 `db:"project_id"`
	Role      int   `db:"role"`
}

// LinksGroupProject struct.
type LinksGroupProject []LinkGroupProject

// ToProjectIDs returns project ids for given links.
func (l LinksGroupProject) ToProjectIDs() []int64 {
	ids := make([]int64, len(l))
	for i := range l {
		ids[i] = l[i].ProjectID
	}
	return ids
}

// ToMapByProjectID groups links by project id in a map.
func (l LinksGroupProject) ToMapByProjectID() map[int64]LinksGroupProject {
	m := make(map[int64]LinksGroupProject)
	for i := range l {
		if _, ok := m[l[i].ProjectID]; !ok {
			m[l[i].ProjectID] = LinksGroupProject{}
		}
		m[l[i].ProjectID] = append(m[l[i].ProjectID], l[i])
	}
	return m
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.Group{}, "group", true, "id"),
		gorpmapping.New(LinkGroupUser{}, "group_user", true, "id"),
		gorpmapping.New(LinkGroupProject{}, "project_group", true, "id"),
	)
}
