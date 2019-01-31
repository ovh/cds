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

func Test_verifyToken(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	_, jwt, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)

	test.NoError(t, err)
	t.Logf("jwt token: %s", jwt)

	_, err = accesstoken.VerifyToken(jwt)
	test.NoError(t, err)

	_, err = accesstoken.VerifyToken("this is not a jwt token")
	assert.Error(t, err)
}

func TestIsValid(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(1 * time.Second)
	token, jwtToken, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)
	test.NoError(t, err)

	test.NoError(t, accesstoken.Insert(db, &token))
	_, isValid, err := accesstoken.IsValid(db, jwtToken)
	test.NoError(t, err)
	assert.True(t, isValid)

	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	token.Groups = append(token.Groups, *grp2)
	jwtToken2, err := accesstoken.Regen(&token)
	test.NoError(t, err)

	_, isValid, err = accesstoken.IsValid(db, jwtToken2)
	test.NoError(t, err)
	assert.False(t, isValid)

	// Wait for expiration, the token should be now expired
	time.Sleep(2 * time.Second)
	_, isValid, err = accesstoken.IsValid(db, jwtToken)
	assert.Error(t, err)
	assert.False(t, isValid)
}

func TestXSRFToken(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, "cds_test", "cds test", &exp)
	test.NoError(t, err)

	x := accesstoken.StoreXSRFToken(cache, token)
	isValid := accesstoken.CheckXSRFToken(cache, token, x)
	assert.True(t, isValid)

	isValid = accesstoken.CheckXSRFToken(cache, token, sdk.UUID())
	assert.False(t, isValid)
}
