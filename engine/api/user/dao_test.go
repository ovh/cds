package user_test

import (
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticatedUserDAO(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username:     sdk.RandomString(10),
		Fullname:     sdk.RandomString(10),
		Ring:         sdk.UserRingAdmin,
		DateCreation: time.Now(),
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

func TestLoadContacts(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username:     sdk.RandomString(10),
		Fullname:     sdk.RandomString(10),
		Ring:         sdk.UserRingAdmin,
		DateCreation: time.Now(),
	}

	db, _, _ := test.SetupPG(t)
	assert.NoError(t, user.Insert(db, &u))

	var c = sdk.UserContact{
		UserID:         u.ID,
		PrimaryContact: true,
		Type:           sdk.UserContactTypeEmail,
		Value:          "test@lolcat.host",
	}
	assert.NoError(t, user.InsertContact(db, &c))

	u1, err := user.LoadUserByID(db, u.ID, user.LoadOptions.WithContacts)
	assert.NoError(t, err)
	assert.NotNil(t, u1)

	assert.Len(t, u1.Contacts, 1)

	assert.NoError(t, user.Delete(db, u1.ID))
}

func TestLoadDeprecatedUser(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username:     sdk.RandomString(10),
		Fullname:     sdk.RandomString(10),
		Ring:         sdk.UserRingAdmin,
		DateCreation: time.Now(),
	}

	db, _, _ := test.SetupPG(t)
	assert.NoError(t, user.Insert(db, &u))

	var c = sdk.UserContact{
		UserID:         u.ID,
		PrimaryContact: true,
		Type:           sdk.UserContactTypeEmail,
		Value:          "test@lolcat.host",
	}
	assert.NoError(t, user.InsertContact(db, &c))

	u1, err := user.LoadUserByID(db, u.ID, user.LoadOptions.WithOldUserStruct)
	assert.NoError(t, err)
	assert.NotNil(t, u1.OldUserStruct)

	assert.NoError(t, user.Delete(db, u1.ID))
}

func TestLoadAll(t *testing.T) {
	db, _, _ := test.SetupPG(t)
	for i := 0; i <= 10; i++ {
		var u = sdk.AuthentifiedUser{
			Username:     sdk.RandomString(10),
			Fullname:     sdk.RandomString(10),
			Ring:         sdk.UserRingAdmin,
			DateCreation: time.Now(),
		}

		assert.NoError(t, user.Insert(db, &u))

		var c = sdk.UserContact{
			UserID:         u.ID,
			PrimaryContact: true,
			Type:           sdk.UserContactTypeEmail,
			Value:          "test@lolcat.host",
		}
		assert.NoError(t, user.InsertContact(db, &c))
	}

	users, err := user.LoadAll(db, user.LoadOptions.WithContacts)
	assert.NoError(t, err)

	assert.True(t, len(users) >= 10)
}
