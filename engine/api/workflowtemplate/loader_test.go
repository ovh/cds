package workflowtemplate_test 

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
)

func TestLoadGroup(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	defer func() {
		assets.DeleteTestGroup(t, db, grp1)
		assets.DeleteTestGroup(t, db, grp2)
	}()

	tmpl := sdk.WorkflowTemplate{
		GroupID: grp1.ID,
		Slug:    "tmpl-2",
		Name:    "Template 2",
	}

	require.NoError(t, workflowtemplate.Insert(db, &tmpl))
	assert.Nil(t, workflowtemplate.LoadOptions.WithGroup(context.TODO(), db, &tmpl))

	assert.NotNil(t, tmpl.Group)
	assert.Equal(t, grp1.Name, tmpl.Group.Name)

	assert.NoError(t, workflowtemplate.Delete(db, &tmpl))
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

	assert.Nil(t, workflowtemplate.LoadInstanceOptions.WithTemplate(context.TODO(), db, wtis...))

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
