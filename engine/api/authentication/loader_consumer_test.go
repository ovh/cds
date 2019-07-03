package authentication_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestWithAuthentifiedUser(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(db, g)

	res, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	require.NotNil(t, res.AuthentifiedUser)
	assert.Equal(t, u.Username, res.AuthentifiedUser.Username)

	require.NotNil(t, res.AuthentifiedUser.OldUserStruct)
	require.Equal(t, 1, len(res.AuthentifiedUser.OldUserStruct.Groups))
	assert.Equal(t, g.ID, res.AuthentifiedUser.OldUserStruct.Groups[0].ID)
}
