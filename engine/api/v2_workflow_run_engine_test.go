package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestJobConditionSuccess(t *testing.T) {
	jobsContext := sdk.JobsResultContext{
		"job1": {
			Result: sdk.V2WorkflowRunJobStatusFail,
		},
		"job2": {
			Result: sdk.V2WorkflowRunJobStatusSuccess,
		},
		"job3": {
			Result: sdk.V2WorkflowRunJobStatusFail,
		},
	}
	allJobs := map[string]sdk.V2Job{
		"job1": {
			ContinueOnError: true,
		},
		"job2": {},
		"job3": {},
		"job4": {},
	}

	run := sdk.V2WorkflowRun{
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Jobs: allJobs,
			},
		},
	}

	tests := []struct {
		name      string
		condition string
		needs     []string
		result    bool
	}{
		{
			name:      "Test success()",
			condition: "success()",
			needs:     []string{"job1", "job2"},
			result:    true,
		},
		{
			name:      "Test success() with 1 fail",
			condition: "success()",
			needs:     []string{"job1", "job2", "job3"},
			result:    false,
		},
		{
			name:      "Test failure()",
			condition: "failure()",
			needs:     []string{"job3"},
			result:    true,
		},
		{
			name:      "Test always()",
			condition: "always()",
			needs:     []string{"job3"},
			result:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run.WorkflowData.Workflow.Jobs["job4"] = sdk.V2Job{
				If:    tt.condition,
				Needs: tt.needs,
			}
			jobDef := run.WorkflowData.Workflow.Jobs["job4"]
			currentJobContext := buildContextForJob(context.TODO(), run.WorkflowData.Workflow.Jobs, jobsContext, sdk.WorkflowRunContext{}, "job4")
			b, err := checkJobCondition(context.TODO(), run, "job4", &jobDef, currentJobContext)
			require.NoError(t, err)
			require.Equal(t, tt.result, b)
		})
	}

}

func TestBuildCurrentJobContext(t *testing.T) {
	allJobs := map[string]sdk.V2Job{
		"job1": {
			ContinueOnError: true,
		},
		"job2": {},
		"job3": {
			Needs: []string{"job1"},
		},
		"job4": {
			Needs: []string{"job1"},
		},
		"job5": {
			Needs: []string{"job3"},
		},
		"job6": {
			Needs: []string{"job5"},
		},
	}

	jobsContext := sdk.JobsResultContext{
		"job1": {
			Result: sdk.V2WorkflowRunJobStatusFail,
		},
		"job2": {
			Result: sdk.V2WorkflowRunJobStatusSuccess,
		},
		"job3": {
			Result: sdk.V2WorkflowRunJobStatusSuccess,
		},
		"job4": {
			Result: sdk.V2WorkflowRunJobStatusFail,
		},
		"job5": {
			Result: sdk.V2WorkflowRunJobStatusFail,
		},
	}

	currentJobContext := sdk.JobsResultContext{}
	buildAncestorJobContext("job6", allJobs, jobsContext, currentJobContext)

	require.Equal(t, 3, len(currentJobContext))
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, currentJobContext["job1"].Result)
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, currentJobContext["job5"].Result)
}

func TestGenerateMatrix(t *testing.T) {
	matrix := map[string][]string{
		"foo": {"foo1", "foo2"},
		"bar": {"bar1", "bar2"},
	}
	all := make([]map[string]string, 0)
	generateMatrix(matrix, []string{"foo", "bar"}, 0, map[string]string{}, &all)

	require.Equal(t, 4, len(all))
	foo1bar1 := false
	foo1bar2 := false
	foo2bar1 := false
	foo2bar2 := false
	for _, m := range all {
		if m["foo"] == "foo1" && m["bar"] == "bar1" {
			foo1bar1 = true
		}
		if m["foo"] == "foo1" && m["bar"] == "bar2" {
			foo1bar2 = true
		}
		if m["foo"] == "foo2" && m["bar"] == "bar1" {
			foo2bar1 = true
		}
		if m["foo"] == "foo2" && m["bar"] == "bar2" {
			foo2bar2 = true
		}
	}
	require.True(t, foo1bar1)
	require.True(t, foo1bar2)
	require.True(t, foo2bar1)
	require.True(t, foo2bar2)
}

func TestWorkflowTrigger1Job(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					VariableSets: []string{"var1"},
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
				"job2": {
					VariableSets: []string{"var1"},
					Needs:        []string{"job1"},
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID:          wr.ID,
		UserID:         admin.ID,
		IsAdminWithMFA: true,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 1, len(runjobs))
	require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, runjobs[0].Status)
	require.Equal(t, "job1", runjobs[0].JobID)
}

func TestWorkflowTrigger1JobAdminNoMFA(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					VariableSets: []string{"var1"},
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
				"job2": {
					VariableSets: []string{"var1"},
					Needs:        []string{"job1"},
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(runInfos))
	require.Contains(t, runInfos[0].Message, "does not have enough right on varset var1")
}

func TestWorkflowTriggerWithCondition(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: wkfName,
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		Contexts: sdk.WorkflowRunContext{
			CDS: sdk.CDSContext{
				Workflow: wkfName,
			},
		},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					If: fmt.Sprintf("cds.workflow == '%s'", wkfName),
				},
				"job2": {
					If: fmt.Sprintf("${{ cds.workflow == '%s' }}", wkfName),
				},
			},
		},
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 2, len(runjobs))
}

func TestWorkflowTriggerWithConditionKOSyntax(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: wkfName,
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					If: fmt.Sprintf("${{ cds.workflow ==< && '%s' }}", wkfName),
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.Error(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	t.Logf("%+v", runInfos)
	require.Equal(t, 1, len(runInfos))
	t.Logf(runInfos[0].Message)

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 0, len(runjobs))
}

func TestTriggerBlockedWorkflowRuns(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	db.Exec("DELETE FROM v2_workflow_run_job")
	db.Exec("DELETE FROM v2_workflow_run")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	wrj11 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj11))

	wr2 := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr2))

	wrj2 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr2.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj2))

	wrj3 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr2.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj3))

	wrs, err := workflow_v2.LoadBuildingRunWithEndedJobs(ctx, api.mustDB())
	require.NoError(t, err)
	require.Equal(t, 1, len(wrs))
	require.Equal(t, wr2.ID, wrs[0].ID)

	api.workflowRunTriggerChan = make(chan sdk.V2WorkflowRunEnqueue, 1)
	require.NoError(t, api.triggerBlockedWorkflowRun(ctx, wrs[0]))

	wrChan := <-api.workflowRunTriggerChan
	require.Equal(t, wrChan.RunID, wrs[0].ID)
}

func TestWorkflowTriggerStage(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Stages: map[string]sdk.WorkflowStage{
				"stage1": {},
				"stage2": {
					Needs: []string{"stage1"},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Stage: "stage1",
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
				"job2": {
					Stage: "stage2",
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 1, len(runjobs))
	require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, runjobs[0].Status)
	require.Equal(t, "job1", runjobs[0].JobID)
}

func TestWorkflowStageNeeds(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Stages: map[string]sdk.WorkflowStage{
				"stage1": {},
				"stage2": {
					Needs: []string{"stage1"},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Stage: "stage1",
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
				"job2": {
					Stage: "stage2",
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		JobID:         "job1",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		RunNumber:     wr.RunNumber,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 2, len(runjobs))

	jobs := make(map[string]sdk.V2WorkflowRunJob)
	for _, r := range runjobs {
		jobs[r.JobID] = r
	}

	require.NotEmpty(t, jobs["job2"])
	require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, jobs["job2"].Status)
}

func TestWorkflowMatrixNeeds(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string][]string{
							"foo": {"foo1", "foo2"},
						},
					},
				},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrjFoo1 := sdk.V2WorkflowRunJob{
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		JobID:         "job1",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Matrix: map[string]string{
			"foo": "foo1",
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjFoo1))

	wrjFoo2 := sdk.V2WorkflowRunJob{
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Status:        sdk.V2WorkflowRunJobStatusBuilding,
		JobID:         "job1",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Matrix: map[string]string{
			"foo": "foo2",
		},
	}
	err = workflow_v2.InsertRunJob(context.TODO(), db, &wrjFoo2)
	t.Logf("%+v", err)
	require.NoError(t, err)

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 2, len(runjobs))

	// END Matrix 2 - It must trigger job 2

	wrjFoo2.Status = sdk.V2WorkflowRunJobStatusSuccess
	require.NoError(t, workflow_v2.UpdateJobRun(context.TODO(), db, &wrjFoo2))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runjobs, err = workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 3, len(runjobs))
}

func TestWorkflowStageMatrixNeeds(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Stages: map[string]sdk.WorkflowStage{
				"stage1": {},
				"stage2": {
					Needs: []string{"stage1"},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string][]string{
							"foo": {"foo1", "foo2"},
						},
					},
					Stage: "stage1",
				},
				"job2": {
					Stage: "stage2",
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrjFoo1 := sdk.V2WorkflowRunJob{
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		JobID:         "job1",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Matrix: map[string]string{
			"foo": "foo1",
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjFoo1))

	wrjFoo2 := sdk.V2WorkflowRunJob{
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Status:        sdk.V2WorkflowRunJobStatusBuilding,
		JobID:         "job1",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Matrix: map[string]string{
			"foo": "foo2",
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjFoo2))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 2, len(runjobs))

	// END Matrix 2 - It must trigger job 2

	wrjFoo2.Status = sdk.V2WorkflowRunJobStatusSuccess
	require.NoError(t, workflow_v2.UpdateJobRun(context.TODO(), db, &wrjFoo2))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runjobs, err = workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 3, len(runjobs))
}

func TestWorkflowSkippedJob(t *testing.T) {
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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
				"job2": {
					Needs: []string{"job1"},
					If:    "1 == 2",
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
				"job3": {
					If:    "${{always()}}",
					Needs: []string{"job2"},
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrj1 := sdk.V2WorkflowRunJob{
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		JobID:         "job1",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Matrix: map[string]string{
			"foo": "foo1",
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj1))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(runInfos))
	require.Equal(t, "Job \"job2\": The condition is not satisfied", runInfos[0].Message)

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 2, len(runjobs))

	mapJob := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runjobs {
		mapJob[rj.JobID] = rj
	}

	require.Equal(t, sdk.V2WorkflowRunJobStatusSkipped, mapJob["job2"].Status)

	// Trigger again to process job2
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	runjobs, err = workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	mapJob = make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runjobs {
		mapJob[rj.JobID] = rj
	}

	require.Equal(t, 3, len(runjobs))
	require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, mapJob["job3"].Status)
}

func TestWorkflowTrigger1JobNoPermissionOnVarset(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	lamdaUser, _ := assets.InsertLambdaUser(t, db)

	org, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	require.NoError(t, err)

	reg := sdk.Region{
		Name: "build",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	api.Config.Workflow.JobDefaultRegion = reg.Name

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
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusBuilding,
		UserID:       lamdaUser.ID,
		Username:     lamdaUser.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					VariableSets: []string{"var1"},
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
				"job2": {
					VariableSets: []string{"var1"},
					Needs:        []string{"job1"},
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: lamdaUser.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(runInfos))
	require.Contains(t, runInfos[0].Message, "does not have enough right on varset var1")

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 1, len(runjobs))
	require.Equal(t, sdk.V2WorkflowRunJobStatusSkipped, runjobs[0].Status)
	require.Equal(t, "job1", runjobs[0].JobID)
}
