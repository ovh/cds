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

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.Group{}, "group", true, "id"),
		gorpmapping.New(LinkGroupUser{}, "group_user", true, "id"),
	)
}
