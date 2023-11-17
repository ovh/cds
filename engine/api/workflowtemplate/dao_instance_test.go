package workflowtemplate_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCRUD_Instance(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	grp := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	tmpl := sdk.WorkflowTemplate{
		GroupID: grp.ID,
		Slug:    "tmpl",
		Name:    "Template",
		Version: 1,
	}
	require.NoError(t, workflowtemplate.Insert(db, &tmpl), "No err should be returned when adding a template instance")

	wti := sdk.WorkflowTemplateInstance{
		ProjectID:               proj.ID,
		WorkflowTemplateID:      tmpl.ID,
		WorkflowTemplateVersion: tmpl.Version,
		Request: sdk.WorkflowTemplateRequest{
			WorkflowName: "my-workflow",
			ProjectKey:   proj.Key,
		},
	}
	require.NoError(t, workflowtemplate.InsertInstance(db, &wti))

	wti.WorkflowTemplateVersion = 2
	require.NoError(t, workflowtemplate.UpdateInstance(db, &wti))

	assert.NoError(t, workflowtemplate.DeleteInstance(db, &wti), "No err should be returned when removing a template instance")
}

func TestLoad_Instance(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	proj1 := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	proj2 := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	grp := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	wk1 := assets.InsertTestWorkflow(t, db, cache, proj2, sdk.RandomString(10))
	wk2 := assets.InsertTestWorkflow(t, db, cache, proj2, sdk.RandomString(10))

	tmpl := sdk.WorkflowTemplate{
		GroupID: grp.ID,
		Slug:    "tmpl",
		Name:    "Template",
		Version: 1,
	}
	require.NoError(t, workflowtemplate.Insert(db, &tmpl), "No err should be returned when adding a template instance")

	wti1 := sdk.WorkflowTemplateInstance{
		ProjectID:               proj1.ID,
		WorkflowTemplateID:      tmpl.ID,
		WorkflowTemplateVersion: tmpl.Version,
		Request: sdk.WorkflowTemplateRequest{
			WorkflowName: "my-workflow",
			ProjectKey:   proj1.Key,
		},
	}
	require.NoError(t, workflowtemplate.InsertInstance(db, &wti1))

	wti2 := sdk.WorkflowTemplateInstance{
		WorkflowID:              &wk1.ID,
		ProjectID:               proj2.ID,
		WorkflowTemplateID:      tmpl.ID,
		WorkflowTemplateVersion: tmpl.Version,
		Request: sdk.WorkflowTemplateRequest{
			WorkflowName: "my-workflow-1",
			ProjectKey:   proj2.Key,
		},
	}
	require.NoError(t, workflowtemplate.InsertInstance(db, &wti2))

	wti3 := sdk.WorkflowTemplateInstance{
		WorkflowID:              &wk2.ID,
		ProjectID:               proj2.ID,
		WorkflowTemplateID:      tmpl.ID,
		WorkflowTemplateVersion: tmpl.Version,
		Request: sdk.WorkflowTemplateRequest{
			WorkflowName: "my-workflow-2",
			ProjectKey:   proj2.Key,
		},
	}
	require.NoError(t, workflowtemplate.InsertInstance(db, &wti3))

	is, err := workflowtemplate.LoadInstancesByTemplateIDAndProjectIDs(context.TODO(), db, tmpl.ID, []int64{proj1.ID})
	require.NoError(t, err)
	assert.Len(t, is, 1)

	is, err = workflowtemplate.LoadInstancesByTemplateIDAndProjectIDs(context.TODO(), db, tmpl.ID, []int64{proj1.ID, proj2.ID})
	require.NoError(t, err)
	assert.Len(t, is, 3)

	is, err = workflowtemplate.LoadInstancesByWorkflowIDs(context.TODO(), db, []int64{wk1.ID})
	require.NoError(t, err)
	assert.Len(t, is, 1)

	is, err = workflowtemplate.LoadInstancesByWorkflowIDs(context.TODO(), db, []int64{wk1.ID, wk2.ID})
	require.NoError(t, err)
	assert.Len(t, is, 2)

	i, err := workflowtemplate.LoadInstanceByWorkflowID(context.TODO(), db, wk1.ID)
	require.NoError(t, err)
	assert.Equal(t, wti2.ID, i.ID)

	i, err = workflowtemplate.LoadInstanceByIDForTemplateIDAndProjectIDs(context.TODO(), db, wti2.ID, tmpl.ID, []int64{proj1.ID})
	require.Error(t, err)
	assert.NotNil(t, i)

	i, err = workflowtemplate.LoadInstanceByIDForTemplateIDAndProjectIDs(context.TODO(), db, wti2.ID, tmpl.ID, []int64{proj2.ID})
	require.NoError(t, err)
	assert.Equal(t, wti2.ID, i.ID)

	i, err = workflowtemplate.LoadInstanceByTemplateIDAndProjectIDAndRequestWorkflowName(context.TODO(), db, tmpl.ID, proj2.ID, "my-unknown-workflow")
	assert.Error(t, err)

	i, err = workflowtemplate.LoadInstanceByTemplateIDAndProjectIDAndRequestWorkflowName(context.TODO(), db, tmpl.ID, proj2.ID, "my-workflow-1")
	require.NoError(t, err)
	assert.Equal(t, i.ID, wti2.ID)
}
