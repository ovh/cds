package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
)

func TestDAO_AuthentifiedUserOrganization(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertLambdaUser(t, db)

	userDB, err := user.LoadByID(context.TODO(), db, u.ID, user.LoadOptions.WithOrganization)
	require.NoError(t, err)
	require.Equal(t, "default", userDB.Organization)
}
