package authentication_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestWithGroups(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	u := sdk.AuthentifiedUser{
		Username: sdk.RandomString(10),
	}
	test.NoError(t, user.Insert(db, &u))

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)

	c, err := builtin.NewConsumer(db, sdk.RandomString(10), "", u.ID, []int64{g1.ID, g2.ID}, nil)
	test.NoError(t, err)

	s, err := authentication.NewSession(db, c, time.Second)
	test.NoError(t, err)

	res, err := authentication.LoadSessionByID(context.TODO(), db, s.ID,
		authentication.LoadSessionOptions.WithGroups)
	test.NoError(t, err)
	test.NotNil(t, res)
	test.Equal(t, 2, len(res.Groups))
	sort.Slice(res.Groups, func(i, j int) bool { return res.Groups[i].ID < res.Groups[j].ID })
	test.Equal(t, g1.Name, res.Groups[0].Name)
	test.Equal(t, g2.Name, res.Groups[1].Name)
}
