package purge

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// WorkflowRunsV2 deletes workflow run v2
func TestApplyRunRetentionOnProject_WorkflowWithRetention(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/bulk/item/delete", gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	wkname := sdk.RandomString(10)
	lambdauser, _ := assets.InsertLambdaUser(t, db)
	p := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, p.Key, vcs.ID, "ovh/cds")
	projectRunRetention := sdk.ProjectRunRetention{
		ProjectKey: p.Key,
		Retentions: sdk.Retentions{
			DefaultRetention: sdk.RetentionRule{
				DurationInDays: 10,
				Count:          10,
			},
			WorkflowRetentions: []sdk.WorkflowRetentions{
				{
					Workflow: "github/**/*",
					Rules: []sdk.WorkflowRetentionRule{
						{
							GitRef: "refs/heads/master",
							RetentionRule: sdk.RetentionRule{
								DurationInDays: 365,
								Count:          3,
							},
						}, {
							GitRef: "refs/heads/dev/**/*",
							RetentionRule: sdk.RetentionRule{
								DurationInDays: 3,
								Count:          3,
							},
						},
					},
					DefaultRetention: &sdk.RetentionRule{
						DurationInDays: 10,
						Count:          1,
					},
				},
			},
		},
	}
	require.NoError(t, project.InsertRunRetention(ctx, db, &projectRunRetention))

	runNumber := 1
	wr := sdk.V2WorkflowRun{
		ProjectKey:   p.Key,
		VCSServerID:  vcs.ID,
		VCSServer:    vcs.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: wkname,
		WorkflowSha:  "123456",
		Status:       sdk.V2WorkflowRunStatusFail,
		RunAttempt:   0,
		Started:      time.Now(),
		LastModified: time.Now(),
		Initiator: &sdk.V2Initiator{
			UserID: lambdauser.ID,
		},
		RunNumber:   0,
		WorkflowRef: "",
	}
	// Create run on master - We must keep only run_number 19 and 20
	for i := 0; i < 20; i++ {
		wr.RunNumber = int64(runNumber)
		runNumber++
		wr.WorkflowRef = "refs/heads/master"

		require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

		if i < 18 {
			wr.Started = time.Now().Add(-400 * 24 * time.Hour)
			require.NoError(t, workflow_v2.UpdateRun(ctx, db, &wr))
		}
		t.Logf("Inset Master %s", wr.Started)
	}

	// Create run on dev branch. We must keep only run_number 38 - 39 - 40
	for i := 0; i < 20; i++ {
		wr.RunNumber = int64(runNumber)
		runNumber++
		wr.WorkflowRef = "refs/heads/dev/my/feat"
		require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))
	}

	// Create run on fix branch. We must keep ony run_number 60
	for i := 0; i < 20; i++ {
		wr.RunNumber = int64(runNumber)
		runNumber++
		wr.WorkflowRef = "refs/heads/fix"
		require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))
	}

	require.NoError(t, ApplyRunRetentionOnProject(ctx, db.DbMap, cache, p.Key))

	wrDB, err := workflow_v2.LoadRuns(ctx, db, p.Key, vcs.ID, repo.ID, wkname)
	require.NoError(t, err)

	require.Equal(t, 6, len(wrDB)) // 19 20 /  38 39 40  / 60
	for _, r := range wrDB {
		t.Logf("%s - %d", r.WorkflowRef, r.RunNumber)
	}
	require.Equal(t, int64(60), wrDB[0].RunNumber)
	require.Equal(t, int64(40), wrDB[1].RunNumber)
	require.Equal(t, int64(39), wrDB[2].RunNumber)
	require.Equal(t, int64(38), wrDB[3].RunNumber)
	require.Equal(t, int64(20), wrDB[4].RunNumber)
	require.Equal(t, int64(19), wrDB[5].RunNumber)
}

func TestApplyRunRetentionOnProject_FallbackProjectDefaultRule(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/bulk/item/delete", gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	wkname := sdk.RandomString(10)
	lambdauser, _ := assets.InsertLambdaUser(t, db)
	p := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "gitlab", "gitlab")
	repo := assets.InsertTestProjectRepository(t, db, p.Key, vcs.ID, "ovh/cds")
	projectRunRetention := sdk.ProjectRunRetention{
		ProjectKey: p.Key,
		Retentions: sdk.Retentions{
			DefaultRetention: sdk.RetentionRule{
				DurationInDays: 10,
				Count:          2,
			},
			WorkflowRetentions: []sdk.WorkflowRetentions{
				{
					Workflow: "github/**/*",
					DefaultRetention: &sdk.RetentionRule{
						DurationInDays: 10,
						Count:          1,
					},
				},
			},
		},
	}
	require.NoError(t, project.InsertRunRetention(ctx, db, &projectRunRetention))

	runNumber := 1
	wr := sdk.V2WorkflowRun{
		ProjectKey:   p.Key,
		VCSServerID:  vcs.ID,
		VCSServer:    vcs.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: wkname,
		WorkflowSha:  "123456",
		Status:       sdk.V2WorkflowRunStatusFail,
		RunAttempt:   0,
		Started:      time.Now(),
		LastModified: time.Now(),
		Initiator: &sdk.V2Initiator{
			UserID: lambdauser.ID,
		},
		RunNumber:   0,
		WorkflowRef: "",
	}
	// Create run on master - We must keep only run_number 19 and 20
	for i := 0; i < 20; i++ {
		wr.RunNumber = int64(runNumber)
		runNumber++
		wr.WorkflowRef = "refs/heads/master"
		require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	}

	require.NoError(t, ApplyRunRetentionOnProject(ctx, db.DbMap, cache, p.Key))

	wrDB, err := workflow_v2.LoadRuns(ctx, db, p.Key, vcs.ID, repo.ID, wkname)
	require.NoError(t, err)

	require.Equal(t, 2, len(wrDB)) // 19 20
	for _, r := range wrDB {
		t.Logf("%s - %d", r.WorkflowRef, r.RunNumber)
	}
	require.Equal(t, int64(20), wrDB[0].RunNumber)
	require.Equal(t, int64(19), wrDB[1].RunNumber)
}
