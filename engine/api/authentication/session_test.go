package authentication_test

import (
	"context"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_CheckSessionJWT(t *testing.T) {
	_, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	now := time.Now()

	session := &sdk.AuthSession{
		ID:         sdk.UUID(),
		GroupIDs:   []int64{0, 1, 3},
		Scopes:     []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAdmin},
		ConsumerID: sdk.UUID(),
		Created:    now,
		ExpireAt:   now.Add(3 * time.Second),
	}

	jwtRaw, err := authentication.NewSessionJWT(session)
	require.NoError(t, err)

	_, err = authentication.CheckSessionJWT(jwtRaw)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	_, err = authentication.CheckSessionJWT(jwtRaw)
	require.Error(t, err)
	jwtErr, ok := sdk.Cause(err).(*jwt.ValidationError)
	require.True(t, ok)
	require.Equal(t, jwt.ValidationErrorExpired, jwtErr.Errors)
}

func Test_SessionCleaner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
	defer cancel()
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	authentication.SessionCleaner(ctx, func() *gorp.DbMap { return db })
}
