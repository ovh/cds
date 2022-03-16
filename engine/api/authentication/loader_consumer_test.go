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

	res, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	require.NotNil(t, res.AuthentifiedUser)
	assert.Equal(t, u.Username, res.AuthentifiedUser.Username)

	require.Equal(t, 1, len(res.AuthentifiedUser.Groups))
	assert.Equal(t, g.ID, res.AuthentifiedUser.Groups[0].ID)
}

func TestWithConsumerGroups(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g1, g2)

	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser, authentication.LoadConsumerOptions.WithConsumerGroups)
	require.NoError(t, err)
	assert.NotNil(t, 0, len(localConsumer.Groups), "no group ids on local consumer so no groups are expected")

	consumerOptions := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{g1.ID, g2.ID},
		Scopes:   sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
	}
	newConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	builtinConsumer, err := authentication.LoadConsumerByID(context.TODO(), db, newConsumer.ID,
		authentication.LoadConsumerOptions.WithConsumerGroups)
	require.NoError(t, err)
	require.Equal(t, 2, len(builtinConsumer.Groups))
	sort.Slice(builtinConsumer.Groups, func(i, j int) bool { return builtinConsumer.Groups[i].ID < builtinConsumer.Groups[j].ID })
	assert.Equal(t, g1.ID, builtinConsumer.Groups[0].ID)
	assert.Equal(t, g2.ID, builtinConsumer.Groups[1].ID)
}
