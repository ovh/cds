package user_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticatedUserDAO(t *testing.T) {
	db, _, _ := test.SetupPG(t)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}
	test.NoError(t, user.Insert(db, &u))

	u1, err := user.LoadByID(context.TODO(), db, u.ID)
	test.NoError(t, err)
	test.Equal(t, u.Username, u1.Username)

	u.Username = sdk.RandomString(10)
	test.NoError(t, user.Update(db, &u))

	u1, err = user.LoadByID(context.TODO(), db, u.ID)
	test.NoError(t, err)
	test.Equal(t, u.Username, u1.Username)

	// Try to corrupt the data
	_, err = db.Exec("UPDATE authentified_user SET ring = $1 WHERE id = $2", sdk.UserRingMaintainer, u.ID)
	test.NoError(t, err)

	// Now the loading should failed
	_, err = user.LoadByID(context.TODO(), db, u.ID)
	test.Error(t, err)
	test.Equal(t, true, sdk.ErrorIs(err, sdk.ErrUserNotFound))

	test.NoError(t, user.DeleteByID(db, u.ID))
}

func TestLoadAll(t *testing.T) {
	db, _, _ := test.SetupPG(t)
	for i := 0; i < 10; i++ {
		var u = sdk.AuthentifiedUser{
			Username: sdk.RandomString(10),
			Fullname: sdk.RandomString(10),
			Ring:     sdk.UserRingAdmin,
		}

		assert.NoError(t, user.Insert(db, &u))
	}

	users, err := user.LoadAll(context.TODO(), db)
	test.NoError(t, err)

	assert.True(t, len(users) >= 10)
}

func TestLoadAllByIDs(t *testing.T) {
	db, _, _ := test.SetupPG(t)

	ids := make([]string, 3)
	for i := 0; i < 2; i++ {
		var u = sdk.AuthentifiedUser{
			Username: sdk.RandomString(10),
			Fullname: sdk.RandomString(10),
			Ring:     sdk.UserRingAdmin,
		}

		assert.NoError(t, user.Insert(db, &u))

		ids[i] = u.ID
	}
	ids[2] = sdk.UUID()

	users, err := user.LoadAllByIDs(context.TODO(), db, ids)
	test.NoError(t, err)

	test.Equal(t, 2, len(users))
}
