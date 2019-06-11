package accesstoken_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestInsert(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, accesstoken.Insert(db, &token))

	reloadedToken, err := accesstoken.LoadByID(context.TODO(), db, token.ID, accesstoken.LoadOptions.WithGroups)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 1)

	test.NoError(t, accesstoken.Delete(db, token.ID))
}

func TestUpdate(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, accesstoken.Insert(db, &token))

	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	token.Groups = append(token.Groups, *grp2)
	_, err = accesstoken.Regen(&token)
	test.NoError(t, err)

	test.NoError(t, accesstoken.Update(db, &token))

	reloadedToken, err := accesstoken.LoadByID(context.TODO(), db, token.ID, accesstoken.LoadOptions.WithGroups)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 2)
	test.NoError(t, accesstoken.Delete(db, reloadedToken.ID))
}

func TestFind(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	usr1, _ := assets.InsertLambdaUser(db, grp1) // This creates an access token

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, accesstoken.Insert(db, &token))

	// TestFindByID
	reloadedToken, err := accesstoken.LoadByID(context.TODO(), db, token.ID, accesstoken.LoadOptions.WithAuthentifiedUser, accesstoken.LoadOptions.WithGroups)
	test.NoError(t, err)
	assert.Len(t, reloadedToken.Groups, 1)
	test.Equal(t, usr1.ID, reloadedToken.AuthentifiedUser.ID)
	assert.Len(t, reloadedToken.AuthentifiedUser.GetGroups(), 1)

	// LoadAllByUserID
	tokens, err := accesstoken.LoadAllByUserID(context.TODO(), db, usr1.ID,
		accesstoken.LoadOptions.WithAuthentifiedUser,
		accesstoken.LoadOptions.WithGroups,
	)
	test.NoError(t, err)
	assert.Len(t, tokens, 2)

	usr2, _ := assets.InsertLambdaUser(db)
	tokens, err = accesstoken.LoadAllByUserID(context.TODO(), db, usr2.ID,
		accesstoken.LoadOptions.WithAuthentifiedUser,
		accesstoken.LoadOptions.WithGroups,
	)
	test.NoError(t, err)
	assert.Len(t, tokens, 1)

	// LoadAllByGroupID
	tokens, err = accesstoken.LoadAllByGroupID(context.TODO(), db, grp1.ID,
		accesstoken.LoadOptions.WithAuthentifiedUser,
		accesstoken.LoadOptions.WithGroups,
	)
	test.NoError(t, err)
	assert.Len(t, tokens, 2)

	test.NoError(t, accesstoken.Delete(db, token.ID))
}
