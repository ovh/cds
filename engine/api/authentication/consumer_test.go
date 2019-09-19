package authentication_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestConsumerLifecycle(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(db, &u))

	g1 := &sdk.Group{ID: 5, Name: "Five"}
	g2 := &sdk.Group{ID: 10, Name: "Ten"}

	c := sdk.AuthConsumer{
		Name:               sdk.RandomString(10),
		Description:        sdk.RandomString(10),
		Type:               sdk.ConsumerLocal,
		Scopes:             []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAdmin},
		GroupIDs:           []int64{g1.ID, g2.ID},
		AuthentifiedUserID: u.ID,
		IssuedAt:           time.Now(),
	}
	require.NoError(t, authentication.InsertConsumer(db, &c))

	// Invalidate group 1 should move the group id to invalid slice and add a warning
	require.NoError(t, authentication.ConsumerInvalidateGroupForUser(context.TODO(), db, g1, u.ID))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	require.Len(t, res.GroupIDs, 1)
	assert.Equal(t, g2.ID, res.GroupIDs[0])
	require.Len(t, res.InvalidGroupIDs, 1)
	assert.Equal(t, g1.ID, res.InvalidGroupIDs[0])
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)

	// Invalidate group 2 should move the group id to invalid slice, disable the consumer and add warnings
	require.NoError(t, authentication.ConsumerInvalidateGroupForUser(context.TODO(), db, g2, u.ID))
	res, err = authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.True(t, res.Disabled)
	require.Len(t, res.GroupIDs, 0)
	require.Len(t, res.InvalidGroupIDs, 2)
	assert.Equal(t, g1.ID, res.InvalidGroupIDs[0])
	assert.Equal(t, g2.ID, res.InvalidGroupIDs[1])
	require.Len(t, res.Warnings, 3)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[1].Type)
	assert.Equal(t, g2.ID, res.Warnings[1].GroupID)
	assert.Equal(t, g2.Name, res.Warnings[1].GroupName)
	assert.Equal(t, sdk.WarningLastGroupRemoved, res.Warnings[2].Type)

	// Remove group 1 should remove the group from the consumer, remove previous warning
	require.NoError(t, authentication.ConsumerRemoveGroup(context.TODO(), db, g1))
	res, err = authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.True(t, res.Disabled)
	require.Len(t, res.GroupIDs, 0)
	require.Len(t, res.InvalidGroupIDs, 1)
	assert.Equal(t, g2.ID, res.InvalidGroupIDs[0])
	require.Len(t, res.Warnings, 3)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[0].Type)
	assert.Equal(t, g2.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g2.Name, res.Warnings[0].GroupName)
	assert.Equal(t, sdk.WarningLastGroupRemoved, res.Warnings[1].Type)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[2].Type)
	assert.Equal(t, g1.ID, res.Warnings[2].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[2].GroupName)

	// Restore group 2 should remove warning, re-enable the consumer and set g2 id in consumer's groups
	require.NoError(t, authentication.ConsumerRestoreInvalidatedGroupForUser(context.TODO(), db, g2.ID, u.ID))
	res, err = authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.False(t, res.Disabled)
	require.Len(t, res.GroupIDs, 1)
	assert.Equal(t, g2.ID, res.GroupIDs[0])
	require.Len(t, res.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)

	// Remove group 2 should disable the consumer and remove the group from it
	require.NoError(t, authentication.ConsumerRemoveGroup(context.TODO(), db, g2))
	res, err = authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.True(t, res.Disabled)
	require.Len(t, res.GroupIDs, 0)
	require.Len(t, res.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 3)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[1].Type)
	assert.Equal(t, g2.ID, res.Warnings[1].GroupID)
	assert.Equal(t, g2.Name, res.Warnings[1].GroupName)
	assert.Equal(t, sdk.WarningLastGroupRemoved, res.Warnings[2].Type)
}
