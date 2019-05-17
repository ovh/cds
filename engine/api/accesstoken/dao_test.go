package accesstoken_test

import (
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

	reloadedToken, err := accesstoken.FindByID(db, token.ID)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 1)

	test.NoError(t, accesstoken.Delete(db, &token))

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

	reloadedToken, err := accesstoken.FindByID(db, token.ID)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 2)
	test.NoError(t, accesstoken.Delete(db, &reloadedToken))
}

func TestFind(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, accesstoken.Insert(db, &token))

	// TestFindByID
	reloadedToken, err := accesstoken.FindByID(db, token.ID)
	test.NoError(t, err)
	assert.Len(t, reloadedToken.Groups, 1)

	// FindAllByUser
	tokens, err := accesstoken.FindAllByUser(db, usr1.ID)
	test.NoError(t, err)
	assert.Len(t, tokens, 1)

	usr2, _ := assets.InsertLambdaUser(db)
	tokens, err = accesstoken.FindAllByUser(db, usr2.ID)
	test.NoError(t, err)
	assert.Len(t, tokens, 0)

	// FindAllByGroup
	tokens, err = accesstoken.FindAllByGroup(db, grp1.ID)
	test.NoError(t, err)
	assert.Len(t, tokens, 1)

	test.NoError(t, accesstoken.Delete(db, &token))

}
