package accesstoken

import (
	"testing"
	"time"

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
