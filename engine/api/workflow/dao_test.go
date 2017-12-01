package workflow_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestLoadAllShouldNotReturnAnyWorkflows(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	ws, err := workflow.LoadAll(db, proj.Key)
	test.NoError(t, err)
	assert.Equal(t, 0, len(ws))
}

func TestInsertSimpleWorkflow(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.Root.Pipeline.ID, w1.Root.Pipeline.ID)
	assert.Equal(t, w.Root.Pipeline.Name, w1.Root.Pipeline.Name)
	assertEqualNode(t, w.Root, w1.Root)

	ws, err := workflow.LoadAll(db, proj.Key)
	test.NoError(t, err)
	assert.Equal(t, 1, len(ws))

}

func TestInsertSimpleWorkflowWithWrongName(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_ 1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	assert.Error(t, workflow.Insert(db, cache, &w, proj, u))
}

func TestInsertSimpleWorkflowWithApplicationAndEnv(t *testing.T) {
	db, cache := test.SetupPG(t)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

	app := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app1",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, u))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "env1",
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.Root.Context.ApplicationID, w1.Root.Context.ApplicationID)
	assert.Equal(t, w.Root.Context.EnvironmentID, w1.Root.Context.EnvironmentID)
}

func TestInsertComplexeWorkflow(t *testing.T) {
	db, cache := test.SetupPG(t)

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip1, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip3, u))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip4, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip1,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip2,
						Context: &sdk.WorkflowNodeContext{
							Conditions: sdk.WorkflowNodeConditions{
								PlainConditions: []sdk.WorkflowNodeCondition{
									sdk.WorkflowNodeCondition{
										Operator: "=",
										Value:    "master",
										Variable: ".git.branch",
									},
								},
							},
						},
						Triggers: []sdk.WorkflowNodeTrigger{
							sdk.WorkflowNodeTrigger{
								WorkflowDestNode: sdk.WorkflowNode{
									Pipeline: pip3,
									Context: &sdk.WorkflowNodeContext{
										Conditions: sdk.WorkflowNodeConditions{
											PlainConditions: []sdk.WorkflowNodeCondition{
												sdk.WorkflowNodeCondition{
													Operator: "=",
													Value:    "master",
													Variable: ".git.branch",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip4,
						Context: &sdk.WorkflowNodeContext{
							Conditions: sdk.WorkflowNodeConditions{
								PlainConditions: []sdk.WorkflowNodeCondition{
									sdk.WorkflowNodeCondition{
										Operator: "=",
										Value:    "master",
										Variable: ".git.branch",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.Root.Pipeline.ID, w1.Root.Pipeline.ID)
	assert.Equal(t, w.Root.Pipeline.Name, w1.Root.Pipeline.Name)
	test.Equal(t, len(w.Root.Triggers), len(w1.Root.Triggers))

	workflow.Sort(&w)

	assertEqualNode(t, w.Root, w1.Root)
}

func assertEqualNode(t *testing.T, n1, n2 *sdk.WorkflowNode) {
	t.Logf("assertEqualNode : %d(%s) on %s", n2.ID, n2.Ref, n2.Pipeline.Name)
	workflow.SortNode(n1)
	workflow.SortNode(n2)
	t.Logf("assertEqualNode : Checking hooks")
	test.Equal(t, len(n1.Hooks), len(n2.Hooks))
	t.Logf("assertEqualNode : Checking triggers")
	test.Equal(t, len(n1.Triggers), len(n2.Triggers))

	assert.Equal(t, n1.Pipeline.Name, n2.Pipeline.Name)
	assert.Equal(t, n1.Pipeline.ProjectKey, n2.Pipeline.ProjectKey)
	for i, t1 := range n1.Triggers {
		t2 := n2.Triggers[i]
		test.Equal(t, len(t1.WorkflowDestNode.Context.Conditions.PlainConditions), len(t2.WorkflowDestNode.Context.Conditions.PlainConditions), "Number of conditions on node does not match")
		test.EqualValuesWithoutOrder(t, t1.WorkflowDestNode.Context.Conditions.PlainConditions, t2.WorkflowDestNode.Context.Conditions.PlainConditions, "Conditions on triggers does not match")
		assertEqualNode(t, &t1.WorkflowDestNode, &t2.WorkflowDestNode)
	}
}
func TestUpdateSimpleWorkflowWithApplicationEnvPipelineParametersAndPayload(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
		Parameter: []sdk.Parameter{
			{
				Name:  "param1",
				Type:  sdk.StringParameter,
				Value: "value1",
			},
		},
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
		Parameter: []sdk.Parameter{
			{
				Name:  "param1",
				Type:  sdk.StringParameter,
				Value: "value1",
			},
		},
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip3, u))

	app := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app1",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, u))

	app2 := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app2",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app2, u))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "env1",
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				Application: &app,
				Environment: &env,
				DefaultPipelineParameters: []sdk.Parameter{
					{
						Name:  "param1",
						Type:  sdk.StringParameter,
						Value: "param1_value",
					},
				},
				DefaultPayload: []sdk.Parameter{
					{
						Name:  "git.branch",
						Type:  sdk.StringParameter,
						Value: "master",
					},
				},
			},
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						PipelineID: pip3.ID,
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	w1old, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	t.Logf("Modifying workflow... with %d instead of %d", app2.ID, app.ID)

	w1.Name = "test_2"
	w1.Root.PipelineID = pip2.ID
	w1.Root.Context.Application = &app2
	w1.Root.Context.ApplicationID = app2.ID

	test.NoError(t, workflow.Update(db, cache, w1, w1old, proj, u))

	t.Logf("Reloading workflow...")
	w2, err := workflow.LoadByID(db, cache, w1.ID, u)
	test.NoError(t, err)

	assert.Equal(t, w1.ID, w2.ID)
	assert.Equal(t, app2.ID, w2.Root.Context.Application.ID)
	assert.Equal(t, env.ID, w2.Root.Context.Environment.ID)

	test.NoError(t, workflow.Delete(db, w2, u))
}

func TestInsertComplexeWorkflowWithJoins(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip1, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip3, u))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip4, u))

	pip5 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip5",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip5, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip1,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip2,
						Context: &sdk.WorkflowNodeContext{
							Conditions: sdk.WorkflowNodeConditions{
								PlainConditions: []sdk.WorkflowNodeCondition{
									sdk.WorkflowNodeCondition{
										Operator: "=",
										Value:    "master",
										Variable: ".git.branch",
									},
								},
							},
						},
						Triggers: []sdk.WorkflowNodeTrigger{
							sdk.WorkflowNodeTrigger{
								WorkflowDestNode: sdk.WorkflowNode{
									Ref:      "pip3",
									Pipeline: pip3,
									Context: &sdk.WorkflowNodeContext{
										Conditions: sdk.WorkflowNodeConditions{
											PlainConditions: []sdk.WorkflowNodeCondition{
												sdk.WorkflowNodeCondition{
													Operator: "=",
													Value:    "master",
													Variable: ".git.branch",
												},
											},
										},
									},
									Triggers: []sdk.WorkflowNodeTrigger{
										sdk.WorkflowNodeTrigger{
											WorkflowDestNode: sdk.WorkflowNode{
												Ref:      "pip4",
												Pipeline: pip4,
												Context: &sdk.WorkflowNodeContext{
													Conditions: sdk.WorkflowNodeConditions{
														PlainConditions: []sdk.WorkflowNodeCondition{
															sdk.WorkflowNodeCondition{
																Operator: "=",
																Value:    "master",
																Variable: ".git.branch",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Joins: []sdk.WorkflowNodeJoin{
			sdk.WorkflowNodeJoin{
				SourceNodeRefs: []string{
					"pip3", "pip4",
				},
				Triggers: []sdk.WorkflowNodeJoinTrigger{
					sdk.WorkflowNodeJoinTrigger{
						WorkflowDestNode: sdk.WorkflowNode{
							Pipeline: pip5,
							Context: &sdk.WorkflowNodeContext{
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										sdk.WorkflowNodeCondition{
											Operator: "=",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.Root.Pipeline.ID, w1.Root.Pipeline.ID)
	assert.Equal(t, w.Root.Pipeline.Name, w1.Root.Pipeline.Name)
	test.Equal(t, len(w.Root.Triggers), len(w1.Root.Triggers))

	workflow.Sort(&w)

	m1, _ := dump.ToMap(w)
	m2, _ := dump.ToMap(w1)

	keys := []string{}
	for k := range m2 {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		v := m2[k]
		v1, ok := m1[k]
		if ok {
			if v1 == v {
				t.Logf("%s: %s", k, v)
			} else {
				t.Logf("%s: %s but was %s", k, v, v1)
			}
		} else {
			t.Logf("%s: %s but was undefined", k, v)
		}
	}
	assertEqualNode(t, w.Root, w1.Root)

	assert.EqualValues(t, w.Joins[0].Triggers[0].WorkflowDestNode.Context.Conditions, w1.Joins[0].Triggers[0].WorkflowDestNode.Context.Conditions)
	assert.Equal(t, w.Joins[0].Triggers[0].WorkflowDestNode.Pipeline.ID, w1.Joins[0].Triggers[0].WorkflowDestNode.Pipeline.ID)

	assert.Equal(t, pip1.Name, w.Root.Pipeline.Name)
	assert.Equal(t, pip2.Name, w.Root.Triggers[0].WorkflowDestNode.Pipeline.Name)
	assert.Equal(t, pip3.Name, w.Root.Triggers[0].WorkflowDestNode.Triggers[0].WorkflowDestNode.Pipeline.Name)
	assert.Equal(t, pip4.Name, w.Root.Triggers[0].WorkflowDestNode.Triggers[0].WorkflowDestNode.Triggers[0].WorkflowDestNode.Pipeline.Name)
	test.EqualValuesWithoutOrder(t, []int64{
		w1.Root.Triggers[0].WorkflowDestNode.Triggers[0].WorkflowDestNode.ID,
		w1.Root.Triggers[0].WorkflowDestNode.Triggers[0].WorkflowDestNode.Triggers[0].WorkflowDestNode.ID,
	}, w1.Joins[0].SourceNodeIDs)
	assert.Equal(t, pip5.Name, w.Joins[0].Triggers[0].WorkflowDestNode.Pipeline.Name)

}

func TestInsertComplexeWorkflowWithComplexeJoins(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip1, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip3, u))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip4, u))

	pip5 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip5",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip5, u))

	pip6 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip6",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip6, u))

	pip7 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip7",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip7, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip1,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						Pipeline: pip2,
						Context: &sdk.WorkflowNodeContext{
							Conditions: sdk.WorkflowNodeConditions{
								PlainConditions: []sdk.WorkflowNodeCondition{
									sdk.WorkflowNodeCondition{
										Operator: "=",
										Value:    "master",
										Variable: ".git.branch",
									},
								},
							},
						},
						Triggers: []sdk.WorkflowNodeTrigger{
							sdk.WorkflowNodeTrigger{
								WorkflowDestNode: sdk.WorkflowNode{
									Ref:      "pip3",
									Pipeline: pip3,
									Context: &sdk.WorkflowNodeContext{
										Conditions: sdk.WorkflowNodeConditions{
											PlainConditions: []sdk.WorkflowNodeCondition{
												sdk.WorkflowNodeCondition{
													Operator: "=",
													Value:    "master",
													Variable: ".git.branch",
												},
											},
										},
									},
									Triggers: []sdk.WorkflowNodeTrigger{
										sdk.WorkflowNodeTrigger{
											WorkflowDestNode: sdk.WorkflowNode{
												Ref:      "pip4",
												Pipeline: pip4,
												Context: &sdk.WorkflowNodeContext{
													Conditions: sdk.WorkflowNodeConditions{
														PlainConditions: []sdk.WorkflowNodeCondition{
															sdk.WorkflowNodeCondition{
																Operator: "=",
																Value:    "master",
																Variable: ".git.branch",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Joins: []sdk.WorkflowNodeJoin{
			sdk.WorkflowNodeJoin{
				SourceNodeRefs: []string{
					"pip3", "pip4",
				},
				Triggers: []sdk.WorkflowNodeJoinTrigger{
					sdk.WorkflowNodeJoinTrigger{
						WorkflowDestNode: sdk.WorkflowNode{
							Pipeline: pip5,
							Ref:      "pip5",
							Context: &sdk.WorkflowNodeContext{
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										sdk.WorkflowNodeCondition{
											Operator: "=",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
						},
					},
					sdk.WorkflowNodeJoinTrigger{
						WorkflowDestNode: sdk.WorkflowNode{
							Pipeline: pip6,
							Ref:      "pip6",
							Context: &sdk.WorkflowNodeContext{
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										sdk.WorkflowNodeCondition{
											Operator: "=",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
						},
					},
				},
			},
			sdk.WorkflowNodeJoin{
				SourceNodeRefs: []string{
					"pip5", "pip6",
				},
				Triggers: []sdk.WorkflowNodeJoinTrigger{
					sdk.WorkflowNodeJoinTrigger{
						WorkflowDestNode: sdk.WorkflowNode{
							Pipeline: pip7,
							Context: &sdk.WorkflowNodeContext{
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										sdk.WorkflowNodeCondition{
											Operator: "=",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	workflow.Sort(&w)

	m1, _ := dump.ToMap(w)
	m2, _ := dump.ToMap(w1)

	keys := []string{}
	for k := range m2 {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		v := m2[k]
		v1, ok := m1[k]
		if ok {
			if v1 == v {
				t.Logf("%s: %s", k, v)
			} else {
				t.Logf("%s: %s but was %s", k, v, v1)
			}
		} else {
			t.Logf("%s: %s but was undefined", k, v)
		}
	}
	assertEqualNode(t, w.Root, w1.Root)
}

func TestUpdateWorkflowWithJoins(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip3, u))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
			Triggers: []sdk.WorkflowNodeTrigger{
				sdk.WorkflowNodeTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						PipelineID: pip2.ID,
					},
				},
			},
		},
	}

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	w1old := w1
	w1.Name = "test_2"
	w1.Root.PipelineID = pip2.ID
	w1.Root.Pipeline = pip2
	w1.Joins = []sdk.WorkflowNodeJoin{
		sdk.WorkflowNodeJoin{
			SourceNodeRefs: []string{
				fmt.Sprintf("%d", w1.Root.ID),
				fmt.Sprintf("%d", w1.Root.Triggers[0].WorkflowDestNode.ID),
			},
			Triggers: []sdk.WorkflowNodeJoinTrigger{
				sdk.WorkflowNodeJoinTrigger{
					WorkflowDestNode: sdk.WorkflowNode{
						PipelineID: pip3.ID,
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Update(db, cache, w1, w1old, proj, u))

	t.Logf("Reloading workflow...")
	w2, err := workflow.LoadByID(db, cache, w1.ID, u)
	test.NoError(t, err)

	m1, _ := dump.ToMap(w1)
	m2, _ := dump.ToMap(w2)

	keys := []string{}
	for k := range m2 {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		v := m2[k]
		v1, ok := m1[k]
		if ok {
			if v1 == v {
				t.Logf("%s: %s", k, v)
			} else {
				t.Logf("%s: %s but was %s", k, v, v1)
			}
		} else {
			t.Logf("%s: %s but was undefined", k, v)
		}
	}

	test.NoError(t, workflow.Delete(db, w2, u))
}

func TestInsertSimpleWorkflowWithHook(t *testing.T) {
	db, cache := test.SetupPG(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))
	u, _ := assets.InsertAdminUser(db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				Conditions: sdk.WorkflowNodeConditions{
					PlainConditions: []sdk.WorkflowNodeCondition{
						sdk.WorkflowNodeCondition{
							Operator: "=",
							Value:    "master",
							Variable: ".git.branch",
						},
					},
				},
			},
			Hooks: []sdk.WorkflowNodeHook{
				{
					WorkflowHookModel: sdk.WorkflowHookModel{
						Name: workflow.WebHookModel.Name,
					},
					Config: sdk.WorkflowNodeHookConfig{
						"method": sdk.WorkflowNodeHookConfigValue{
							Value:        "POST",
							Configurable: true,
						},
						"username": sdk.WorkflowNodeHookConfigValue{
							Value:        "test",
							Configurable: false,
						},
						"password": sdk.WorkflowNodeHookConfigValue{
							Value:        "password",
							Configurable: false,
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(db, cache, key, "test_1", u)
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.Root.Pipeline.ID, w1.Root.Pipeline.ID)
	assert.Equal(t, w.Root.Pipeline.Name, w1.Root.Pipeline.Name)
	assertEqualNode(t, w.Root, w1.Root)

	ws, err := workflow.LoadAll(db, proj.Key)
	test.NoError(t, err)
	assert.Equal(t, 1, len(ws))

	if t.Failed() {
		return
	}

	assert.Len(t, w.Root.Hooks, 1)
	t.Log(w.Root.Hooks)

	test.NoError(t, workflow.Delete(db, &w, u))
}
