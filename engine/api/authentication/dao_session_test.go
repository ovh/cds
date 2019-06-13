package authentication_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	token, _, err := authentication.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, authentication.Insert(db, &token))

	reloadedToken, err := authentication.LoadByID(context.TODO(), db, token.ID, authentication.LoadOptions.WithGroups)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 1)

	test.NoError(t, authentication.Delete(db, token.ID))
}

func TestUpdate(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := authentication.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, authentication.Insert(db, &token))

	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	token.Groups = append(token.Groups, *grp2)
	_, err = authentication.Regen(&token)
	test.NoError(t, err)

	test.NoError(t, authentication.Update(db, &token))

	reloadedToken, err := authentication.LoadByID(context.TODO(), db, token.ID, authentication.LoadOptions.WithGroups)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 2)
	test.NoError(t, authentication.Delete(db, reloadedToken.ID))
}

func TestFind(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	usr1, _ := assets.InsertLambdaUser(db, grp1) // This creates an access token

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := authentication.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, authentication.Insert(db, &token))

	// TestFindByID
	reloadedToken, err := authentication.LoadByID(context.TODO(), db, token.ID, authentication.LoadOptions.WithAuthentifiedUser, authentication.LoadOptions.WithGroups)
	test.NoError(t, err)
	assert.Len(t, reloadedToken.Groups, 1)
	test.Equal(t, usr1.ID, reloadedToken.AuthentifiedUser.ID)
	assert.Len(t, reloadedToken.AuthentifiedUser.GetGroups(), 1)

	// LoadAllByUserID
	tokens, err := authentication.LoadAllByUserID(context.TODO(), db, usr1.ID,
		authentication.LoadOptions.WithAuthentifiedUser,
		authentication.LoadOptions.WithGroups,
	)
	test.NoError(t, err)
	assert.Len(t, tokens, 2)

	usr2, _ := assets.InsertLambdaUser(db)
	tokens, err = authentication.LoadAllByUserID(context.TODO(), db, usr2.ID,
		authentication.LoadOptions.WithAuthentifiedUser,
		authentication.LoadOptions.WithGroups,
	)
	test.NoError(t, err)
	assert.Len(t, tokens, 1)

	// LoadAllByGroupID
	tokens, err = authentication.LoadAllByGroupID(context.TODO(), db, grp1.ID,
		authentication.LoadOptions.WithAuthentifiedUser,
		authentication.LoadOptions.WithGroups,
	)
	test.NoError(t, err)
	assert.Len(t, tokens, 2)

	test.NoError(t, authentication.Delete(db, token.ID))
}
