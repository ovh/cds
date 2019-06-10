package action

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_loadGroup(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		gs := i.(*[]sdk.Group)
		*gs = append(*gs, sdk.Group{
			ID:   1,
			Name: "grp-1",
		}, sdk.Group{
			ID:   2,
			Name: "grp-2",
		})
	}

	id1, id2 := int64(1), int64(2)
	as := []*sdk.Action{
		{GroupID: &id1},
		{GroupID: &id2},
	}

	assert.Nil(t, loadGroup(context.TODO(), db, as...))

	assert.NotNil(t, as[0].Group)
	assert.Equal(t, "grp-1", as[0].Group.Name)
	assert.NotNil(t, as[1].Group)
	assert.Equal(t, "grp-2", as[1].Group.Name)
}
