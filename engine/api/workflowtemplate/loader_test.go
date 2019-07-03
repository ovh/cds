package workflowtemplate

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestLoadGroup(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		gs := i.(*[]*sdk.Group)
		*gs = append(*gs, &sdk.Group{
			ID:   1,
			Name: "grp-1",
		}, &sdk.Group{
			ID:   2,
			Name: "grp-2",
		})
	}

	wts := []*sdk.WorkflowTemplate{
		{GroupID: 1},
		{GroupID: 2},
	}

	assert.Nil(t, loadGroup(context.TODO(), db, wts...))

	assert.NotNil(t, wts[0].Group)
	assert.Equal(t, "grp-1", wts[0].Group.Name)
	assert.NotNil(t, wts[1].Group)
	assert.Equal(t, "grp-2", wts[1].Group.Name)
}

func TestLoadInstanceTemplate(t *testing.T) {
	db := &test.SqlExecutorMock{}

	db.OnSelect = func(i interface{}) {
		if wts, ok := i.(*[]*sdk.WorkflowTemplate); ok {
			*wts = append(*wts,
				&sdk.WorkflowTemplate{
					ID:    1,
					Slug:  "one",
					Group: &sdk.Group{Name: "one"},
				},
				&sdk.WorkflowTemplate{
					ID:    2,
					Slug:  "two",
					Group: &sdk.Group{Name: "two"},
				})
		}
	}

	wtis := []*sdk.WorkflowTemplateInstance{
		{WorkflowTemplateID: 1},
		{WorkflowTemplateID: 1},
		{WorkflowTemplateID: 2},
	}

	assert.Nil(t, loadInstanceTemplate(context.TODO(), db, wtis...))

	if !assert.NotNil(t, wtis[0].Template) {
		t.FailNow()
	}
	if !assert.NotNil(t, wtis[0].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "one/one", fmt.Sprintf("%s/%s", wtis[0].Template.Group.Name, wtis[0].Template.Slug))

	if !assert.NotNil(t, wtis[1].Template) {
		t.FailNow()
	}
	if !assert.NotNil(t, wtis[1].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "one/one", fmt.Sprintf("%s/%s", wtis[1].Template.Group.Name, wtis[1].Template.Slug))

	if !assert.NotNil(t, wtis[2].Template) {
		t.FailNow()
	}
	if !assert.NotNil(t, wtis[2].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "two/two", fmt.Sprintf("%s/%s", wtis[2].Template.Group.Name, wtis[2].Template.Slug))
}
