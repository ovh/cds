package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func TestLoadAllShouldNotReturnAnyWorkflows(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	ws, err := workflow.LoadAll(db, proj.Key)
	test.NoError(t, err)
	assert.Equal(t, 0, len(ws))
}

func TestInsertSimpleWorkflowAndExport(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.WorkflowData.Node.Context.PipelineID, w1.WorkflowData.Node.Context.PipelineID)
	assertEqualNode(t, &w.WorkflowData.Node, &w1.WorkflowData.Node)

	assert.False(t, w1.WorkflowData.Node.Context.Mutex)

	ws, err := workflow.LoadAll(db, proj.Key)
	test.NoError(t, err)
	assert.Equal(t, 1, len(ws))

	exp, err := exportentities.NewWorkflow(*w1)
	test.NoError(t, err)
	btes, err := exportentities.Marshal(exp, exportentities.FormatYAML)
	test.NoError(t, err)
}

func TestInsertSimpleWorkflowWithWrongName(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_ 1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Ref:  "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	assert.Error(t, workflow.Insert(context.TODO(), db, cache, &w, proj))
}

func TestInsertSimpleWorkflowWithApplicationAndEnv(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	app := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app1",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "env1",
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
					EnvironmentID: env.ID,
					Mutex:         true,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.WorkflowData.Node.Context.ApplicationID, w1.WorkflowData.Node.Context.ApplicationID)
	assert.Equal(t, w.WorkflowData.Node.Context.EnvironmentID, w1.WorkflowData.Node.Context.EnvironmentID)
	assert.Equal(t, w.WorkflowData.Node.Context.Mutex, w1.WorkflowData.Node.Context.Mutex)
}

func TestInsertComplexeWorkflowAndExport(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "Root",
				Ref:  "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip1.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "First",
							Ref:  "first",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip2.ID,
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										{
											Operator: "eq",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Name: "Second",
										Ref:  "second",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip3.ID,
											Conditions: sdk.WorkflowNodeConditions{
												PlainConditions: []sdk.WorkflowNodeCondition{
													{
														Operator: "eq",
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
					{
						ChildNode: sdk.Node{
							Name: "Last",
							Ref:  "last",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip4.ID,
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										{
											Operator: "eq",
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

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.WorkflowData.Node.Context.PipelineID, w1.WorkflowData.Node.Context.PipelineID)
	test.Equal(t, len(w.WorkflowData.Node.Triggers), len(w1.WorkflowData.Node.Triggers))

	workflow.Sort(&w)

	assertEqualNode(t, &w.WorkflowData.Node, &w1.WorkflowData.Node)

	exp, err := exportentities.NewWorkflow(w)
	test.NoError(t, err)
	btes, err := exportentities.Marshal(exp, exportentities.FormatYAML)
	test.NoError(t, err)
}

func TestInsertComplexeWorkflowWithBadOperator(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "Root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:   pip1.ID,
					PipelineName: pip1.Name,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "First",
							Context: &sdk.NodeContext{
								PipelineID:   pip2.ID,
								PipelineName: pip2.Name,
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										{
											Operator: "=",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Name: "Second",
										Context: &sdk.NodeContext{
											PipelineID:   pip3.ID,
											PipelineName: pip3.Name,
											Conditions: sdk.WorkflowNodeConditions{
												PlainConditions: []sdk.WorkflowNodeCondition{
													{
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
					{
						ChildNode: sdk.Node{
							Name: "Last",
							Context: &sdk.NodeContext{
								PipelineID:   pip4.ID,
								PipelineName: pip4.Name,
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										{
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

	assert.Error(t, workflow.Insert(context.TODO(), db, cache, &w, proj))
}

func assertEqualNode(t *testing.T, n1, n2 *sdk.Node) {
	t.Logf("assertEqualNode : %d(%s)", n2.ID, n2.Ref)
	workflow.SortNode(n1)
	workflow.SortNode(n2)
	t.Logf("assertEqualNode : Checking hooks")
	test.Equal(t, len(n1.Hooks), len(n2.Hooks))
	t.Logf("assertEqualNode : Checking triggers")
	test.Equal(t, len(n1.Triggers), len(n2.Triggers))

	for i, t1 := range n1.Triggers {
		t2 := n2.Triggers[i]
		test.Equal(t, len(t1.ChildNode.Context.Conditions.PlainConditions), len(t2.ChildNode.Context.Conditions.PlainConditions), "Number of conditions on node does not match")
		test.EqualValuesWithoutOrder(t, t1.ChildNode.Context.Conditions.PlainConditions, t2.ChildNode.Context.Conditions.PlainConditions, "Conditions on triggers does not match")
		assertEqualNode(t, &t1.ChildNode, &t2.ChildNode)
	}
}
func TestUpdateSimpleWorkflowWithApplicationEnvPipelineParametersAndPayload(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Parameter: []sdk.Parameter{
			{
				Name:  "param1",
				Type:  sdk.StringParameter,
				Value: "value1",
			},
		},
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
		Parameter: []sdk.Parameter{
			{
				Name:  "param1",
				Type:  sdk.StringParameter,
				Value: "value1",
			},
		},
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3))

	app := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app1",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app))

	app2 := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "app2",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app2))

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "env1",
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
					EnvironmentID: env.ID,
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
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Context: &sdk.NodeContext{
								PipelineID: pip3.ID,
							},
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	t.Logf("Modifying workflow... with %d instead of %d", app2.ID, app.ID)

	w1.Name = "test_2"
	w1.WorkflowData.Node.Context.PipelineID = pip2.ID
	w1.WorkflowData.Node.Context.ApplicationID = app2.ID

	test.NoError(t, workflow.Update(context.TODO(), db, cache, w1, proj, workflow.UpdateOptions{}))

	t.Logf("Reloading workflow...")
	w2, err := workflow.LoadByID(context.TODO(), db, cache, proj, w1.ID, workflow.LoadOptions{})
	test.NoError(t, err)

	assert.Equal(t, w1.ID, w2.ID)
	assert.Equal(t, app2.ID, w2.WorkflowData.Node.Context.ApplicationID)
	assert.Equal(t, env.ID, w2.WorkflowData.Node.Context.EnvironmentID)

	test.NoError(t, workflow.Delete(context.TODO(), db, cache, proj, w2))
}

func TestInsertComplexeWorkflowWithJoinsAndExport(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4))

	pip5 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip5",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip5))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Joins: []sdk.Node{
				{
					Type: sdk.NodeTypeJoin,
					JoinContext: []sdk.NodeJoin{
						{
							ParentName: "pip3",
						},
						{
							ParentName: "pip4",
						},
					},
					Triggers: []sdk.NodeTrigger{
						{
							ChildNode: sdk.Node{
								Type: sdk.NodeTypePipeline,
								Context: &sdk.NodeContext{
									PipelineID: pip5.ID,
									Conditions: sdk.WorkflowNodeConditions{
										PlainConditions: []sdk.WorkflowNodeCondition{
											{
												Operator: "eq",
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
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip1.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip2.ID,
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										{
											Operator: "eq",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Ref:  "pip3",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip3.ID,
											Conditions: sdk.WorkflowNodeConditions{
												PlainConditions: []sdk.WorkflowNodeCondition{
													{
														Operator: "eq",
														Value:    "master",
														Variable: ".git.branch",
													},
												},
											},
										},
										Triggers: []sdk.NodeTrigger{{
											ChildNode: sdk.Node{
												Ref:  "pip4",
												Type: sdk.NodeTypePipeline,
												Context: &sdk.NodeContext{
													PipelineID: pip4.ID,
													Conditions: sdk.WorkflowNodeConditions{
														PlainConditions: []sdk.WorkflowNodeCondition{
															{
																Operator: "eq",
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
		},
	}

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.WorkflowData.Node.Context.PipelineID, w1.WorkflowData.Node.Context.PipelineID)
	test.Equal(t, len(w.WorkflowData.Node.Triggers), len(w1.WorkflowData.Node.Triggers))

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
	assertEqualNode(t, &w.WorkflowData.Node, &w1.WorkflowData.Node)

	assert.EqualValues(t, w.WorkflowData.Joins[0].Triggers[0].ChildNode.Context.Conditions, w1.WorkflowData.Joins[0].Triggers[0].ChildNode.Context.Conditions)
	assert.Equal(t, w.WorkflowData.Joins[0].Triggers[0].ChildNode.Context.PipelineID, w1.WorkflowData.Joins[0].Triggers[0].ChildNode.Context.PipelineID)

	assert.Equal(t, pip1.ID, w.WorkflowData.Node.Context.PipelineID)
	assert.Equal(t, pip2.ID, w.WorkflowData.Node.Triggers[0].ChildNode.Context.PipelineID)
	assert.Equal(t, pip3.ID, w.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.Context.PipelineID)
	assert.Equal(t, pip4.ID, w.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.Triggers[0].ChildNode.Context.PipelineID)

	log.Warning("%d-%d", w1.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.ID,
		w1.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.Triggers[0].ChildNode.ID)

	log.Warning("%+v", w1.WorkflowData.Joins[0].JoinContext)
	test.EqualValuesWithoutOrder(t, []int64{
		w1.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.ID,
		w1.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.Triggers[0].ChildNode.ID,
	}, []int64{w1.WorkflowData.Joins[0].JoinContext[0].ParentID, w1.WorkflowData.Joins[0].JoinContext[1].ParentID})
	assert.Equal(t, pip5.ID, w.WorkflowData.Joins[0].Triggers[0].ChildNode.Context.PipelineID)

	exp, err := exportentities.NewWorkflow(*w1)
	test.NoError(t, err)
	btes, err := exportentities.Marshal(exp, exportentities.FormatYAML)
	test.NoError(t, err)
}

func TestInsertComplexeWorkflowWithComplexeJoins(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip1 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4))

	pip5 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip5",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip5))

	pip6 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip6",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip6))

	pip7 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip7",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip7))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip1.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip2.ID,
								Conditions: sdk.WorkflowNodeConditions{
									PlainConditions: []sdk.WorkflowNodeCondition{
										{
											Operator: "eq",
											Value:    "master",
											Variable: ".git.branch",
										},
									},
								},
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Ref:  "pip3",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip3.ID,
											Conditions: sdk.WorkflowNodeConditions{
												PlainConditions: []sdk.WorkflowNodeCondition{
													{
														Operator: "eq",
														Value:    "master",
														Variable: ".git.branch",
													},
												},
											},
										},
										Triggers: []sdk.NodeTrigger{
											{
												ChildNode: sdk.Node{
													Ref:  "pip4",
													Type: sdk.NodeTypePipeline,
													Context: &sdk.NodeContext{
														PipelineID: pip4.ID,
														Conditions: sdk.WorkflowNodeConditions{
															PlainConditions: []sdk.WorkflowNodeCondition{
																{
																	Operator: "eq",
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
			Joins: []sdk.Node{
				{
					Type: sdk.NodeTypeJoin,
					JoinContext: []sdk.NodeJoin{
						{
							ParentName: "pip3",
						},
						{
							ParentName: "pip4",
						},
					},
					Triggers: []sdk.NodeTrigger{
						{
							ChildNode: sdk.Node{
								Ref:  "pip5",
								Type: sdk.NodeTypePipeline,
								Context: &sdk.NodeContext{
									PipelineID: pip5.ID,
									Conditions: sdk.WorkflowNodeConditions{
										PlainConditions: []sdk.WorkflowNodeCondition{
											{
												Operator: "eq",
												Value:    "master",
												Variable: ".git.branch",
											},
										},
									},
								},
							},
						},
						{
							ChildNode: sdk.Node{
								Ref:  "pip6",
								Type: sdk.NodeTypePipeline,
								Context: &sdk.NodeContext{
									PipelineID: pip6.ID,
									Conditions: sdk.WorkflowNodeConditions{
										PlainConditions: []sdk.WorkflowNodeCondition{
											{
												Operator: "eq",
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
				{
					Type: sdk.NodeTypeJoin,
					JoinContext: []sdk.NodeJoin{
						{
							ParentName: "pip5",
						},
						{
							ParentName: "pip6",
						},
					},
					Triggers: []sdk.NodeTrigger{
						{
							ChildNode: sdk.Node{
								Type: sdk.NodeTypePipeline,
								Context: &sdk.NodeContext{
									PipelineID: pip7.ID,
									Conditions: sdk.WorkflowNodeConditions{
										PlainConditions: []sdk.WorkflowNodeCondition{
											{
												Operator: "eq",
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
		Notifications: []sdk.WorkflowNotification{
			{
				Type:           "jabber",
				SourceNodeRefs: []string{"pip6", "pip5"},
				Settings: sdk.UserNotificationSettings{
					OnFailure:    sdk.UserNotificationAlways,
					OnStart:      &sdk.True,
					OnSuccess:    sdk.UserNotificationAlways,
					SendToAuthor: &sdk.True,
					SendToGroups: &sdk.True,
					Template: &sdk.UserNotificationTemplate{
						Body:    "body",
						Subject: "title",
					},
				},
			},
		},
	}

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
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
	assertEqualNode(t, &w.WorkflowData.Node, &w1.WorkflowData.Node)
}

func TestUpdateWorkflowWithJoins(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip2.ID,
							},
						},
					},
				},
			},
		},
	}

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	//w1old := *w1
	w1.Name = "test_2"
	w1.WorkflowData.Joins = []sdk.Node{
		{
			Type: sdk.NodeTypeJoin,
			JoinContext: []sdk.NodeJoin{
				{
					ParentName: "pip1",
				},
				{
					ParentName: "pip2",
				},
			},
			Triggers: []sdk.NodeTrigger{
				{
					ChildNode: sdk.Node{
						Type: sdk.NodeTypePipeline,
						Context: &sdk.NodeContext{
							PipelineID: pip3.ID,
						},
					},
				},
			},
		},
	}

	test.NoError(t, workflow.RenameNode(context.TODO(), db, w1))

	test.NoError(t, workflow.Update(context.TODO(), db, cache, w1, proj, workflow.UpdateOptions{}))

	t.Logf("Reloading workflow...")
	w2, err := workflow.LoadByID(context.TODO(), db, cache, proj, w1.ID, workflow.LoadOptions{})
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

	test.NoError(t, workflow.Delete(context.TODO(), db, cache, proj, w2))
}

func TestInsertSimpleWorkflowWithHookAndExport(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))
	test.NoError(t, workflow.CreateBuiltinWorkflowOutgoingHookModels(db))

	hookModels, err := workflow.LoadHookModels(db)
	test.NoError(t, err)
	var webHookID int64
	for _, h := range hookModels {
		if h.Name == sdk.WebHookModel.Name {
			webHookID = h.ID
			break
		}
	}

	mockHookSservice, _ := assets.InsertService(t, db, "TestManualRunBuildParameterMultiApplication", services.TypeHooks)
	defer func() {
		services.Delete(db, mockHookSservice) // nolint
	}()

	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			// NEED get REPO
			case "/task/bulk":
				h := sdk.NodeHook{
					HookModelID: webHookID,
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
						"payload": sdk.WorkflowNodeHookConfigValue{
							Value:        "{}",
							Configurable: true,
						},
						"URL": sdk.WorkflowNodeHookConfigValue{
							Value:        "https://www.github.com",
							Configurable: true,
						},
					},
				}
				if err := enc.Encode(map[string]sdk.NodeHook{"uuid": h}); err != nil {
					return writeError(w, err)
				}
			default:
				t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
			}

			return w, nil
		},
	)

	outHookModels, err := workflow.LoadOutgoingHookModels(db)
	test.NoError(t, err)
	var outWebHookID int64
	for _, h := range outHookModels {
		if h.Name == sdk.OutgoingWebHookModel.Name {
			outWebHookID = h.ID
			break
		}
	}

	u, _ := assets.InsertAdminUser(db)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
					Conditions: sdk.WorkflowNodeConditions{
						PlainConditions: []sdk.WorkflowNodeCondition{
							{
								Operator: "eq",
								Value:    "master",
								Variable: ".git.branch",
							},
						},
					},
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelID: webHookID,
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
							"payload": sdk.WorkflowNodeHookConfigValue{
								Value:        "{}",
								Configurable: true,
							},
							"URL": sdk.WorkflowNodeHookConfigValue{
								Value:        "https://www.github.com",
								Configurable: true,
							},
						},
					},
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypeOutGoingHook,
							OutGoingHookContext: &sdk.NodeOutGoingHook{
								HookModelID: outWebHookID,
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
									"payload": sdk.WorkflowNodeHookConfigValue{
										Value:        "{}",
										Configurable: true,
									},
									"URL": sdk.WorkflowNodeHookConfigValue{
										Value:        "https://www.github.com",
										Configurable: true,
									},
								},
							},
						},
					},
					{
						ChildNode: sdk.Node{
							Type: sdk.NodeTypeOutGoingHook,
							OutGoingHookContext: &sdk.NodeOutGoingHook{
								HookModelID: outWebHookID,
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
									"payload": sdk.WorkflowNodeHookConfigValue{
										Value:        "{}",
										Configurable: true,
									},
									"URL": sdk.WorkflowNodeHookConfigValue{
										Value:        "https://www.github.com",
										Configurable: true,
									},
								},
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip.ID,
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

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj), "unable to insert workflow")

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)

	assert.Equal(t, w.ID, w1.ID)
	assert.Equal(t, w.ProjectID, w1.ProjectID)
	assert.Equal(t, w.Name, w1.Name)
	assert.Equal(t, w.WorkflowData.Node.Context.PipelineID, w1.WorkflowData.Node.Context.PipelineID)
	assertEqualNode(t, &w.WorkflowData.Node, &w1.WorkflowData.Node)

	ws, err := workflow.LoadAll(db, proj.Key)
	test.NoError(t, err)
	assert.Equal(t, 1, len(ws))

	if t.Failed() {
		return
	}

	assert.Len(t, w.WorkflowData.Node.Hooks, 1)

	exp, err := exportentities.NewWorkflow(*w1)
	test.NoError(t, err)
	btes, err := exportentities.Marshal(exp, exportentities.FormatYAML)
	test.NoError(t, err)

	test.NoError(t, workflow.Delete(context.TODO(), db, cache, proj, &w))
}
