package api

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/hatchery"
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

// A job template whose content declares a concrete job with a matrix strategy:
// the first trigger replaces the templated job by the matrix job in the workflow
// definition; the second trigger enqueues one run job per matrix permutation.
func TestWorkflowTrigger_JobTemplateContainingMatrixJob(t *testing.T) {
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

	// Hatchery allowed to spawn docker jobs on the region
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))
	rbHatch := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rbHatch))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/mymodel.yml",
		Name:                "mymodel",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`,
	}
	require.NoError(t, entity.Insert(ctx, db, &entityModel))

	// Template containing a concrete job with a matrix strategy
	entityTmpl := sdk.Entity{
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
    deploy:
      runs-on: .cds/worker-models/mymodel.yml
      strategy:
        matrix:
          region: [region1, region2]
      steps:
      - run: echo "Deploy ${{ matrix.region }}"`,
	}
	require.NoError(t, entity.Insert(ctx, db, &entityTmpl))

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

	// First trigger: the templated job is expanded into the workflow definition, no run job yet
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
	require.Equal(t, 1, len(wrAfter1.WorkflowData.Workflow.Jobs))
	deployJob, has := wrAfter1.WorkflowData.Workflow.Jobs["deploy"]
	require.True(t, has)
	require.NotNil(t, deployJob.Strategy)

	rjs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 0, len(rjs))

	// Second trigger: all matrix permutations must be enqueued
	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	runInfos, err = workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, ri := range runInfos {
		t.Logf("RunInfo: %s", ri.Message)
	}
	require.Equal(t, 0, len(runInfos))

	rjs, err = workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 2, len(rjs))
	permutations := map[string]bool{}
	for _, rj := range rjs {
		require.Equal(t, "deploy", rj.JobID)
		require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, rj.Status)
		permutations[rj.Matrix["region"]] = true
	}
	require.True(t, permutations["region1"])
	require.True(t, permutations["region2"])
}

// A matrix job coming from a job template with some permutations already run:
// triggering the workflow must enqueue only the missing permutations, leaving
// the already-run ones untouched.
func TestWorkflowTrigger_JobTemplateContainingMatrixJobPartialPermutations(t *testing.T) {
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

	// Hatchery allowed to spawn docker jobs on the region
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))
	rbHatch := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rbHatch))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/mymodel.yml",
		Name:                "mymodel",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`,
	}
	require.NoError(t, entity.Insert(ctx, db, &entityModel))

	// Template containing a concrete job with a matrix strategy
	entityTmpl := sdk.Entity{
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
    deploy:
      runs-on: .cds/worker-models/mymodel.yml
      strategy:
        matrix:
          region: [region1, region2]
      steps:
      - run: echo "Deploy ${{ matrix.region }}"`,
	}
	require.NoError(t, entity.Insert(ctx, db, &entityTmpl))

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

	// First trigger: the templated job is expanded into the workflow definition, no run job yet
	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	wrAfter1, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	deployJob, has := wrAfter1.WorkflowData.Workflow.Jobs["deploy"]
	require.True(t, has)

	// Simulate a permutation that already ran
	now := time.Now()
	rjDone := sdk.V2WorkflowRunJob{
		JobID:         "deploy",
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		Queued:        time.Now(),
		Scheduled:     &now,
		Started:       &now,
		Ended:         &now,
		Job:           deployJob,
		Matrix:        map[string]string{"region": "region1"},
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &rjDone))

	// Second trigger: only the missing permutation must be enqueued
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

	rjs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 2, len(rjs))
	for _, rj := range rjs {
		require.Equal(t, "deploy", rj.JobID)
		switch rj.Matrix["region"] {
		case "region1":
			require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, rj.Status)
		case "region2":
			require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, rj.Status)
		default:
			t.Fatalf("unexpected matrix permutation %v", rj.Matrix)
		}
	}
}
