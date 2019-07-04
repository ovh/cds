package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

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

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
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

	fmt.Println(string(btes))
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	assert.Error(t, workflow.Insert(db, cache, &w, proj, u))
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

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

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3, u))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
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

	fmt.Println(string(btes))
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3, u))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	assert.Error(t, workflow.Insert(db, cache, &w, proj, u))
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3, u))

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

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
	test.NoError(t, err)

	t.Logf("Modifying workflow... with %d instead of %d", app2.ID, app.ID)

	w1.Name = "test_2"
	w1.WorkflowData.Node.Context.PipelineID = pip2.ID
	w1.WorkflowData.Node.Context.ApplicationID = app2.ID

	test.NoError(t, workflow.Update(context.TODO(), db, cache, w1, proj, u, workflow.UpdateOptions{}))

	t.Logf("Reloading workflow...")
	w2, err := workflow.LoadByID(db, cache, proj, w1.ID, u, workflow.LoadOptions{})
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3, u))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4, u))

	pip5 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip5",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip5, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.RenameNode(db, &w))
	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
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

	fmt.Println(string(btes))
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3, u))

	pip4 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip4",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip4, u))

	pip5 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip5",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip5, u))

	pip6 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip6",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip6, u))

	pip7 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip7",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip7, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.RenameNode(db, &w))
	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2, u))

	pip3 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip3",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3, u))

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

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.RenameNode(db, &w))
	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
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

	test.NoError(t, workflow.RenameNode(db, w1))

	test.NoError(t, workflow.Update(context.TODO(), db, cache, w1, proj, u, workflow.UpdateOptions{}))

	t.Logf("Reloading workflow...")
	w2, err := workflow.LoadByID(db, cache, proj, w1.ID, u, workflow.LoadOptions{})
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

	mockHookSservice := &sdk.Service{Name: "TestManualRunBuildParameterMultiApplication", Type: services.TypeHooks}
	test.NoError(t, services.Insert(db, mockHookSservice))
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
				var hooks map[string]sdk.NodeHook
				bts, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				if err := json.Unmarshal(bts, &hooks); err != nil {
					return writeError(w, err)
				}
				k := reflect.ValueOf(hooks).MapKeys()[0].String()

				hooks[k].Config["method"] = sdk.WorkflowNodeHookConfigValue{
					Value:        "POST",
					Configurable: true,
				}
				hooks[k].Config["username"] = sdk.WorkflowNodeHookConfigValue{
					Value:        "test",
					Configurable: false,
				}
				hooks[k].Config["password"] = sdk.WorkflowNodeHookConfigValue{
					Value:        "password",
					Configurable: false,
				}
				hooks[k].Config["payload"] = sdk.WorkflowNodeHookConfigValue{
					Value:        "{}",
					Configurable: true,
				}
				hooks[k].Config["URL"] = sdk.WorkflowNodeHookConfigValue{
					Value:        "https://www.github.com",
					Configurable: true,
				}
				if err := enc.Encode(hooks); err != nil {
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

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

	test.NoError(t, workflow.RenameNode(db, &w))
	test.NoError(t, workflow.Insert(db, cache, &w, proj, u), "unable to insert workflow")

	w1, err := workflow.Load(context.TODO(), db, cache, proj, "test_1", u, workflow.LoadOptions{})
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

	fmt.Println(string(btes))

	test.NoError(t, workflow.Delete(context.TODO(), db, cache, proj, &w))
}

func TestInsertAndDeleteMultiHook(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	hookModels, err := workflow.LoadHookModels(db)
	test.NoError(t, err)
	var webHookID int64
	var schedulerID int64
	var repoWebHookID int64
	for _, h := range hookModels {
		if h.Name == sdk.WebHookModel.Name {
			webHookID = h.ID
		}
		if h.Name == sdk.RepositoryWebHookModel.Name {
			repoWebHookID = h.ID
		}
		if h.Name == sdk.SchedulerModel.Name {
			schedulerID = h.ID
		}
	}

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	_, err = db.Exec("DELETE FROM services")
	assert.NoError(t, err)

	mockVCSSservice := &sdk.Service{Name: "TestInsertAndDeleteMultiHookVCS", Type: services.TypeVCS}
	test.NoError(t, services.Insert(db, mockVCSSservice))

	mockHookServices := &sdk.Service{Name: "TestInsertAndDeleteMultiHookHook", Type: services.TypeHooks}
	test.NoError(t, services.Insert(db, mockHookServices))

	//This is a mock for the vcs service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)
			switch r.URL.String() {
			// NEED get REPO

			case "/vcs/github/repos/sguiheux/demo":
				repo := sdk.VCSRepo{
					URL:          "https",
					Name:         "demo",
					ID:           "123",
					Fullname:     "sguiheux/demo",
					Slug:         "sguiheux",
					HTTPCloneURL: "https://github.com/sguiheux/demo.git",
					SSHCloneURL:  "git://github.com/sguiheux/demo.git",
				}
				if err := enc.Encode(repo); err != nil {
					return writeError(w, err)
				}

				// NEED for default payload on insert
			case "/vcs/github/repos/sguiheux/demo/branches":
				b := sdk.VCSBranch{
					Default:      true,
					DisplayID:    "master",
					LatestCommit: "mylastcommit",
				}
				if err := enc.Encode([]sdk.VCSBranch{b}); err != nil {
					return writeError(w, err)
				}
			case "/task/bulk":
				var hooks map[string]sdk.NodeHook
				request, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				if err := json.Unmarshal(request, &hooks); err != nil {
					return writeError(w, err)
				}
				if len(hooks) != 1 {
					return writeError(w, fmt.Errorf("Must only have 1 hook"))
				}
				k := reflect.ValueOf(hooks).MapKeys()[0].String()
				hooks[k].Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
					Value:        fmt.Sprintf("http://6.6.6:8080/%s", hooks[k].UUID),
					Type:         "string",
					Configurable: false,
				}

				if err := enc.Encode(map[string]sdk.NodeHook{
					hooks[k].UUID: hooks[k],
				}); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/webhooks":

				infos := repositoriesmanager.WebhooksInfos{
					WebhooksDisabled:  false,
					WebhooksSupported: true,
					Icon:              "github",
				}
				if err := enc.Encode(infos); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/sguiheux/demo/hooks":
				pr := sdk.VCSHook{
					ID: "666",
				}
				if err := enc.Encode(pr); err != nil {
					return writeError(w, err)
				}
			default:
				if strings.HasPrefix(r.URL.String(), "/vcs/github/repos/sguiheux/demo/hooks?url=htt") && strings.HasSuffix(r.URL.String(), "&id=666") {
					// Do NOTHING
				} else {
					t.Fatalf("UNKNOWN ROUTE: %s", r.URL.String())
				}

			}

			return w, nil
		},
	)

	pip := &sdk.Pipeline{
		Name: "build",
		Stages: []sdk.Stage{
			{
				Name:       "stage1",
				BuildOrder: 1,
				Enabled:    true,
				Jobs: []sdk.Job{
					{
						Enabled: true,
						Action: sdk.Action{
							Name:    "JOb1",
							Enabled: true,
							Actions: []sdk.Action{
								{
									Name:    "gitClone",
									Type:    sdk.BuiltinAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "branch",
											Value: "{{.git.branch}}",
										},
										{
											Name:  "commit",
											Value: "{{.git.hash}}",
										},
										{
											Name:  "directory",
											Value: "{{.cds.workspace}}",
										},
										{
											Name:  "password",
											Value: "",
										},
										{
											Name:  "privateKey",
											Value: "",
										},
										{
											Name:  "url",
											Value: "{{.git.url}}",
										},
										{
											Name:  "user",
											Value: "",
										},
										{
											Name:  "depth",
											Value: "12",
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
	assert.NoError(t, pipeline.Import(context.TODO(), db, cache, proj, pip, nil, u))
	var errPip error
	pip, errPip = pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, errPip)

	// Add application
	appS := `version: v1.0
name: blabla
vcs_server: github
repo: sguiheux/demo
vcs_ssh_key: proj-blabla
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, globalError := application.ParseAndImport(db, cache, proj, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)

	proj.Applications = append(proj.Applications, *app)
	proj.Pipelines = append(proj.Pipelines, *pip)

	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    proj.Pipelines[0].ID,
					ApplicationID: proj.Applications[0].ID,
				},
				Hooks: []sdk.NodeHook{
					{
						Config:      sdk.RepositoryWebHookModel.DefaultConfig,
						HookModelID: repoWebHookID,
					},
				},
			},
		},
		Applications: map[int64]sdk.Application{
			proj.Applications[0].ID: proj.Applications[0],
		},
		Pipelines: map[int64]sdk.Pipeline{
			proj.Pipelines[0].ID: proj.Pipelines[0],
		},
	}
	assert.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	// Add check on Hook
	assert.Equal(t, "666", w.WorkflowData.Node.Hooks[0].Config["webHookID"].Value)
	assert.Equal(t, "github", w.WorkflowData.Node.Hooks[0].Config["hookIcon"].Value)
	assert.Equal(t, fmt.Sprintf("http://6.6.6:8080/%s", w.WorkflowData.Node.Hooks[0].UUID), w.WorkflowData.Node.Hooks[0].Config["webHookURL"].Value)
	t.Logf("%+v", w.WorkflowData.Node.Hooks[0])

	// Load workflow
	oldW, err := workflow.LoadByID(db, cache, proj, w.ID, u, workflow.LoadOptions{})
	assert.NoError(t, err)

	// Add WEB HOOK
	w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, sdk.NodeHook{
		Config:      sdk.WebHookModel.DefaultConfig,
		HookModelID: webHookID,
	})

	assert.NoError(t, workflow.Update(context.TODO(), db, cache, &w, proj, u, workflow.UpdateOptions{OldWorkflow: oldW}))

	// Add check on HOOKS
	assert.Equal(t, 2, len(w.WorkflowData.Node.Hooks))
	for _, h := range w.WorkflowData.Node.Hooks {
		if h.HookModelID == repoWebHookID {
			assert.True(t, oldW.WorkflowData.Node.Hooks[0].Equals(h))
		} else if h.HookModelID == webHookID {
			assert.Equal(t, fmt.Sprintf("http://6.6.6:8080/%s", h.UUID), h.Config["webHookURL"].Value)
		} else {
			// Must not go here
			t.Fail()
		}

	}

	oldW, err = workflow.LoadByID(db, cache, proj, w.ID, u, workflow.LoadOptions{})
	assert.NoError(t, err)

	// Add Scheduler
	w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks, sdk.NodeHook{
		Config:      sdk.SchedulerModel.DefaultConfig,
		HookModelID: schedulerID,
	})

	assert.NoError(t, workflow.Update(context.TODO(), db, cache, &w, proj, u, workflow.UpdateOptions{OldWorkflow: oldW}))

	// Add check on HOOKS
	assert.Equal(t, 3, len(w.WorkflowData.Node.Hooks))

	oldHooks := oldW.WorkflowData.GetHooks()
	for _, h := range w.WorkflowData.Node.Hooks {
		if h.HookModelID == repoWebHookID {
			assert.True(t, h.Equals(*oldHooks[h.UUID]))
		} else if h.HookModelID == webHookID {
			assert.True(t, h.Equals(*oldHooks[h.UUID]))
		} else if h.HookModelID == schedulerID {
			assert.Contains(t, h.Config["payload"].Value, "git.branch")
			assert.Contains(t, h.Config["payload"].Value, "git.author")
			assert.Contains(t, h.Config["payload"].Value, "git.hash")
			assert.Contains(t, h.Config["payload"].Value, "git.hash.before")
			assert.Contains(t, h.Config["payload"].Value, "git.message")
			assert.Contains(t, h.Config["payload"].Value, "git.repository")
			assert.Contains(t, h.Config["payload"].Value, "git.tag")
		} else {
			// Must not go here
			t.Fail()
		}
	}

	oldW, err = workflow.LoadByID(db, cache, proj, w.ID, u, workflow.LoadOptions{})
	assert.NoError(t, err)

	// Delete repository webhook
	var index = 0
	for i, h := range w.WorkflowData.Node.Hooks {
		if h.HookModelID == repoWebHookID {
			index = i
		}
	}
	w.WorkflowData.Node.Hooks = append(w.WorkflowData.Node.Hooks[:index], w.WorkflowData.Node.Hooks[index+1:]...)
	assert.NoError(t, workflow.Update(context.TODO(), db, cache, &w, proj, u, workflow.UpdateOptions{OldWorkflow: oldW}))

	// Add check on HOOKS
	assert.Equal(t, 2, len(w.WorkflowData.Node.Hooks))

	oldHooks = oldW.WorkflowData.GetHooks()
	for _, h := range w.WorkflowData.Node.Hooks {
		if h.HookModelID == webHookID {
			assert.True(t, h.Equals(*oldHooks[h.UUID]))
		} else if h.HookModelID == schedulerID {
			assert.True(t, h.Equals(*oldHooks[h.UUID]))
		} else {
			// Must not go here
			t.Fail()
		}
	}

}
