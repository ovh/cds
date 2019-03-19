package localauthentication_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/localauthentication"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestLocalAutenticationDAO(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
		Email:    sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Origin:   sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}

	db, _, _ := test.SetupPG(t)

	assert.NoError(t, user.Insert(db, &u))

	var localAuth = sdk.UserLocalAuthentication{
		UserID:        u.ID,
		ClearPassword: sdk.RandomString(10),
	}

	assert.NoError(t, localauthentication.Insert(db, &localAuth))

	ok, err := localauthentication.Authentify(db, u.Username, localAuth.ClearPassword)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = localauthentication.Authentify(db, u.Username, "wrong password")
	assert.NoError(t, err)
	assert.False(t, ok)

	err = localauthentication.Delete(db, u.ID)
	assert.NoError(t, err)
}
