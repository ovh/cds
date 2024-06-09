package authentication_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSession(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	u := sdk.AuthentifiedUser{Username: sdk.RandomString(10)}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c1, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)

	s1, err := authentication.NewSession(context.TODO(), db, &c1.AuthConsumer, time.Second)
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, &c1.AuthConsumer, time.Second)
	require.NoError(t, err)

	c2, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)
	s3, err := authentication.NewSession(context.TODO(), db, &c2.AuthConsumer, time.Second)
	require.NoError(t, err)

	// LoadSessionByID
	res, err := authentication.LoadSessionByID(context.TODO(), db, sdk.RandomString(10))
	assert.Error(t, err)
	res, err = authentication.LoadSessionByID(context.TODO(), db, s1.ID)
	assert.NoError(t, err)
	assert.Equal(t, res.ID, s1.ID)

	// LoadSessionByConsumerIDs
	ress, err := authentication.LoadSessionsByConsumerIDs(context.TODO(), db, nil)
	assert.NoError(t, err)
	ress, err = authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{c1.ID})
	assert.NoError(t, err)
	require.Equal(t, 2, len(ress))
	assert.Equal(t, s1.ID, ress[0].ID)
	assert.Equal(t, s2.ID, ress[1].ID)
	ress, err = authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{c1.ID, c2.ID})
	assert.NoError(t, err)
	require.Equal(t, len(ress), 3)
	assert.Equal(t, s1.ID, ress[0].ID)
	assert.Equal(t, s2.ID, ress[1].ID)
	assert.Equal(t, s3.ID, ress[2].ID)
}

func TestInsertSession(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(context.TODO(), db, &u))

	c, err := local.NewConsumer(context.TODO(), db, u.ID)
	test.NoError(t, err)

	s, err := authentication.NewSession(context.TODO(), db, &c.AuthConsumer, time.Second)
	test.NoError(t, err)

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, s, res)
}

func TestDeleteSession(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)

	s, err := authentication.NewSession(context.TODO(), db, &c.AuthConsumer, time.Second)
	require.NoError(t, err)

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.NoError(t, authentication.DeleteSessionByID(db, s.ID))

	res, err = authentication.LoadSessionByID(context.TODO(), db, s.ID)
	require.Error(t, err)
}

func Test_GetAndDeleteCorruptedSessions(t *testing.T) {
	db, _ := test.SetupPG(t)

	sessions, err := authentication.UnsafeLoadCorruptedSessions(context.TODO(), db)
	require.NoError(t, err)
	for _, s := range sessions {
		err := authentication.DeleteSessionByID(db, s.ID)
		require.NoError(t, err)
	}
}
