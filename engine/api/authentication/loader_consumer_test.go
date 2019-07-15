package authentication_test

import (
	"context"
	"sort"
	"testing"

	"github.com/ovh/cds/engine/api/authentication/builtin"
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

func TestWithConsumerGroups(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(db, g1, g2)

	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser, authentication.LoadConsumerOptions.WithConsumerGroups)
	require.NoError(t, err)
	assert.NotNil(t, 0, len(localConsumer.Groups), "no group ids on local consumer so no groups are expected")

	newConsumer, _, err := builtin.NewConsumer(db, sdk.RandomString(10), sdk.RandomString(10), localConsumer,
		[]int64{g1.ID, g2.ID}, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAccessToken})
	require.NoError(t, err)
	builtinConsumer, err := authentication.LoadConsumerByID(context.TODO(), db, newConsumer.ID,
		authentication.LoadConsumerOptions.WithConsumerGroups)
	require.NoError(t, err)
	require.Equal(t, 2, len(builtinConsumer.Groups))
	sort.Slice(builtinConsumer.Groups, func(i, j int) bool { return builtinConsumer.Groups[i].ID < builtinConsumer.Groups[j].ID })
	assert.Equal(t, g1.ID, builtinConsumer.Groups[0].ID)
	assert.Equal(t, g2.ID, builtinConsumer.Groups[1].ID)
}
