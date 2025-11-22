package api

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestWorkflowTrigger_JobTemplateInsideTemplate(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	admin, _ := assets.InsertAdminUser(t, db)

	org, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	require.NoError(t, err)

	reg := sdk.Region{
		Name: "build",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	api.Config.Workflow.JobDefaultRegion = reg.Name

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       "var1",
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		Regions: []sdk.RBACRegion{
			{
				RegionID:            reg.ID,
				AllUsers:            true,
				RBACOrganizationIDs: []string{org.ID},
				Role:                sdk.RegionRoleExecute,
			},
		},
		RegionProjects: []sdk.RBACRegionProject{
			{
				Role:        sdk.RegionRoleExecute,
				AllProjects: true,
				RegionID:    reg.ID,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    build:
    test:
    deploy:
      from: .cds/workflow-templates/mytmpl2.yml`,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

	eTmpl2 := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl2.yml",
		Name:                "myJobTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    it:
    it2:
    it3:
      needs: [it,it2]
    it4:
      needs: [it3]`,
	}
	require.NoError(t, entity.Insert(ctx, db, &eTmpl2))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123456789",
		WorkflowRef:  "refs/heads/master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: "myworkflow",
			Jobs: map[string]sdk.V2Job{
				"root": {
					From: ".cds/workflow-templates/mytmpl.yml",
				},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	wrAfter1, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for j := range wrAfter1.WorkflowData.Workflow.Jobs {
		t.Logf("Job %s", j)
	}

	require.Equal(t, 3, len(wrAfter1.WorkflowData.Workflow.Jobs)) // root must be replaced by build / test/ deploy
	_, has := wrAfter1.WorkflowData.Workflow.Jobs["build"]
	require.True(t, has)
	_, has = wrAfter1.WorkflowData.Workflow.Jobs["test"]
	require.True(t, has)
	_, has = wrAfter1.WorkflowData.Workflow.Jobs["deploy"]
	require.True(t, has)

	rjs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 0, len(rjs)) // No run jobs

	// Trigger again
	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))
	rjs, err = workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	for _, rj := range rjs {
		t.Logf("RunJob: %s status: %s", rj.JobID, rj.Status)
	}
	require.Equal(t, 2, len(rjs)) //build and test are success

	wrAfter2, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for j := range wrAfter2.WorkflowData.Workflow.Jobs {
		t.Logf("Job %s", j)
	}
	require.Equal(t, 6, len(wrAfter2.WorkflowData.Workflow.Jobs)) //build / test /  it / it2 / it3 / it4
	_, has = wrAfter2.WorkflowData.Workflow.Jobs["build"]
	require.True(t, has)
	_, has = wrAfter2.WorkflowData.Workflow.Jobs["test"]
	require.True(t, has)
	_, has = wrAfter2.WorkflowData.Workflow.Jobs["it"]
	require.True(t, has)
	_, has = wrAfter2.WorkflowData.Workflow.Jobs["it2"]
	require.True(t, has)
	_, has = wrAfter2.WorkflowData.Workflow.Jobs["it3"]
	require.True(t, has)
	_, has = wrAfter2.WorkflowData.Workflow.Jobs["it4"]
	require.True(t, has)

}

func TestWorkflowTrigger_JobTemplateDuplicateJob(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	admin, _ := assets.InsertAdminUser(t, db)

	org, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	require.NoError(t, err)

	reg := sdk.Region{
		Name: "build",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	api.Config.Workflow.JobDefaultRegion = reg.Name

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       "var1",
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		Regions: []sdk.RBACRegion{
			{
				RegionID:            reg.ID,
				AllUsers:            true,
				RBACOrganizationIDs: []string{org.ID},
				Role:                sdk.RegionRoleExecute,
			},
		},
		RegionProjects: []sdk.RBACRegionProject{
			{
				Role:        sdk.RegionRoleExecute,
				AllProjects: true,
				RegionID:    reg.ID,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    build:
    test:
    deploy:
      from: .cds/workflow-templates/mytmpl2.yml`,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

	eTmpl2 := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl2.yml",
		Name:                "myJobTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    it:
    it2:
    it3:
      needs: [it,it2]
    it4:
      needs: [it3]`,
	}
	require.NoError(t, entity.Insert(ctx, db, &eTmpl2))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123456789",
		WorkflowRef:  "refs/heads/master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: "myworkflow",
			Jobs: map[string]sdk.V2Job{
				"root": {
					From: ".cds/workflow-templates/mytmpl.yml",
				},
				"root2": {
					From: ".cds/workflow-templates/mytmpl.yml",
				},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, ri := range runInfos {
		t.Logf("RunInfo: %s", ri.Message)
	}
	require.Equal(t, 1, len(runInfos))
	require.Contains(t, runInfos[0].Message, "already exist in the parent workflow")

	wrAfter1, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrAfter1.Status)

}

func TestWorkflowTrigger_JobTemplateAddStageOnNonStagedWorkflow(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	admin, _ := assets.InsertAdminUser(t, db)

	org, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	require.NoError(t, err)

	reg := sdk.Region{
		Name: "build",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	api.Config.Workflow.JobDefaultRegion = reg.Name

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       "var1",
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		Regions: []sdk.RBACRegion{
			{
				RegionID:            reg.ID,
				AllUsers:            true,
				RBACOrganizationIDs: []string{org.ID},
				Role:                sdk.RegionRoleExecute,
			},
		},
		RegionProjects: []sdk.RBACRegionProject{
			{
				Role:        sdk.RegionRoleExecute,
				AllProjects: true,
				RegionID:    reg.ID,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  stages:
    build: {}
  jobs:
    build:
      stage: build
    test:
      stage: build
    deploy:
      stage: build
      from: .cds/workflow-templates/mytmpl2.yml`,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

	eTmpl2 := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl2.yml",
		Name:                "myJobTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    it:
    it2:
    it3:
      needs: [it,it2]
    it4:
      needs: [it3]`,
	}
	require.NoError(t, entity.Insert(ctx, db, &eTmpl2))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123456789",
		WorkflowRef:  "refs/heads/master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: "myworkflow",
			Jobs: map[string]sdk.V2Job{
				"root": {
					From: ".cds/workflow-templates/mytmpl.yml",
				},
				"root2": {},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, ri := range runInfos {
		t.Logf("RunInfo: %s", ri.Message)
	}
	require.Equal(t, 1, len(runInfos))
	require.Contains(t, runInfos[0].Message, "workflow myworkflow: missing stage on job root2")

	wrAfter1, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for k, j := range wrAfter1.WorkflowData.Workflow.Jobs {
		t.Logf("Job %s: %s", k, j.Stage)
	}

	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrAfter1.Status)

}

func TestWorkflowTrigger_JobTemplateNoStageOnTemplate(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	admin, _ := assets.InsertAdminUser(t, db)

	org, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	require.NoError(t, err)

	reg := sdk.Region{
		Name: "build",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	api.Config.Workflow.JobDefaultRegion = reg.Name

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       "var1",
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		Regions: []sdk.RBACRegion{
			{
				RegionID:            reg.ID,
				AllUsers:            true,
				RBACOrganizationIDs: []string{org.ID},
				Role:                sdk.RegionRoleExecute,
			},
		},
		RegionProjects: []sdk.RBACRegionProject{
			{
				Role:        sdk.RegionRoleExecute,
				AllProjects: true,
				RegionID:    reg.ID,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    build:
    test:
    deploy:
      from: .cds/workflow-templates/mytmpl2.yml`,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123456789",
		WorkflowRef:  "refs/heads/master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: "myworkflow",
			Stages: map[string]sdk.WorkflowStage{
				"build": {},
			},
			Jobs: map[string]sdk.V2Job{
				"root": {
					From:  ".cds/workflow-templates/mytmpl.yml",
					Stage: "build",
				},
				"root2": {
					Stage: "build",
				},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, ri := range runInfos {
		t.Logf("RunInfo: %s", ri.Message)
	}
	require.Equal(t, 0, len(runInfos))

	wrAfter1, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for k, j := range wrAfter1.WorkflowData.Workflow.Jobs {
		t.Logf("Job %s: %s", k, j.Stage)
		require.Equal(t, "build", j.Stage)
	}
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrAfter1.Status)
}
