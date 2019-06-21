package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestLoadContacts(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username:     sdk.RandomString(10),
		Fullname:     sdk.RandomString(10),
		Ring:         sdk.UserRingAdmin,
		DateCreation: time.Now(),
	}

	db, _, _ := test.SetupPG(t)
	assert.NoError(t, user.Insert(db, &u))

	c := sdk.UserContact{
		UserID:         u.ID,
		PrimaryContact: true,
		Type:           sdk.UserContactTypeEmail,
		Value:          u.Username + "@lolcat.host",
	}
	assert.NoError(t, user.InsertContact(db, &c))

	u1, err := user.LoadByID(context.TODO(), db, u.ID, user.LoadOptions.WithContacts)
	assert.NoError(t, err)
	assert.NotNil(t, u1)
	assert.Len(t, u1.Contacts, 1)

	assert.NoError(t, user.DeleteByID(db, u1.ID))
}
