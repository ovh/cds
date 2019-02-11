package migrate

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_mergePermissions(t *testing.T) {
	gps1 := []sdk.GroupPermission{
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 1,
			},
			Permission: 5,
		},
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 2,
			},
			Permission: 3,
		},
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 55,
			},
			Permission: 3,
		},
	}
	gps2 := []sdk.GroupPermission{
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 1,
			},
			Permission: 3,
		},
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 2,
			},
			Permission: 7,
		},
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 66,
			},
			Permission: 7,
		},
	}

	resp := mergePermissions(gps1, gps2)
	t.Logf("%+v\n", resp)
	test.Equal(t, 4, len(resp))
	for _, gp := range resp {
		switch gp.Group.ID {
		case 1:
			test.Equal(t, 5, gp.Permission)
		case 2:
			test.Equal(t, 7, gp.Permission)
		case 55:
			test.Equal(t, 3, gp.Permission)
		case 66:
			test.Equal(t, 7, gp.Permission)
		}
	}
}

func Test_diffPermissions(t *testing.T) {
	gps1 := []sdk.GroupPermission{
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 1,
			},
			Permission: 5,
		},
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 2,
			},
			Permission: 3,
		},
	}
	gps2 := []sdk.GroupPermission{
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 1,
			},
			Permission: 3,
		},
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 2,
			},
			Permission: 7,
		},
		sdk.GroupPermission{
			Group: sdk.Group{
				ID: 66,
			},
			Permission: 7,
		},
	}

	resp := diffPermission(gps1, gps2)
	t.Logf("%+v\n", resp)
	test.Equal(t, 1, len(resp))
	test.Equal(t, int64(66), resp[0].Group.ID)
}
