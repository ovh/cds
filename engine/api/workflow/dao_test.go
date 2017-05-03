package workflow

import (
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestLoadAllShouldNotReturnAnyWorkflows(t *testing.T) {
	db := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, nil)

	ws, err := LoadAll(db, proj.Key)
	test.NoError(t, err)
	assert.Equal(t, 0, len(ws))
}

func TestInsertSimpleWorkflow(t *testing.T) {
	db := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, nil)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip, nil))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	test.NoError(t, Insert(db, &w, nil))

	w1, err := Load(db, key, "test_1", nil)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.Root.Pipeline.ID, w1.Root.Pipeline.ID)
	assert.Equal(t, w.Root.Pipeline.Name, w1.Root.Pipeline.Name)

	t.Logf("%s", dump.MustSdump(w))
	t.Logf("%s", dump.MustSdump(w1))

	assertEqualNode(t, w.Root, w1.Root)

}

func TestInsertSimpleWorkflowWithApplicationAndEnv(t *testing.T) {
	db := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, nil)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip, nil))

	app := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app1",
	}

	test.NoError(t, application.Insert(db, proj, &app, nil))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "env1",
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				Application: &app,
				Environment: &env,
			},
		},
	}

	test.NoError(t, Insert(db, &w, nil))

	w1, err := Load(db, key, "test_1", nil)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.Root.Context.ApplicationID, w1.Root.Context.ApplicationID)
	assert.Equal(t, w.Root.Context.EnvironmentID, w1.Root.Context.EnvironmentID)

	t.Logf("%s", dump.MustSdump(w))
	t.Logf("%s", dump.MustSdump(w1))

}

func TestInsertComplexeWorkflowWith(t *testing.T) {
	db := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, nil)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip1, nil))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip2, nil))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip3, nil))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip1,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					Conditions: []sdk.WorkflowTriggerCondition{
						sdk.WorkflowTriggerCondition{
							Operator: "=",
							Value:    "master",
							Variable: ".git.branch",
						},
					},
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip2,
						Triggers: []sdk.WorkflowNodeTrigger{
							sdk.WorkflowNodeTrigger{
								Conditions: []sdk.WorkflowTriggerCondition{
									sdk.WorkflowTriggerCondition{
										Operator: "=",
										Value:    "master",
										Variable: ".git.branch",
									},
								},
								WorkflowDestNode: sdk.WorkflowNode{
									Pipeline: pip3,
								},
							},
						},
					},
				},
				sdk.WorkflowNodeTrigger{
					Conditions: []sdk.WorkflowTriggerCondition{
						sdk.WorkflowTriggerCondition{
							Operator: "=",
							Value:    "master",
							Variable: ".git.branch",
						},
					},
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip2,
					},
				},
			},
		},
	}

	test.NoError(t, Insert(db, &w, nil))

	w1, err := Load(db, key, "test_1", nil)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.Root.Pipeline.ID, w1.Root.Pipeline.ID)
	assert.Equal(t, w.Root.Pipeline.Name, w1.Root.Pipeline.Name)
	test.Equal(t, len(w.Root.Triggers), len(w1.Root.Triggers))

	assertEqualNode(t, w.Root, w1.Root)

	t.Logf("%s", dump.MustSdump(w))
	t.Logf("%s", dump.MustSdump(w1))
}

func assertEqualNode(t *testing.T, n1 *sdk.WorkflowNode, n2 *sdk.WorkflowNode) {
	SortNode(n1)
	SortNode(n2)

	test.Equal(t, len(n1.Hooks), len(n2.Hooks))
	test.Equal(t, len(n1.Triggers), len(n2.Triggers))

	assert.Equal(t, n1.Pipeline.Name, n2.Pipeline.Name)
	assert.Equal(t, n1.Pipeline.ProjectKey, n2.Pipeline.ProjectKey)

	for i, t1 := range n1.Triggers {
		t2 := n2.Triggers[i]
		test.EqualValuesWithoutOrder(t, t1.Conditions, t2.Conditions)
		assertEqualNode(t, &t1.WorkflowDestNode, &t2.WorkflowDestNode)
	}

}

func TestUpdateSimpleWorkflowWithApplicationAndEnv(t *testing.T) {
	db := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, nil)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip, nil))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip2, nil))

	app := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app1",
	}

	test.NoError(t, application.Insert(db, proj, &app, nil))

	app2 := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app2",
	}

	test.NoError(t, application.Insert(db, proj, &app2, nil))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "env1",
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				Application: &app,
				Environment: &env,
			},
		},
	}

	test.NoError(t, Insert(db, &w, nil))

	w1, err := Load(db, key, "test_1", nil)
	test.NoError(t, err)

	t.Logf("%s", dump.MustSdump(w1))

	w1.Name = "test_2"
	w1.Root.PipelineID = pip2.ID
	w1.Root.Context.ApplicationID = app2.ID

	Update(db, w1, nil)

	w2, err := LoadByID(db, w1.ID, nil)
	test.NoError(t, err)

	assert.Equal(t, w1.ID, w2.ID)
	assert.Equal(t, w1.Root.Context.ApplicationID, w2.Root.Context.Application.ID)
	assert.Equal(t, w1.Root.Context.EnvironmentID, w2.Root.Context.Environment.ID)

	t.Logf("%s", dump.MustSdump(w2))
}
