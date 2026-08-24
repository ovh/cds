package group_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestDAO_LinkGroupUser(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u1, _ := assets.InsertLambdaUser(t, db)
	u2, _ := assets.InsertLambdaUser(t, db)

	groupName := sdk.RandomString(10)

	require.NoError(t, group.Create(context.TODO(), db, &sdk.Group{
		Name: groupName,
	}, u1))

	grp, err := group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 1)

	link := &group.LinkGroupUser{
		GroupID:            grp.ID,
		AuthentifiedUserID: u2.ID,
	}
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, link))

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 2)

	var m1, m2 *sdk.GroupMember
	for i := range grp.Members {
		if grp.Members[i].ID == u1.ID {
			m1 = &grp.Members[i]
		}
		if grp.Members[i].ID == u2.ID {
			m2 = &grp.Members[i]
		}
	}
	require.NotNil(t, m1)
	require.True(t, m1.Admin)
	require.NotNil(t, m2)
	require.False(t, m2.Admin)

	link.Admin = true
	require.NoError(t, group.UpdateLinkGroupUser(context.TODO(), db, link))

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 2)

	m1, m2 = nil, nil
	for i := range grp.Members {
		if grp.Members[i].ID == u1.ID {
			m1 = &grp.Members[i]
		}
		if grp.Members[i].ID == u2.ID {
			m2 = &grp.Members[i]
		}
	}
	require.NotNil(t, m1)
	require.True(t, m1.Admin)
	require.NotNil(t, m2)
	require.True(t, m2.Admin)

	links, err := group.LoadLinksGroupUserForUserIDs(context.TODO(), db, []string{u1.ID, u2.ID})
	require.NoError(t, err)
	assert.Len(t, links, 2)
}

// TestDAO_LinkGroupUserDisabled checks that a disabled user still shows up in its groups,
// flagged as such: memberships are kept on purpose, so the flag is the only way to tell
// an active member from a neutralized one.
func TestDAO_LinkGroupUserDisabled(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u1, _ := assets.InsertLambdaUser(t, db)
	u2, _ := assets.InsertLambdaUser(t, db)

	groupName := sdk.RandomString(10)
	require.NoError(t, group.Create(context.TODO(), db, &sdk.Group{Name: groupName}, u1))

	grp, err := group.LoadByName(context.TODO(), db, groupName)
	require.NoError(t, err)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            grp.ID,
		AuthentifiedUserID: u2.ID,
	}))

	u2.Disabled = true
	require.NoError(t, user.Update(context.TODO(), db, u2))

	grp, err = group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	require.Len(t, grp.Members, 2, "a disabled user must remain a member of its groups")

	for i := range grp.Members {
		switch grp.Members[i].ID {
		case u1.ID:
			assert.False(t, grp.Members[i].Disabled)
		case u2.ID:
			assert.True(t, grp.Members[i].Disabled, "a disabled member must be flagged")
		}
	}
}

// TestDAO_LoadGroupIDsWithoutActiveMember checks that a group nobody can act through is
// reported: with every member disabled, or with no member at all. Both cases make the
// group, and the projects it grants access to, unreachable.
func TestDAO_LoadGroupIDsWithoutActiveMember(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	active, _ := assets.InsertLambdaUser(t, db)
	disabled, _ := assets.InsertLambdaUser(t, db)
	disabled.Disabled = true
	require.NoError(t, user.Update(context.TODO(), db, disabled))

	// A group with one active member and one disabled member
	withActive := &sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Create(context.TODO(), db, withActive, active))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            withActive.ID,
		AuthentifiedUserID: disabled.ID,
	}))

	// A group whose only member is disabled
	onlyDisabled := &sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Create(context.TODO(), db, onlyDisabled, disabled))

	// A group without any member
	empty := &sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Insert(context.TODO(), db, empty))

	ids, err := group.LoadGroupIDsWithoutActiveMember(context.TODO(), db,
		[]int64{withActive.ID, onlyDisabled.ID, empty.ID})
	require.NoError(t, err)

	assert.NotContains(t, ids, withActive.ID, "a group keeping one active member is still usable")
	assert.Contains(t, ids, onlyDisabled.ID, "a group whose members are all disabled must be reported")
	assert.Contains(t, ids, empty.ID, "a group without member must be reported")

	// Same result through the load option used by the group listing
	grps, err := group.LoadAllByIDs(context.TODO(), db,
		[]int64{withActive.ID, onlyDisabled.ID, empty.ID}, group.LoadOptions.WithNoActiveMember)
	require.NoError(t, err)
	require.Len(t, grps, 3)
	for _, g := range grps {
		switch g.ID {
		case withActive.ID:
			assert.False(t, g.NoActiveMember)
		default:
			assert.True(t, g.NoActiveMember, "group %q should have no active member", g.Name)
		}
	}
}
