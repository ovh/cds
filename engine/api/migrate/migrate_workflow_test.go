package migrate

import (
	"context"
	"fmt"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_ToWorkflow(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	app1 := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	app2 := &sdk.Application{
		Name: sdk.RandomString(10),
	}

	test.NoError(t, application.Insert(db, cache, proj, app1, u))
	test.NoError(t, application.Insert(db, cache, proj, app2, u))

	pip1 := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		Type:      "build",
		ProjectID: proj.ID,
	}

	pip2 := &sdk.Pipeline{
		Name: sdk.RandomString(10),
		Type: "deployment",
		Parameter: []sdk.Parameter{
			{
				Name:  "param1",
				Type:  "string",
				Value: "",
			},
		},
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, pip1, u))
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, pip2, u))

	env1 := &sdk.Environment{
		Name:      "env1",
		ProjectID: proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(db, env1))

	oldW := []sdk.CDPipeline{
		{
			Application: *app1,
			Pipeline:    *pip1,
			SubPipelines: []sdk.CDPipeline{
				{
					Application: *app2,
					Pipeline:    *pip2,
					Trigger: sdk.PipelineTrigger{
						Parameters: []sdk.Parameter{
							{
								Name:  "param1",
								Value: "valueTriggered",
							},
						},
						Prerequisites: []sdk.Prerequisite{
							{
								Parameter:     "git.branch",
								ExpectedValue: "master",
							},
						},
						Manual: true,
					},
					Environment: *env1,
				},
			},
		},
	}

	proj2, errP := project.Load(db, cache, proj.Key, u, project.LoadOptions.WithEnvironments, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines)
	fmt.Printf("%+v", proj2)
	test.NoError(t, errP)
	wfs, err := ToWorkflow(db, cache, oldW, proj2, u, true, false, true, true)
	test.NoError(t, err)
	assert.Equal(t, 1, len(wfs))

	app2DB, errA := application.LoadByName(db, cache, proj2.Key, app2.Name, u)
	test.NoError(t, errA)
	assert.Equal(t, "CLEANING", app2DB.WorkflowMigration)

	wf, errW := workflow.Load(context.TODO(), db, cache, proj, "w"+app1.Name, u, workflow.LoadOptions{})
	test.NoError(t, errW)

	assert.Equal(t, pip1.ID, wf.Root.PipelineID)
	assert.Equal(t, 1, len(wf.Root.Triggers))
	assert.Equal(t, pip2.ID, wf.Root.Triggers[0].WorkflowDestNode.PipelineID)
	assert.Equal(t, env1.ID, wf.Root.Triggers[0].WorkflowDestNode.Context.Environment.ID)
	assert.Equal(t, "master", wf.Root.Triggers[0].WorkflowDestNode.Context.Conditions.PlainConditions[0].Value)
	assert.Equal(t, "git.branch", wf.Root.Triggers[0].WorkflowDestNode.Context.Conditions.PlainConditions[0].Variable)
	assert.Equal(t, 1, len(wf.Root.Triggers[0].WorkflowDestNode.Context.DefaultPipelineParameters))
	assert.Equal(t, "valueTriggered", wf.Root.Triggers[0].WorkflowDestNode.Context.DefaultPipelineParameters[0].Value)
	assert.Equal(t, "param1", wf.Root.Triggers[0].WorkflowDestNode.Context.DefaultPipelineParameters[0].Name)
	assert.Equal(t, 3, len(wf.Root.Triggers[0].WorkflowDestNode.Context.Conditions.PlainConditions))

}
