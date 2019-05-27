package worker_test

import (
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_verifyToken(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	_, jwt, err := accesstoken.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)

	test.NoError(t, err)
	t.Logf("jwt token: %s", jwt)

	_, err = worker.VerifyToken(db, jwt)
	test.NoError(t, err)

	_, err = worker.VerifyToken(db, "this is not a jwt token")
	assert.Error(t, err)
}
