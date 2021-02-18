package authentication_test

import (
	"context"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func Test_CheckSessionJWT(t *testing.T) {
	_, _ = test.SetupPG(t, bootstrap.InitiliazeDB)
	now := time.Now()

	session := &sdk.AuthSession{
		ID:         sdk.UUID(),
		ConsumerID: sdk.UUID(),
		Created:    now,
		ExpireAt:   now.Add(3 * time.Second),
	}

	jwtRaw, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	_, _, err = service.CheckSessionJWT(jwtRaw, authentication.VerifyJWT)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	_, _, err = service.CheckSessionJWT(jwtRaw, authentication.VerifyJWT)
	require.Error(t, err)
	jwtErr, ok := sdk.Cause(err).(*jwt.ValidationError)
	require.True(t, ok)
	require.Equal(t, jwt.ValidationErrorExpired, jwtErr.Errors)
}

func Test_SessionCleaner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	authentication.SessionCleaner(ctx, func() *gorp.DbMap { return db.DbMap }, 1*time.Second)
}

func Test_CheckSession(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)

	u := sdk.AuthentifiedUser{Username: sdk.RandomString(10)}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	c, err := local.NewConsumer(context.TODO(), db, u.ID)
	require.NoError(t, err)

	// Create and check a non MFA session, it should be valid and no activity should be stored
	s, err := authentication.NewSession(context.TODO(), db, c, time.Second)
	require.NoError(t, err)
	r, err := authentication.CheckSession(context.TODO(), db, store, s.ID)
	require.NoError(t, err)
	require.Equal(t, s.ID, r.ID)
	a, err := authentication.GetSessionActivity(store, s.ID)
	require.NoError(t, err)
	require.False(t, a)

	// Create and check a MFA session
	s, err = authentication.NewSessionWithMFACustomDuration(context.TODO(), db, store, c, 5*time.Second, time.Second)
	require.NoError(t, err)
	a, err = authentication.GetSessionActivity(store, s.ID)
	require.NoError(t, err)
	require.True(t, a, "activity should be initially set by NewSession func")
	r, err = authentication.CheckSessionWithCustomMFADuration(context.TODO(), db, store, s.ID, time.Second)
	require.NoError(t, err)
	require.Equal(t, s.ID, r.ID, "check session should be valid for 1 second")
	time.Sleep(time.Second)
	_, err = authentication.CheckSessionWithCustomMFADuration(context.TODO(), db, store, s.ID, time.Second)
	require.Error(t, err, "session activity is expired, session should be removed from database")
	a, err = authentication.GetSessionActivity(store, s.ID)
	require.NoError(t, err)
	require.False(t, a, "activity should not be in cache anymore")
	_, err = authentication.LoadSessionByID(context.TODO(), db, s.ID)
	require.Error(t, err, "session should not be in database anymore")
}
