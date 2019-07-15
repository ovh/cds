package user

import (
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type authentifiedUser struct {
	sdk.AuthentifiedUser
	gorpmapping.SignedEntity
}

func (u authentifiedUser) Canonical() ([]byte, error) {
	var canonical string
	canonical += u.ID
	canonical += u.Username
	canonical += u.Fullname
	canonical += u.Ring
	canonical += u.Created.In(time.UTC).Format(time.RFC3339)
	return []byte(canonical), nil
}

type userContact struct {
	sdk.UserContact
	gorpmapping.SignedEntity
}

func (c userContact) Canonical() ([]byte, error) {
	var canonical string
	canonical += fmt.Sprintf("%d", c.ID)
	canonical += c.UserID
	canonical += c.Type
	canonical += c.Value
	canonical += fmt.Sprintf("%t", c.Primary)
	canonical += fmt.Sprintf("%t", c.Verified)
	return []byte(canonical), nil
}

type authentifiedUserMigration struct {
	AuthentifiedUserID string `db:"authentified_user_id"`
	UserID             int64  `db:"user_id"`
}

type deprecatedUser struct {
	ID       int64    `db:"id"`
	Username string   `db:"username"`
	Admin    bool     `db:"admin"`
	Origin   string   `db:"origin"`
	Data     sdk.User `db:"data"`
}

func init() {
	gorpmapping.Register(gorpmapping.New(authentifiedUser{}, "authentified_user", false, "id"))
	gorpmapping.Register(gorpmapping.New(userContact{}, "user_contact", true, "id"))
	gorpmapping.Register(gorpmapping.New(authentifiedUserMigration{}, "authentified_user_migration", false, "authentified_user_id", "user_id"))
	gorpmapping.Register(gorpmapping.New(deprecatedUser{}, "user", true, "id"))
}
