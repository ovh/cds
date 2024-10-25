package authentication_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestWithAuthentifiedUser(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g)

	res, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	require.NotNil(t, res.AuthConsumerUser.AuthentifiedUser)
	assert.Equal(t, u.Username, res.AuthConsumerUser.AuthentifiedUser.Username)

	require.Equal(t, 1, len(res.AuthConsumerUser.AuthentifiedUser.Groups))
	assert.Equal(t, g.ID, res.AuthConsumerUser.AuthentifiedUser.Groups[0].ID)
}

func TestWithConsumerGroups(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g1, g2)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser, authentication.LoadUserConsumerOptions.WithConsumerGroups)
	require.NoError(t, err)
	assert.NotNil(t, 0, len(localConsumer.AuthConsumerUser.Groups), "no group ids on local consumer so no groups are expected")

	consumerOptions := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{g1.ID, g2.ID},
		Scopes:   sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
	}
	newConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	builtinConsumer, err := authentication.LoadUserConsumerByID(context.TODO(), db, newConsumer.ID,
		authentication.LoadUserConsumerOptions.WithConsumerGroups)
	require.NoError(t, err)
	require.Equal(t, 2, len(builtinConsumer.AuthConsumerUser.Groups))
	sort.Slice(builtinConsumer.AuthConsumerUser.Groups, func(i, j int) bool {
		return builtinConsumer.AuthConsumerUser.Groups[i].ID < builtinConsumer.AuthConsumerUser.Groups[j].ID
	})
	assert.Equal(t, g1.ID, builtinConsumer.AuthConsumerUser.Groups[0].ID)
	assert.Equal(t, g2.ID, builtinConsumer.AuthConsumerUser.Groups[1].ID)
}
