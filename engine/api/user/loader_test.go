package user_test

import (
	"context"
	"github.com/ovh/cds/engine/api/organization"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	db, _ := test.SetupPG(t)

	assert.NoError(t, user.Insert(context.TODO(), db, &u))

	c := sdk.UserContact{
		UserID:  u.ID,
		Primary: true,
		Type:    sdk.UserContactTypeEmail,
		Value:   u.Username + "@lolcat.local",
	}
	assert.NoError(t, user.InsertContact(context.TODO(), db, &c))

	result, err := user.LoadByID(context.TODO(), db, u.ID, user.LoadOptions.WithContacts)
	assert.NoError(t, err)
	assert.Len(t, result.Contacts, 1)
	assert.Equal(t, c.Value, result.Contacts[0].Value)

	assert.NoError(t, user.DeleteByID(db, result.ID))
}

func TestLoadOrganization(t *testing.T) {
	var u = sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
		Fullname: sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}

	db, _ := test.SetupPG(t)

	require.NoError(t, user.Insert(context.TODO(), db, &u))
	t.Cleanup(func() {
		require.NoError(t, user.DeleteByID(db, u.ID))
	})

	o := sdk.Organization{Name: sdk.RandomString(10)}
	require.NoError(t, organization.Insert(context.TODO(), db, &o))

	require.NoError(t, user.InsertUserOrganization(context.TODO(), db, &user.UserOrganization{
		AuthentifiedUserID: u.ID,
		OrganizationID:     o.ID,
	}))

	result, err := user.LoadByID(context.TODO(), db, u.ID, user.LoadOptions.WithOrganization)
	require.NoError(t, err)
	require.Equal(t, o.Name, result.Organization)
}
