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

// MigrationUser is the temporary link between a deprecated user and an authentified user.
type MigrationUser struct {
	ID                 int64  `db:"id"`
	AuthentifiedUserID string `db:"authentified_user_id"`
	UserID             int64  `db:"user_id"`
}

type migrationUser struct {
	MigrationUser
	gorpmapping.SignedEntity
}

func (m migrationUser) Canonical() ([]byte, error) {
	var canonical string
	canonical += m.AuthentifiedUserID
	canonical += fmt.Sprintf("%d", m.UserID)
	return []byte(canonical), nil
}

// MigrationUsers provides func for MigrationUser list.
type MigrationUsers []MigrationUser

// ToMapByAuthentifiedUserID returns a map of migrations indexed by authentified user ids.
func (m MigrationUsers) ToMapByAuthentifiedUserID() map[string]MigrationUser {
	mMigrations := make(map[string]MigrationUser, len(m))
	for i := range m {
		mMigrations[m[i].AuthentifiedUserID] = m[i]
	}
	return mMigrations
}

// ToMapByUserID returns a map of migrations indexed by deprecated user ids.
func (m MigrationUsers) ToMapByUserID() map[int64]MigrationUser {
	mMigrations := make(map[int64]MigrationUser, len(m))
	for i := range m {
		mMigrations[m[i].UserID] = m[i]
	}
	return mMigrations
}

// ToUserIDs returns a list of deprecated user ids for migration list.
func (m MigrationUsers) ToUserIDs() []int64 {
	ids := make([]int64, len(m))
	for i := range m {
		ids[i] = m[i].UserID
	}
	return ids
}

// ToAuthentifiedUserIDs returns a list of authentified user ids for migration list.
func (m MigrationUsers) ToAuthentifiedUserIDs() []string {
	ids := make([]string, len(m))
	for i := range m {
		ids[i] = m[i].AuthentifiedUserID
	}
	return ids
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
	gorpmapping.Register(gorpmapping.New(migrationUser{}, "authentified_user_migration", true, "id"))
	gorpmapping.Register(gorpmapping.New(deprecatedUser{}, "user", true, "id"))
}
