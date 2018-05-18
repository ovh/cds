package warning

import (
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManageProjectDeletePermission(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, p, u))

	pip := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		Type:      "build",
		ProjectID: p.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, store, p, pip, u))

	e := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
	}
	assert.NoError(t, environment.InsertEnvironment(db, e))

	w := &sdk.Workflow{
		ProjectID:  p.ID,
		ProjectKey: p.Key,
		Name:       sdk.RandomString(10),
		Root: &sdk.WorkflowNode{
			Name:     sdk.RandomString(10),
			Pipeline: *pip,
		},
	}
	projUpdated, err := project.Load(db, store, p.Key, u, project.LoadOptions.WithPipelines)
	assert.NoError(t, err)
	assert.NoError(t, workflow.Insert(db, store, w, projUpdated, u))

	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	assert.NoError(t, group.InsertGroup(db, g))

	gp := sdk.GroupPermission{
		Permission: 7,
		Group:      *g,
	}

	assert.NoError(t, workflow.AddGroup(db, w, gp))
	assert.NoError(t, group.InsertGroupInEnvironment(db, e.ID, g.ID, 7))

	assert.NoError(t, manageProjectDeletePermission(db, p.Key, gp))

	warnings, err := GetByProject(db, p.Key)
	assert.NoError(t, err)

	nbGoodWarnings := 0
	for _, w := range warnings {
		if w.Type == MissingProjectPermissionWorkflow || w.Type == MissingProjectPermissionEnv {
			nbGoodWarnings++
		}
	}
	assert.Equal(t, 2, nbGoodWarnings)
}
