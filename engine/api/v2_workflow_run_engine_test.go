package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestJobConditionSuccess(t *testing.T) {
	_, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)

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
			currentJobContext := buildContextForJob(context.TODO(), run.WorkflowData.Workflow, jobsContext, sdk.WorkflowRunContext{}, nil, "job4")
			initiator := sdk.V2Initiator{
				UserID:         admin.ID,
				User:           admin.Initiator(),
				IsAdminWithMFA: true,
			}
			b, err := checkCanRunJob(context.TODO(), db, run, nil, jobDef, currentJobContext, initiator)
			require.NoError(t, err)
			require.Equal(t, tt.result, b)
		})
	}
}

func TestJobConditionReviewers(t *testing.T) {
	_, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	lambda, _ := assets.InsertLambdaUser(t, db)

	jobsContext := sdk.JobsResultContext{}

	run := sdk.V2WorkflowRun{
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Gates: map[string]sdk.V2JobGate{
					"gate1": {
						Reviewers: sdk.V2JobGateReviewers{
							Users: []string{lambda.Username},
						},
					},
				},
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Gate: "gate1",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		u              sdk.AuthentifiedUser
		isAdminWithMFA bool
		result         bool
	}{
		{
			name:           "Test reviewers user match",
			u:              *lambda,
			isAdminWithMFA: false,
			result:         true,
		},
		{
			name:           "Test reviewers user not match",
			u:              *admin,
			isAdminWithMFA: false,
			result:         false,
		},
		{
			name:           "Test reviewers user not match but admin with mfa",
			u:              *admin,
			isAdminWithMFA: true,
			result:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobDef := run.WorkflowData.Workflow.Jobs["job1"]
			currentJobContext := buildContextForJob(context.TODO(), run.WorkflowData.Workflow, jobsContext, sdk.WorkflowRunContext{}, nil, "job1")
			initiator := sdk.V2Initiator{
				UserID:         tt.u.ID,
				User:           tt.u.Initiator(),
				IsAdminWithMFA: tt.isAdminWithMFA,
			}
			b, err := checkCanRunJob(context.TODO(), db, run, nil, jobDef, currentJobContext, initiator)
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
	wf := sdk.V2Workflow{Jobs: allJobs}

	currentJobContext := sdk.JobsResultContext{}
	buildAncestorJobContext(context.TODO(), "job6", wf, jobsContext, nil, currentJobContext)

	require.Equal(t, 3, len(currentJobContext))
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, currentJobContext["job1"].Result)
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, currentJobContext["job5"].Result)
}

func TestBuildCurrentJobContextWithStages(t *testing.T) {
	wf := sdk.V2Workflow{
		Stages: map[string]sdk.WorkflowStage{
			"stage1": {
				Needs: []string{},
			},
			"stage2": {
				Needs: []string{"stage1"},
			},
			"stage3": {
				Needs: []string{},
			},
		},
		Jobs: map[string]sdk.V2Job{
			"job1": {
				ContinueOnError: true,
				Stage:           "stage1",
			},
			"job2": {
				Stage: "stage1",
			},
			"job3": {
				Needs: []string{"job1"},
				Stage: "stage1",
			},
			"job4": {
				Needs: []string{"job1"},
				Stage: "stage1",
			},
			"job5": {
				Needs: []string{"job3"},
				Stage: "stage1",
			},
			"job6": {
				Stage: "stage2",
			},
			"job7": {
				Stage: "stage3",
			},
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
		"job7": {
			Result: sdk.V2WorkflowRunJobStatusFail,
		},
	}

	run := sdk.V2WorkflowRun{
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: wf,
		},
	}
	stages := run.GetStages()
	if len(stages) > 0 {
		for k, j := range jobsContext {
			stageName := run.WorkflowData.Workflow.Jobs[k].Stage
			jobInStage := stages[stageName].Jobs[k]
			jobInStage.Status = j.Result
			stages[stageName].Jobs[k] = jobInStage
		}
		stages.ComputeStatus()
	}

	currentJobContext := sdk.JobsResultContext{}
	buildAncestorJobContext(context.TODO(), "job6", wf, jobsContext, stages, currentJobContext)

	require.Equal(t, 5, len(currentJobContext))
	_, has := currentJobContext["job7"]
	require.False(t, has)

	fullContext := buildContextForJob(context.TODO(), run.WorkflowData.Workflow, currentJobContext, sdk.WorkflowRunContext{}, stages, "job6")
	require.Equal(t, 3, len(fullContext.Needs))
	_, has = fullContext.Needs["job5"]
	require.True(t, has)
	_, has = fullContext.Needs["job4"]
	require.True(t, has)
	_, has = fullContext.Needs["job2"]
	require.True(t, has)
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

	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       "var1",
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

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

		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					If: fmt.Sprintf("${{ cds.workflow ==< && '%s' }}", wkfName),
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
	t.Logf("%+v", runInfos)
	require.Equal(t, 1, len(runInfos))
	require.Contains(t, runInfos[0].Message, "mismatched input")
	require.Contains(t, runInfos[0].Message, "into a boolean")

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	require.Equal(t, 0, len(runjobs))

	runDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusFail, runDB.Status)
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusBuilding,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	wrj11 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr2))

	wrj2 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr2.ID,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj2))

	wrj3 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr2.ID,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
				Role:            sdk.RegionRoleExecute,
				RBACProjectKeys: []string{proj.Key},
				RegionID:        reg.ID,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

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
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.Background(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
				Role:            sdk.RegionRoleExecute,
				RBACProjectKeys: []string{proj.Key},
				RegionID:        reg.ID,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

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
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"foo": []string{"foo1", "foo2"},
						},
					},
				},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		Initiator: *wr.Initiator,
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
		Initiator: *wr.Initiator,
	}
	err = workflow_v2.InsertRunJob(context.TODO(), db, &wrjFoo2)
	t.Logf("%+v", err)
	require.NoError(t, err)

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
						Matrix: map[string]interface{}{
							"foo": []string{"foo1", "foo2"},
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
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		Initiator: *wr.Initiator,
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
		Initiator: *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjFoo2))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

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
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj1))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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

	lambdaUser, _ := assets.InsertLambdaUser(t, db)

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

	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       "var1",
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

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
		Initiator: &sdk.V2Initiator{
			UserID: lambdaUser.ID,
			User:   lambdaUser.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: lambdaUser.ID,
			User:   lambdaUser.Initiator(),
		},
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

func TestWorkflowIntegrationInterpoloated(t *testing.T) {
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

	reg2 := sdk.Region{
		Name: "myregion",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg2))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	model := sdk.IntegrationModel{Name: sdk.RandomString(10), Event: true, DefaultConfig: sdk.IntegrationConfig{
		"myparam": {
			Value: "myregion",
			Type:  sdk.IntegrationConfigTypeRegion,
		},
	}}
	require.NoError(t, integration.InsertModel(db, &model))
	projInt := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"myparam": model.DefaultConfig["myparam"],
		},
		Name:               "myinteg-eu",
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &projInt))

	vs := sdk.ProjectVariableSet{
		Name:       "myvar",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	vsItem := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 "item",
		Type:                 sdk.ProjectVariableTypeString,
		Value:                `{"region": "eu", "token": "mytoken"}`,
	}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &vsItem))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
		Regions: []sdk.RBACRegion{
			{
				RegionID:            reg.ID,
				AllUsers:            true,
				RBACOrganizationIDs: []string{org.ID},
				Role:                sdk.RegionRoleExecute,
			},
			{
				RegionID:            reg2.ID,
				AllUsers:            true,
				RBACOrganizationIDs: []string{org.ID},
				Role:                sdk.RegionRoleExecute,
			},
		},
		RegionProjects: []sdk.RBACRegionProject{
			{
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
			{
				RegionID:        reg2.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

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
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{
						{
							ID: "1",
						},
					},
					Integrations: []string{"myinteg-${{vars.myvar.item.region}}"},
					VariableSets: []string{"myvar"},
				},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)

	jobInfos, err := workflow_v2.LoadRunJobInfosByRunJobID(context.TODO(), db, runjobs[0].ID)
	t.Logf("%+v", jobInfos)
	require.NoError(t, err)
	require.Equal(t, 0, len(runInfos))

	t.Logf("%+v", runjobs[0])
	require.Equal(t, []string{"myinteg-eu"}, runjobs[0].Job.Integrations)
	require.Equal(t, "myregion", runjobs[0].Job.Region)
	require.Equal(t, "myregion", runjobs[0].Region)

}

func TestCreateJobsFromTemplatedMatrix(t *testing.T) {
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
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *admin)

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	tmplRaw := `name: jobtmpl
parameters:
- key: region
spec: |-
  jobs:
    deploy_[[.params.region]]:
      runs-on: .cds/worker-models/mymodel.yml
      steps:
      - run: echo "Deploy"
    smoke_[[.params.region]]:
      needs:
      - deploy_[[.params.region]]
      runs-on: .cds/worker-models/mymodel.yml
      steps:
      - run: echo "SmokeTest"`
	entityTmpl := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflowTemplate,
		Name:                "jobTmpl",
		FilePath:            ".cds/workflow-templates/jobTemplate.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                tmplRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityTmpl))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{},
				},
				"job2": {
					Needs: []string{"job1"},
					From:  ".cds/workflow-templates/jobTemplate.yml",
					Parameters: map[string]string{
						"region": "${{matrix.region}}",
					},
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
				},
				"job3": {
					Steps: []sdk.ActionStep{},
					Needs: []string{"job2"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	now := time.Now()
	job1RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job1",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSuccess,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job1"],
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job1RunJob))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        sdk.V2Initiator{UserID: admin.ID},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 6, len(wrDB.WorkflowData.Workflow.Jobs))

	// region 1
	deploy1, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region1"]
	require.True(t, has)
	require.Equal(t, []string{"job1"}, deploy1.Needs)

	smoke1, has := wrDB.WorkflowData.Workflow.Jobs["smoke_region1"]
	require.True(t, has)
	require.Equal(t, []string{"deploy_region1"}, smoke1.Needs)

	deploy2, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region2"]
	require.True(t, has)
	require.Equal(t, []string{"job1"}, deploy2.Needs)

	smoke2, has := wrDB.WorkflowData.Workflow.Jobs["smoke_region2"]
	require.True(t, has)
	require.Equal(t, []string{"deploy_region2"}, smoke2.Needs)

	_, has = wrDB.WorkflowData.Workflow.Jobs["job2"]
	require.False(t, has)

	job3 := wrDB.WorkflowData.Workflow.Jobs["job3"]
	require.Equal(t, 2, len(job3.Needs))
	var sm1, sm2 bool
	for _, v := range job3.Needs {
		if v == "smoke_region1" {
			sm1 = true
		}
		if v == "smoke_region2" {
			sm2 = true
		}
	}
	require.True(t, sm1)
	require.True(t, sm2)
}

func TestCreateJobsFromTemplatedMatrix_WithStage(t *testing.T) {
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
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *admin)

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	tmplRaw := `name: jobtmpl
parameters:
- key: region
- key: previousStage
spec: |-
  stages:
    [[.params.region]]:
      needs:
      - [[.params.previousStage]]
  jobs:
    deploy_[[.params.region]]:
      runs-on: .cds/worker-models/mymodel.yml
      steps:
      - run: echo "Deploy"
      stage: [[.params.region]]
    smoke_[[.params.region]]:
      stage: [[.params.region]]
      needs:
      - deploy_[[.params.region]]
      runs-on: .cds/worker-models/mymodel.yml
      steps:
      - run: echo "SmokeTest"`
	entityTmpl := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflowTemplate,
		Name:                "jobTmpl",
		FilePath:            ".cds/workflow-templates/jobTemplate.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                tmplRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityTmpl))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Stages: map[string]sdk.WorkflowStage{
				"build": {
					Needs: []string{},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{},
					Stage: "build",
				},
				"job2": {
					Needs: []string{"job1"},
					From:  ".cds/workflow-templates/jobTemplate.yml",
					Parameters: map[string]string{
						"region":        "${{matrix.region}}",
						"previousStage": "build",
					},
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
					Stage: "build",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	now := time.Now()
	job1RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job1",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSuccess,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job1"],
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job1RunJob))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        sdk.V2Initiator{UserID: admin.ID},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 5, len(wrDB.WorkflowData.Workflow.Jobs))

	// region 1
	deploy1, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region1"]
	require.True(t, has)
	require.Equal(t, 0, len(deploy1.Needs))
	require.Equal(t, "region1", deploy1.Stage)

	smoke1, has := wrDB.WorkflowData.Workflow.Jobs["smoke_region1"]
	require.True(t, has)
	require.Equal(t, []string{"deploy_region1"}, smoke1.Needs)

	deploy2, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region2"]
	require.True(t, has)
	require.Equal(t, 0, len(deploy2.Needs))
	require.Equal(t, "region2", deploy2.Stage)

	smoke2, has := wrDB.WorkflowData.Workflow.Jobs["smoke_region2"]
	require.True(t, has)
	require.Equal(t, []string{"deploy_region2"}, smoke2.Needs)

	_, has = wrDB.WorkflowData.Workflow.Jobs["job2"]
	require.False(t, has)

	stageRegion1, has := wrDB.WorkflowData.Workflow.Stages["region1"]
	require.True(t, has)
	require.Equal(t, 1, len(stageRegion1.Needs))
	require.Equal(t, "build", stageRegion1.Needs[0])

	stageRegion2, has := wrDB.WorkflowData.Workflow.Stages["region2"]
	require.True(t, has)
	require.Equal(t, 1, len(stageRegion2.Needs))
	require.Equal(t, "build", stageRegion2.Needs[0])

	require.Equal(t, 3, len(wrDB.WorkflowData.Workflow.Stages))
}

func TestCreateJobsWithMatrix(t *testing.T) {
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

	model := sdk.IntegrationModel{Name: sdk.RandomString(10), Event: true, DefaultConfig: sdk.IntegrationConfig{
		"myparam": {
			Value: "myregion",
			Type:  sdk.IntegrationConfigTypeRegion,
		},
	}}
	require.NoError(t, integration.InsertModel(db, &model))
	integRegion1 := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"myparam": sdk.IntegrationConfigValue{
				Value: "region1",
				Type:  sdk.IntegrationConfigTypeRegion,
			},
		},
		Name:               "integ-region1",
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &integRegion1))

	integRegion2 := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"myparam": sdk.IntegrationConfigValue{
				Value: "region2",
				Type:  sdk.IntegrationConfigTypeRegion,
			},
		},
		Name:               "integ-region2",
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &integRegion2))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	vs := sdk.ProjectVariableSet{
		Name:       "region",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	items := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 "regions",
		Type:                 "string",
		Value:                "[\"region1\",\"region2\"]",
	}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &items))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": "${{ vars.region.regions }}",
						},
					},
					Integrations: []string{"${{ format('integ-{0}', matrix.region) }}"},
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
					VariableSets: []string{"region"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 1, len(wrDB.WorkflowData.Workflow.Jobs))

	t.Logf("%+v", wrDB.WorkflowData.Workflow.Jobs)

	jobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)

	reg1Found := false
	reg2Found := false
	for _, j := range jobs {
		if j.Region == "region1" {
			reg1Found = true
			require.Equal(t, []string{"integ-region1"}, j.Job.Integrations)
		}
		if j.Region == "region2" {
			reg2Found = true
			require.Equal(t, []string{"integ-region2"}, j.Job.Integrations)
		}
	}
	require.True(t, reg1Found)
	require.True(t, reg2Found)
}

func TestRestartMatrixJob(t *testing.T) {
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
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *admin)
	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	tmplRaw := `name: jobtmpl
parameters:
- key: region
spec: |-
  jobs:
    deploy_[[.params.region]]:
      runs-on: .cds/worker-models/mymodel.yml
      steps:
      - run: echo "Deploy"
    smoke_[[.params.region]]:
      needs:
      - deploy_[[.params.region]]
      runs-on: .cds/worker-models/mymodel.yml
      steps:
      - run: echo "SmokeTest"`
	entityTmpl := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflowTemplate,
		Name:                "jobTmpl",
		FilePath:            ".cds/workflow-templates/jobTemplate.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                tmplRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityTmpl))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{},
				},
				"job2": {
					Needs: []string{"job1"},
					From:  ".cds/workflow-templates/jobTemplate.yml",
					Parameters: map[string]string{
						"region": "${{matrix.region}}",
					},
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
				},
				"job3": {
					Steps: []sdk.ActionStep{},
					Needs: []string{"job2"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	now := time.Now()
	job1RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job1",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSuccess,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job1"],
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job1RunJob))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		Initiator:        sdk.V2Initiator{UserID: admin.ID},
		DeprecatedUserID: admin.ID,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 6, len(wrDB.WorkflowData.Workflow.Jobs))

	// region 1
	deploy1, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region1"]
	require.True(t, has)
	require.Equal(t, []string{"job1"}, deploy1.Needs)

	smoke1, has := wrDB.WorkflowData.Workflow.Jobs["smoke_region1"]
	require.True(t, has)
	require.Equal(t, []string{"deploy_region1"}, smoke1.Needs)

	deploy2, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region2"]
	require.True(t, has)
	require.Equal(t, []string{"job1"}, deploy2.Needs)

	smoke2, has := wrDB.WorkflowData.Workflow.Jobs["smoke_region2"]
	require.True(t, has)
	require.Equal(t, []string{"deploy_region2"}, smoke2.Needs)

	_, has = wrDB.WorkflowData.Workflow.Jobs["job2"]
	require.False(t, has)

	job3 := wrDB.WorkflowData.Workflow.Jobs["job3"]
	require.Equal(t, 2, len(job3.Needs))
	var sm1, sm2 bool
	for _, v := range job3.Needs {
		if v == "smoke_region1" {
			sm1 = true
		}
		if v == "smoke_region2" {
			sm2 = true
		}
	}
	require.True(t, sm1)
	require.True(t, sm2)
}

func TestRestartMatrixRunJob(t *testing.T) {
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
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{},
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
				},
				"job2": {
					Needs: []string{"job1"},
					Steps: []sdk.ActionStep{},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	now := time.Now()
	job1RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job1",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSuccess,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job1"],
		DeprecatedUserID: admin.ID,
		Matrix: sdk.JobMatrix{
			"region": "region1",
		},
		Initiator: *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job1RunJob))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	rjs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	require.Len(t, rjs, 2)
	for _, rjj := range rjs {
		require.Equal(t, "job1", rjj.JobID)
	}
}

func TestCreateJobsFromTemplatedMatrix_WithIntegrationInTemplate(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	admin, _ := assets.InsertAdminUser(t, db)

	org, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	require.NoError(t, err)

	reg := sdk.Region{
		Name: "region1",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	reg2 := sdk.Region{
		Name: "region2",
	}
	require.NoError(t, region.Insert(context.TODO(), db, &reg2))

	api.Config.Workflow.JobDefaultRegion = reg.Name

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *admin)

	model := sdk.IntegrationModel{Name: sdk.RandomString(10), Event: true, DefaultConfig: sdk.IntegrationConfig{
		"myparam": {
			Value: "myregion",
			Type:  sdk.IntegrationConfigTypeRegion,
		},
	}}
	require.NoError(t, integration.InsertModel(db, &model))
	integRegion1 := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"myparam": sdk.IntegrationConfigValue{
				Value: "region1",
				Type:  sdk.IntegrationConfigTypeRegion,
			},
		},
		Name:               "integ-region1",
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &integRegion1))

	integRegion2 := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"myparam": sdk.IntegrationConfigValue{
				Value: "region2",
				Type:  sdk.IntegrationConfigTypeRegion,
			},
		},
		Name:               "integ-region2",
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &integRegion2))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	// Create hatchery2
	hatch2 := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch2))

	perm2 := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg2.ID,
				HatcheryID: hatch2.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm2))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	tmplRaw := `name: jobtmpl
parameters:
- key: region
spec: |-
  jobs:
    deploy_[[.params.region]]:
      runs-on: .cds/worker-models/mymodel.yml
      integrations:
      - integ-[[.params.region]]
      steps:
      - run: echo "Deploy"`
	entityTmpl := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflowTemplate,
		Name:                "jobTmpl",
		FilePath:            ".cds/workflow-templates/jobTemplate.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                tmplRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityTmpl))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{},
				},
				"job2": {
					Needs: []string{"job1"},
					From:  ".cds/workflow-templates/jobTemplate.yml",
					Parameters: map[string]string{
						"region": "${{matrix.region}}",
					},
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	now := time.Now()
	job1RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job1",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSuccess,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job1"],
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job1RunJob))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        sdk.V2Initiator{UserID: admin.ID},
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 3, len(wrDB.WorkflowData.Workflow.Jobs))

	jRegion1, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region1"]
	require.True(t, has)
	jRegion2, has := wrDB.WorkflowData.Workflow.Jobs["deploy_region2"]
	require.True(t, has)

	t.Logf("[%s] %v", jRegion1.Region, jRegion1.Integrations)
	t.Logf("[%s] %v", jRegion2.Region, jRegion2.Integrations)

	require.Equal(t, "region1", jRegion1.Region)
	require.Equal(t, []string{"integ-region1"}, jRegion1.Integrations)

	require.Equal(t, "region2", jRegion2.Region)
	require.Equal(t, []string{"integ-region2"}, jRegion2.Integrations)

}

func TestGateWIthNoJobIf(t *testing.T) {
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
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{!failure()}}",
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
				},
				"job2": {
					Needs: []string{},
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
				"job3": {
					Gate:  "preprod",
					Needs: []string{"job1", "job2"},
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	now := time.Now()
	job1RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job1",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSuccess,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job1"],
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
		Matrix: sdk.JobMatrix{
			"region": "region1",
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job1RunJob))

	job2RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job1",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSuccess,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job1"],
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
		Matrix: sdk.JobMatrix{
			"region": "region2",
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job2RunJob))

	job3RunJob := sdk.V2WorkflowRunJob{
		JobID:            "job2",
		WorkflowRunID:    wr.ID,
		ProjectKey:       proj.Key,
		WorkflowName:     wr.WorkflowName,
		RunNumber:        wr.RunNumber,
		RunAttempt:       wr.RunAttempt,
		Status:           sdk.V2WorkflowRunJobStatusSkipped,
		Queued:           time.Now(),
		Scheduled:        &now,
		Started:          &now,
		Ended:            &now,
		Job:              wr.WorkflowData.Workflow.Jobs["job2"],
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &job3RunJob))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, runInfos, 0)

	rjs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	require.Len(t, rjs, 4)
	job3Found := false
	for _, rj := range rjs {
		if rj.JobID != "job3" {
			continue
		}
		require.Equal(t, rj.Status, sdk.V2WorkflowRunJobStatusWaiting)
		job3Found = true
		break
	}
	require.True(t, job3Found)

}

func TestCreateJobsWithMatrix_WithProjectConcurrency(t *testing.T) {
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

	pc := sdk.ProjectConcurrency{
		ProjectKey:       proj.Key,
		Name:             "mycc",
		Pool:             1,
		Order:            sdk.ConcurrencyOrderOldestFirst,
		CancelInProgress: false,
	}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pc))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	vs := sdk.ProjectVariableSet{
		Name:       "region",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	items := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 "regions",
		Type:                 "string",
		Value:                "[\"region1\",\"region2\"]",
	}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &items))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Concurrency: pc.Name,
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 1, len(wrDB.WorkflowData.Workflow.Jobs))

	t.Logf("%+v", wrDB.WorkflowData.Workflow.Jobs)

	jobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)

	waiting := false
	blocked := false
	for _, j := range jobs {
		if j.Status == sdk.V2WorkflowRunJobStatusBlocked {
			blocked = true
		}
		if j.Status == sdk.V2WorkflowRunJobStatusWaiting {
			waiting = true
		}

	}
	require.Equal(t, 2, len(jobs))
	require.True(t, blocked)
	require.True(t, waiting)
}

func TestConcurrencyCancelInProgress(t *testing.T) {
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

	pc := sdk.ProjectConcurrency{
		ProjectKey:       proj.Key,
		Name:             "mycc",
		Pool:             1,
		Order:            sdk.ConcurrencyOrderOldestFirst,
		CancelInProgress: true,
	}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pc))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	vs := sdk.ProjectVariableSet{
		Name:       "region",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	items := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 "regions",
		Type:                 "string",
		Value:                "[\"region1\",\"region2\"]",
	}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &items))

	wr1 := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Concurrency: pc.Name,
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr1))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr1.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr1.Initiator,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr1.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr1.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 1, len(wrDB.WorkflowData.Workflow.Jobs))

	jobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)

	waiting := false
	cancelled := false
	for _, j := range jobs {
		if j.Status == sdk.V2WorkflowRunJobStatusCancelled {
			cancelled = true
		}
		if j.Status == sdk.V2WorkflowRunJobStatusWaiting {
			waiting = true
		}

	}
	require.Equal(t, 2, len(jobs))
	require.True(t, cancelled)
	require.True(t, waiting)

	// Running a 2 workflow and check cancelled
	wr2 := wr1
	wr2.ID = ""
	wr2.RunNumber += 1
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr2))
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr2.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr2.Initiator,
	}))

	runInfos2, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr2.ID)
	require.NoError(t, err)
	for _, info := range runInfos2 {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB2, err := workflow_v2.LoadRunByID(context.TODO(), db, wr2.ID)
	require.NoError(t, err)

	jobs2, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB2.ID, wrDB2.RunAttempt)
	require.NoError(t, err)

	waiting2 := false
	cancelled2 := false
	for _, j := range jobs2 {
		if j.Status == sdk.V2WorkflowRunJobStatusCancelled {
			cancelled2 = true
		}
		if j.Status == sdk.V2WorkflowRunJobStatusWaiting {
			waiting2 = true
		}

	}
	require.Equal(t, 2, len(jobs2))
	require.True(t, cancelled2)
	require.True(t, waiting2)

	// Reload wr1 jobs
	jobs1Reloaded, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)
	for _, j := range jobs1Reloaded {
		require.Equal(t, sdk.V2WorkflowRunJobStatusCancelled, j.Status)
	}

	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/workflow/outgoing", gomock.Any(), gomock.Any())

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr1.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr1.Initiator,
	}))
	// reload wr1 to check cancelled status
	wr1DBUpdated, err := workflow_v2.LoadRunByID(context.TODO(), db, wrDB.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusCancelled, wr1DBUpdated.Status)
}

func TestConcurrencyCancelInProgress_BlockedJobs(t *testing.T) {
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

	pc := sdk.ProjectConcurrency{
		ProjectKey:       proj.Key,
		Name:             "mycc",
		Pool:             2,
		Order:            sdk.ConcurrencyOrderOldestFirst,
		CancelInProgress: true,
	}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pc))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	vs := sdk.ProjectVariableSet{
		Name:       "region",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	items := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 "regions",
		Type:                 "string",
		Value:                "[\"region1\",\"region2\"]",
	}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &items))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Concurrency: pc.Name,
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
				"job2": {
					Concurrency: pc.Name,
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
				"job3": {
					Concurrency: pc.Name,
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	jobRun1 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         "job1",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now(),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRun2 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         "job2",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRun3 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         "job3",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(2 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRun1))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRun2))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRun3))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	jobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)

	var job1, job2, job3 bool

	for _, j := range jobs {
		t.Logf("%s >>> %s", j.JobID, j.Status)
		if j.ID == jobRun1.ID {
			if j.Status == sdk.V2WorkflowRunJobStatusCancelled {
				job1 = true
			}
		}

		if j.ID == jobRun2.ID {
			if j.Status == sdk.V2WorkflowRunJobStatusWaiting {
				job2 = true
			}
		}

		if j.ID == jobRun3.ID {
			if j.Status == sdk.V2WorkflowRunJobStatusWaiting {
				job3 = true
			}
		}

	}
	require.True(t, job1)
	require.True(t, job2)
	require.True(t, job3)
}

func TestCreateJobsWithMatrix_WithProjectConcurrencyWithCondition(t *testing.T) {
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

	pc := sdk.ProjectConcurrency{
		ProjectKey:       proj.Key,
		Name:             "mycc",
		Pool:             1,
		Order:            sdk.ConcurrencyOrderOldestFirst,
		If:               "git.ref_name == 'toto'",
		CancelInProgress: false,
	}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pc))

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		VariableSets: []sdk.RBACVariableSet{
			{
				AllUsers:        true,
				Role:            sdk.VariableSetRoleUse,
				AllVariableSets: true,
				ProjectKey:      proj.Key,
			},
		},
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
				RegionID:        reg.ID,
				RBACProjectKeys: []string{proj.Key},
				Role:            sdk.RegionRoleExecute,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: "docker"}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &perm))

	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	modelRaw := `name: mymodel
type: docker
osarch: linux-amd64
spec:
  image: debian:12`
	entityModel := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		Name:                "mymodel",
		FilePath:            ".cds/worker-models/mymodel.yml",
		Commit:              "abcdef",
		Ref:                 "refs/heads/master",
		Data:                modelRaw,
		UserID:              &admin.ID,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &entityModel))

	vs := sdk.ProjectVariableSet{
		Name:       "region",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	items := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 "regions",
		Type:                 "string",
		Value:                "[\"region1\",\"region2\"]",
	}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &items))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "abcdef",
		WorkflowRef:        "refs/heads/master",
		RunAttempt:         1,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: sdk.RandomString(10),
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Concurrency: pc.Name,
					Strategy: &sdk.V2JobStrategy{
						Matrix: map[string]interface{}{
							"region": []string{"region1", "region2"},
						},
					},
					Steps: []sdk.ActionStep{
						{
							Run: "echo toto",
						},
					},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:            wr.ID,
		DeprecatedUserID: admin.ID,
		Initiator:        *wr.Initiator,
	}))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	for _, info := range runInfos {
		t.Logf("%+v", info)
	}
	require.Equal(t, 0, len(runInfos))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	// Total of jobs
	require.Equal(t, 1, len(wrDB.WorkflowData.Workflow.Jobs))

	t.Logf("%+v", wrDB.WorkflowData.Workflow.Jobs)

	jobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)

	for _, j := range jobs {
		require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, j.Status)

		infos, err := workflow_v2.LoadRunJobInfosByRunJobID(context.TODO(), db, j.ID)
		require.NoError(t, err)
		t.Logf("%+v", infos)
		require.Equal(t, 1, len(infos))
		require.Equal(t, "Concurrency \"mycc\" skipped", infos[0].Message)
	}
	require.Equal(t, 2, len(jobs))
}
