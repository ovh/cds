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

// Given a consumer with two groups, if we invalidate one it should be invalidated and one warning should be set.
func TestConsumerInvalidateGroupForUser_InvalidateOneConsumerGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}
	g2 := &sdk.Group{ID: 10, Name: "B"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g1.ID, g2.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Invalidate group 1 should move the group id to invalid slice and add a warning
	require.NoError(t, authentication.ConsumerInvalidateGroupForUser(context.TODO(), db, g1, &u))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 1)
	assert.Equal(t, g2.ID, res.AuthConsumerUser.GroupIDs[0])
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 1)
	assert.Equal(t, g1.ID, res.AuthConsumerUser.InvalidGroupIDs[0])
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)
}

// Given a consumer with two groups, if we invalidate one it should not be invalidated if the user is an admin.
func TestConsumerInvalidateGroupForUser_InvalidateOneConsumerGroupForAdmin(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
		Ring:     sdk.UserRingAdmin,
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}
	g2 := &sdk.Group{ID: 10, Name: "B"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g1.ID, g2.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Invalidate group 1 should move the group id to invalid slice and add a warning
	require.NoError(t, authentication.ConsumerInvalidateGroupForUser(context.TODO(), db, g1, &u))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 2)
	assert.Equal(t, g1.ID, res.AuthConsumerUser.GroupIDs[0])
	assert.Equal(t, g2.ID, res.AuthConsumerUser.GroupIDs[1])
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 0)
}

// Given a consumer with one group, if we invalidate the group it should disable the consumer and add two warnings.
func TestConsumerInvalidateGroupForUser_InvalidateLastConsumerGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g1.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Invalidate group 1 should move the group id to invalid slice, disable the consumer and add warnings
	require.NoError(t, authentication.ConsumerInvalidateGroupForUser(context.TODO(), db, g1, &u))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.True(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 0)
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 1)
	assert.Equal(t, g1.ID, res.AuthConsumerUser.InvalidGroupIDs[0])
	require.Len(t, res.Warnings, 2)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)
	assert.Equal(t, sdk.WarningLastGroupRemoved, res.Warnings[1].Type)
}

// Given a consumer with two groups, if we remove one a warning should be set.
func TestConsumerRemoveGroup_RemoveOneConsumerGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}
	g2 := &sdk.Group{ID: 10, Name: "B"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g1.ID, g2.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Remove group 1 should remove the group from the consumer, remove previous warning
	require.NoError(t, authentication.ConsumerRemoveGroup(context.TODO(), db, g1))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.False(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 1)
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)
}

// Given a consumer with a valid and an invalid group, if we remove the invalid one a warning should be set to replace previous warning.
func TestConsumerRemoveGroup_RemoveOneInvalidConsumerGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}
	g2 := &sdk.Group{ID: 10, Name: "B"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		Warnings: sdk.AuthConsumerWarnings{{
			Type:      sdk.WarningGroupInvalid,
			GroupID:   g1.ID,
			GroupName: g1.Name,
		}},
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g2.ID},
			InvalidGroupIDs:    []int64{g1.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Remove group 1 should remove the group from the consumer, remove previous warning
	require.NoError(t, authentication.ConsumerRemoveGroup(context.TODO(), db, g1))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.False(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 1)
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)
}

// Given a consumer with one group, if we remove the group it should disable the consumer and add two warnings.
func TestConsumerRemoveGroup_RemoveLastConsumerGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g1.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Remove group 1 should remove the group from the consumer, remove previous warning
	require.NoError(t, authentication.ConsumerRemoveGroup(context.TODO(), db, g1))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.True(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 0)
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 2)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[0].Type)
	assert.Equal(t, g1.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[0].GroupName)
	assert.Equal(t, sdk.WarningLastGroupRemoved, res.Warnings[1].Type)
}

// Given a consumer with one invalid group, if we remove the group it should disable the consumer and add two warnings.
func TestConsumerRemoveGroup_RemoveLastInvalidConsumerGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		Disabled:        true,
		Warnings: sdk.AuthConsumerWarnings{
			{
				Type:      sdk.WarningGroupInvalid,
				GroupID:   g1.ID,
				GroupName: g1.Name,
			},
			{
				Type: sdk.WarningLastGroupRemoved,
			},
		},
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			InvalidGroupIDs:    []int64{g1.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Remove group 1 should remove the group from the consumer, remove previous warning
	require.NoError(t, authentication.ConsumerRemoveGroup(context.TODO(), db, g1))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.True(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 0)
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 2)
	assert.Equal(t, sdk.WarningLastGroupRemoved, res.Warnings[0].Type)
	assert.Equal(t, sdk.WarningGroupRemoved, res.Warnings[1].Type)
	assert.Equal(t, g1.ID, res.Warnings[1].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[1].GroupName)
}

// Given a consumer with a valid and an invalid group, restoring the invalid one should remove warning.
func TestConsumerRestoreInvalidatedGroupForUser_RestoreInvalidatedGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "A"}
	g2 := &sdk.Group{ID: 10, Name: "B"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		Warnings: sdk.AuthConsumerWarnings{
			{
				Type:      sdk.WarningGroupInvalid,
				GroupID:   g1.ID,
				GroupName: g1.Name,
			},
		},
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g2.ID},
			InvalidGroupIDs:    []int64{g1.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Restore group 1 should remove warnings then move group 1 to valid ones
	require.NoError(t, authentication.ConsumerRestoreInvalidatedGroupForUser(context.TODO(), db, g1.ID, u.ID))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.False(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 2)
	assert.Equal(t, g2.ID, res.AuthConsumerUser.GroupIDs[0])
	assert.Equal(t, g1.ID, res.AuthConsumerUser.GroupIDs[1])
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 0)
}

// Given a disabled consumer with an invalid group, restoring the group remove warning and re-enable the consumer.
func TestConsumerLifecycle_RestoreInvalidatedLastGroup(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := &sdk.Group{ID: 5, Name: "Five"}

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		Disabled:        true,
		Warnings: sdk.AuthConsumerWarnings{
			{
				Type:      sdk.WarningGroupInvalid,
				GroupID:   g1.ID,
				GroupName: g1.Name,
			},
			{
				Type: sdk.WarningLastGroupRemoved,
			},
		},
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			InvalidGroupIDs:    []int64{g1.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Restore group 1 should remove warnings then move group 1 to valid ones
	require.NoError(t, authentication.ConsumerRestoreInvalidatedGroupForUser(context.TODO(), db, g1.ID, u.ID))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.False(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 1)
	assert.Equal(t, g1.ID, res.AuthConsumerUser.GroupIDs[0])
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 0)
}

func TestConsumerInvalidateGroupsForUser_InvalidateLastGroups(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		Warnings: sdk.AuthConsumerWarnings{
			{
				Type:      sdk.WarningGroupInvalid,
				GroupID:   g2.ID,
				GroupName: g2.Name,
			},
		},
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			GroupIDs:           []int64{g1.ID},
			InvalidGroupIDs:    []int64{g2.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	// Should invalidate g2
	require.NoError(t, authentication.ConsumerInvalidateGroupsForUser(context.TODO(), db, u.ID, []int64{}))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.True(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 0)
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 2)
	assert.Equal(t, g2.ID, res.AuthConsumerUser.InvalidGroupIDs[0])
	assert.Equal(t, g1.ID, res.AuthConsumerUser.InvalidGroupIDs[1])
	require.Len(t, res.Warnings, 3)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[0].Type)
	assert.Equal(t, g2.ID, res.Warnings[0].GroupID)
	assert.Equal(t, g2.Name, res.Warnings[0].GroupName)
	assert.Equal(t, sdk.WarningGroupInvalid, res.Warnings[1].Type)
	assert.Equal(t, g1.ID, res.Warnings[1].GroupID)
	assert.Equal(t, g1.Name, res.Warnings[1].GroupName)
	assert.Equal(t, sdk.WarningLastGroupRemoved, res.Warnings[2].Type)
}

func TestConsumerRestoreInvalidatedGroupsForUser(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	assets.DeleteConsumers(t, db)

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	require.NoError(t, user.Insert(context.TODO(), db, &u))

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)

	c := sdk.AuthConsumer{
		Name:            sdk.RandomString(10),
		Description:     sdk.RandomString(10),
		Type:            sdk.ConsumerLocal,
		ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		Disabled:        true,
		Warnings: sdk.AuthConsumerWarnings{
			{
				Type:      sdk.WarningGroupInvalid,
				GroupID:   g1.ID,
				GroupName: g1.Name,
			},
			{
				Type:      sdk.WarningGroupInvalid,
				GroupID:   g2.ID,
				GroupName: g2.Name,
			},
			{
				Type: sdk.WarningLastGroupRemoved,
			},
		},
		AuthConsumerUser: &sdk.AuthConsumerUser{
			ScopeDetails:       sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAdmin),
			InvalidGroupIDs:    []int64{g1.ID, g2.ID},
			AuthentifiedUserID: u.ID,
		},
	}
	require.NoError(t, authentication.InsertConsumer(context.TODO(), db, &c))

	require.NoError(t, authentication.ConsumerRestoreInvalidatedGroupsForUser(context.TODO(), db, u.ID))
	res, err := authentication.LoadConsumerByID(context.TODO(), db, c.ID)
	require.NoError(t, err)
	assert.False(t, res.Disabled)
	require.Len(t, res.AuthConsumerUser.GroupIDs, 2)
	assert.Equal(t, g1.ID, res.AuthConsumerUser.GroupIDs[0])
	assert.Equal(t, g2.ID, res.AuthConsumerUser.GroupIDs[1])
	require.Len(t, res.AuthConsumerUser.InvalidGroupIDs, 0)
	require.Len(t, res.Warnings, 0)
}
