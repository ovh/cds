package workflowtemplate_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
)

func TestCRUD(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	defer func() {
		assert.Nil(t, assets.DeleteTestGroup(t, db, grp1))
		assert.Nil(t, assets.DeleteTestGroup(t, db, grp2))
	}()

	tmpls := []sdk.WorkflowTemplate{
		{
			GroupID:     grp1.ID,
			Slug:        "tmpl-1",
			Name:        "Template 1",
			Description: "My template 1 description",
			Parameters: []sdk.WorkflowTemplateParameter{
				{Key: "my-bool", Type: sdk.ParameterTypeBoolean, Required: true},
				{Key: "my-string", Type: sdk.ParameterTypeString, Required: true},
				{Key: "my-repository", Type: sdk.ParameterTypeRepository, Required: true},
			},
			Workflow:     "the-yml-workflow-encoded",
			Pipelines:    []sdk.PipelineTemplate{{Value: "the-yml-pipeline-encoded"}},
			Applications: []sdk.ApplicationTemplate{{Value: "the-yml-application-encoded"}},
			Environments: []sdk.EnvironmentTemplate{{Value: "the-yml-environment-encoded"}},
			Version:      10,
		},
		{
			GroupID: grp1.ID,
			Slug:    "tmpl-2",
			Name:    "Template 2",
		},
		{
			GroupID: grp2.ID,
			Slug:    "tmpl-3",
			Name:    "Template 3",
		},
	}

	// Insert
	for i := range tmpls {
		if !assert.Nil(t, workflowtemplate.Insert(db, &tmpls[i]), "No err should be returned when adding a template") {
			t.FailNow()
		}
	}

	// Update
	tmpls[0].Version++
	assert.Nil(t, workflowtemplate.Update(db, &tmpls[0]), "No err should be returned when updating a template")
	assert.Equal(t, int64(11), tmpls[0].Version)

	// LoadByID
	result, err := workflowtemplate.LoadByID(context.TODO(), db, 0)
	assert.Nil(t, err)
	assert.Nil(t, result)
	result, err = workflowtemplate.LoadByID(context.TODO(), db, tmpls[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, result.Name, tmpls[0].Name)

	// LoadBySlugAndGroupID
	result, err = workflowtemplate.LoadBySlugAndGroupID(context.TODO(), db, tmpls[0].Slug, grp1.ID)
	assert.Nil(t, err)
	assert.Equal(t, result.Name, tmpls[0].Name)

	// LoadAllByGroupIDs
	results, err := workflowtemplate.LoadAllByGroupIDs(context.TODO(), db, nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(results))
	results, err = workflowtemplate.LoadAllByGroupIDs(context.TODO(), db, []int64{grp1.ID, grp2.ID})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(results))

	// LoadAllByIDs
	results, err = workflowtemplate.LoadAllByIDs(context.TODO(), db, nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(results))
	results, err = workflowtemplate.LoadAllByIDs(context.TODO(), db, []int64{tmpls[0].ID, tmpls[1].ID})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(results))

	// Delete
	for i := range tmpls {
		assert.Nil(t, workflowtemplate.Delete(db, &tmpls[i]), "No err should be returned when removing a template")
	}
}
