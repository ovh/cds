package authentication_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestLoadSession(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u := sdk.AuthentifiedUser{Username: sdk.RandomString(10)}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c1, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)

	s1, err := authentication.NewSession(context.TODO(), db, &c1.AuthConsumer, time.Minute)
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, &c1.AuthConsumer, time.Minute)
	require.NoError(t, err)

	c2, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)
	s3, err := authentication.NewSession(context.TODO(), db, &c2.AuthConsumer, time.Minute)
	require.NoError(t, err)

	// LoadSessionByID
	_, err = authentication.LoadSessionByID(context.TODO(), db, sdk.RandomString(10))
	require.Error(t, err)
	res, err := authentication.LoadSessionByID(context.TODO(), db, s1.ID)
	require.NoError(t, err)
	require.Equal(t, res.ID, s1.ID)

	// LoadSessionByConsumerIDs
	_, err = authentication.LoadSessionsByConsumerIDs(context.TODO(), db, nil)
	require.NoError(t, err)
	ress, err := authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{c1.ID})
	require.NoError(t, err)
	require.Len(t, ress, 2)
	require.True(t, slices.ContainsFunc(ress, func(e sdk.AuthSession) bool { return e.ID == s1.ID }))
	require.True(t, slices.ContainsFunc(ress, func(e sdk.AuthSession) bool { return e.ID == s2.ID }))
	ress, err = authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{c1.ID, c2.ID})
	require.NoError(t, err)
	require.Len(t, ress, 3)
	require.True(t, slices.ContainsFunc(ress, func(e sdk.AuthSession) bool { return e.ID == s1.ID }))
	require.True(t, slices.ContainsFunc(ress, func(e sdk.AuthSession) bool { return e.ID == s2.ID }))
	require.True(t, slices.ContainsFunc(ress, func(e sdk.AuthSession) bool { return e.ID == s3.ID }))
}

func TestInsertSession(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)

	s, err := authentication.NewSession(context.TODO(), db, &c.AuthConsumer, time.Minute)
	require.NoError(t, err)

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	test.Equal(t, s, res)
}

func TestDeleteSession(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)

	s, err := authentication.NewSession(context.TODO(), db, &c.AuthConsumer, time.Minute)
	require.NoError(t, err)

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.NoError(t, authentication.DeleteSessionByID(db, s.ID))

	_, err = authentication.LoadSessionByID(context.TODO(), db, s.ID)
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
