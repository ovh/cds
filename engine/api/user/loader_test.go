package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestLoadContacts(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}

	db, _, _ := test.SetupPG(t)
	assert.NoError(t, user.Insert(context.TODO(), db, &u))

	c := sdk.UserContact{
		UserID:  u.ID,
		Primary: true,
		Type:    sdk.UserContactTypeEmail,
		Value:   u.Username + "@lolcat.host",
	}
	assert.NoError(t, user.InsertContact(context.TODO(), db, &c))

	result, err := user.LoadByID(context.TODO(), db, u.ID, user.LoadOptions.WithContacts)
	assert.NoError(t, err)
	assert.Len(t, result.Contacts, 1)
	assert.Equal(t, c.Value, result.Contacts[0].Value)

	assert.NoError(t, user.DeleteByID(db, result.ID))
}
