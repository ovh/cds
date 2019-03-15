package user_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticatedUserDAO(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
		Email:    sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Origin:   sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}

	db, _, _ := test.SetupPG(t)

	assert.NoError(t, user.Insert(db, &u))
	assert.NoError(t, user.Update(db, &u))

	u1, err := user.LoadUserByID(db, u.ID)
	assert.NoError(t, err)
	assert.NotNil(t, u1)

	// Try to corrupt the data
	_, err = db.Exec("UPDATE authentified_user SET ring = $1 WHERE id = $2", sdk.UserRingMaintainer, u.ID)
	assert.NoError(t, err)

	// Now the loading should failed
	u2, err := user.LoadUserByID(db, u.ID)
	assert.Error(t, err)
	assert.Nil(t, u2)

	assert.NoError(t, user.Delete(db, u.ID))
}
