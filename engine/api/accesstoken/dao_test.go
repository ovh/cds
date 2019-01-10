package accesstoken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestInserts(t *testing.T) {
	Init("cds_test", test.TestKey)
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)
	test.NoError(t, err)

	test.NoError(t, Insert(db, &token))

	reloadedToken, err := FindByID(db, token.ID)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 1)

	test.NoError(t, Delete(db, &token))

}

func TestUpdate(t *testing.T) {
	Init("cds_test", test.TestKey)
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)
	test.NoError(t, err)

	test.NoError(t, Insert(db, &token))

	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	token.Groups = append(token.Groups, *grp2)
	_, err = Regen(&token)
	test.NoError(t, err)

	test.NoError(t, Update(db, &token))

	reloadedToken, err := FindByID(db, token.ID)
	test.NoError(t, err)

	t.Logf("reloaded token is s%+v", reloadedToken)

	assert.Len(t, reloadedToken.Groups, 2)
	test.NoError(t, Delete(db, &reloadedToken))
}
