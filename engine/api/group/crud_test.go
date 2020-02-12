package group_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Create_LoadByName_Delete(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u, _ := assets.InsertLambdaUser(t, db)

	groupName := sdk.RandomString(10)

	err := group.Create(context.TODO(), db, &sdk.Group{
		Name: groupName,
	}, u.ID)
	require.NoError(t, err)

	grp, err := group.LoadByName(context.TODO(), db, groupName, group.LoadOptions.WithMembers)
	require.NoError(t, err)
	assert.Len(t, grp.Members, 1)

	assert.NoError(t, group.Delete(context.TODO(), db, grp))
}
