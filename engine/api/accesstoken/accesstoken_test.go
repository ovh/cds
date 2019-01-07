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

func Test_verifyToken(t *testing.T) {
	test.NoError(t, Init("cds_test", TestKey))
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	_, jwt, err := New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)

	test.NoError(t, err)
	t.Logf("jwt token: %s", jwt)

	_, err = verifyToken(jwt)
	test.NoError(t, err)

}

func TestIsValid(t *testing.T) {
	Init("cds_test", TestKey)
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, jwtToken, err := New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)
	test.NoError(t, err)

	test.NoError(t, Insert(db, &token))
	isValid, err := IsValid(db, jwtToken)
	test.NoError(t, err)
	assert.True(t, isValid)

	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	token.Groups = append(token.Groups, *grp2)
	jwtToken2, err := Regen(&token)
	test.NoError(t, err)

	isValid, err = IsValid(db, jwtToken2)
	test.NoError(t, err)
	assert.False(t, isValid)
}
