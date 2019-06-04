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

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.Action{}, "group", true, "id"),
		gorpmapping.New(LinkGroupUser{}, "group_user", true, "id"),
	)
}
