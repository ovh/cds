package rbac_test

import (
	"context"
	"fmt"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"

	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImportRbacHatchery(t *testing.T) {
	db, _ := test.SetupPG(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM hatchery")
	require.NoError(t, err)

	hatch := sdk.Hatchery{Name: sdk.RandomString(10)}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	rbacYaml := `name: perm-%s
hatcheries:
- role: %s
  region: %s
  hatchery: %s
`

	rbacYaml = fmt.Sprintf(rbacYaml, reg.Name, sdk.HatcheryRoleSpawn, reg.Name, hatch.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))
	r.Hatcheries[0].RegionID = reg.ID
	r.Hatcheries[0].HatcheryID = hatch.ID

	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	rbacRB, err := rbac.LoadRBACByName(context.TODO(), db, r.Name, rbac.LoadOptions.LoadRBACHatchery)
	require.NoError(t, err)
	require.Equal(t, 1, len(rbacRB.Hatcheries))
	require.Equal(t, reg.ID, rbacRB.Hatcheries[0].RegionID)
	require.Equal(t, hatch.ID, rbacRB.Hatcheries[0].HatcheryID)
}
