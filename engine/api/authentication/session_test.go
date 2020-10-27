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

	jwtRaw, err := authentication.NewSessionJWT(session)
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
