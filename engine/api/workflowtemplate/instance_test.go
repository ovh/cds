package workflowtemplate_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	v2 "github.com/ovh/cds/sdk/exportentities/v2"
)

func TestCheckAndExecuteTemplate(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	grp := proj.ProjectGroups[0].Group
	usr, _ := assets.InsertLambdaUser(t, db, &grp)
	consumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, usr.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser,
		authentication.LoadConsumerOptions.WithConsumerGroups,
	)
	require.NoError(t, err)

	tmplV1 := sdk.WorkflowTemplate{
		Slug:    "my-template",
		Name:    "my-template",
		GroupID: grp.ID,
		Parameters: []sdk.WorkflowTemplateParameter{
			{Key: "param1", Type: sdk.ParameterTypeString},
		},
		Workflow: base64.StdEncoding.EncodeToString([]byte(`
name: v1-[[.name]]
version: v2.0`)),
	}
	_, err = workflowtemplate.Push(context.TODO(), db, &tmplV1, consumer)
	require.NoError(t, err)
	require.NoError(t, workflowtemplate.LoadOptions.WithGroup(context.TODO(), db, &tmplV1))

	tmplV2 := sdk.WorkflowTemplate{
		Slug:    "my-template",
		Name:    "my-template",
		GroupID: grp.ID,
		Parameters: []sdk.WorkflowTemplateParameter{
			{Key: "param1", Type: sdk.ParameterTypeString},
		},
		Workflow: base64.StdEncoding.EncodeToString([]byte(`
name: v2-[[.name]]
version: v2.0`)),
	}
	_, err = workflowtemplate.Push(context.TODO(), db, &tmplV2, consumer)
	require.NoError(t, err)
	require.NoError(t, workflowtemplate.LoadOptions.WithGroup(context.TODO(), db, &tmplV2))

	cases := []struct {
		Name             string
		Data             exportentities.WorkflowComponents
		Detached         bool
		ErrorExists      bool
		InstanceStored   bool
		ExpectedInstance sdk.WorkflowTemplateInstance
		WorkflowExists   bool
		ExpectedWorkflow exportentities.Workflow
	}{{
		Name: "No template given in data should return no error and instance",
		Data: exportentities.WorkflowComponents{},
	}, {
		Name: "Given data is a workflow",
		Data: exportentities.WorkflowComponents{
			Workflow: v2.Workflow{Name: "my-workflow"},
		},
		WorkflowExists:   true,
		ExpectedWorkflow: v2.Workflow{Name: "my-workflow"},
	}, {
		Name: "Given data is the template with v1 version",
		Data: exportentities.WorkflowComponents{
			Template: exportentities.TemplateInstance{
				Name:       "my-workflow",
				From:       tmplV1.PathWithVersion(),
				Parameters: map[string]string{"param1": "value1"},
			},
		},
		InstanceStored: true,
		ExpectedInstance: sdk.WorkflowTemplateInstance{
			Request: sdk.WorkflowTemplateRequest{
				WorkflowName: "my-workflow",
				ProjectKey:   proj.Key,
				Parameters:   map[string]string{"param1": "value1"},
			},
		},
		WorkflowExists:   true,
		ExpectedWorkflow: v2.Workflow{Name: "v1-my-workflow"},
	}, {
		Name: "Given data is the template with latest version",
		Data: exportentities.WorkflowComponents{
			Template: exportentities.TemplateInstance{
				Name:       "my-workflow",
				From:       tmplV1.Path(),
				Parameters: map[string]string{"param1": "value1"},
			},
		},
		InstanceStored: true,
		ExpectedInstance: sdk.WorkflowTemplateInstance{
			Request: sdk.WorkflowTemplateRequest{
				WorkflowName: "my-workflow",
				ProjectKey:   proj.Key,
				Parameters:   map[string]string{"param1": "value1"},
			},
		},
		WorkflowExists:   true,
		ExpectedWorkflow: v2.Workflow{Name: "v2-my-workflow"},
	}, {
		Name: "Given data is a template with detached option",
		Data: exportentities.WorkflowComponents{
			Template: exportentities.TemplateInstance{
				Name:       "my-workflow",
				From:       tmplV1.PathWithVersion(),
				Parameters: map[string]string{"param1": "value1"},
			},
		},
		Detached:         true,
		WorkflowExists:   true,
		ExpectedWorkflow: v2.Workflow{Name: "v1-my-workflow"},
	}, {
		Name: "Invalid given template from",
		Data: exportentities.WorkflowComponents{
			Template: exportentities.TemplateInstance{
				Name: "my-workflow",
				From: "unknown-group/unknown-template",
			},
		},
		ErrorExists: true,
	}}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var mods []workflowtemplate.TemplateRequestModifierFunc
			if c.Detached {
				mods = append(mods, workflowtemplate.TemplateRequestModifiers.Detached)
			}
			_, wti, err := workflowtemplate.CheckAndExecuteTemplate(context.TODO(), db, *consumer, *proj, &c.Data, mods...)
			if c.ErrorExists {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			instanceStored := wti != nil && !wti.Request.Detached
			require.Equal(t, c.InstanceStored, instanceStored, "Instance stored should be %t buf is %t", c.InstanceStored, instanceStored)
			if instanceStored {
				assert.Equal(t, c.ExpectedInstance.Request, wti.Request)
			}

			workflowExists := c.Data.Workflow != nil && c.Data.Workflow.GetName() != ""
			require.Equal(t, c.WorkflowExists, workflowExists, "Workflow exists should be %t buf is %t", c.WorkflowExists, workflowExists)
			if workflowExists {
				assert.Equal(t, c.ExpectedWorkflow.GetName(), c.Data.Workflow.GetName())
			}
		})
	}
}

func TestUpdateTemplateInstanceWithWorkflow(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	grp := proj.ProjectGroups[0].Group
	usr, _ := assets.InsertLambdaUser(t, db, &grp)
	consumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, usr.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser,
		authentication.LoadConsumerOptions.WithConsumerGroups,
	)
	require.NoError(t, err)

	tmpl := sdk.WorkflowTemplate{
		Slug:    "my-template",
		Name:    "my-template",
		GroupID: grp.ID,
		Parameters: []sdk.WorkflowTemplateParameter{
			{Key: "param1", Type: sdk.ParameterTypeString},
		},
		Workflow: base64.StdEncoding.EncodeToString([]byte(`
name: [[.name]]
version: v2.0
workflow:
  Node-1:
    pipeline: Pipeline-[[.id]]`)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
version: v1.0
name: Pipeline-[[.id]]`)),
		}},
	}
	_, err = workflowtemplate.Push(context.TODO(), db, &tmpl, consumer)
	require.NoError(t, err)
	require.NoError(t, workflowtemplate.LoadOptions.WithGroup(context.TODO(), db, &tmpl))

	data := exportentities.WorkflowComponents{
		Template: exportentities.TemplateInstance{
			Name:       "my-workflow",
			From:       tmpl.PathWithVersion(),
			Parameters: map[string]string{"param1": "value1"},
		},
	}
	_, wti, err := workflowtemplate.CheckAndExecuteTemplate(context.TODO(), db, *consumer, *proj, &data)
	require.NoError(t, err)

	_, wkf, _, err := workflow.Push(context.TODO(), db, cache, proj, data, nil, consumer, project.DecryptWithBuiltinKey)
	require.NoError(t, err)

	require.NoError(t, workflowtemplate.UpdateTemplateInstanceWithWorkflow(context.TODO(), db, *wkf, consumer, wti))
	require.NotNil(t, wti.WorkflowID)
	assert.Equal(t, wkf.ID, *wti.WorkflowID)
}
