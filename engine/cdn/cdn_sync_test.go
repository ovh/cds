package cdn

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/cds"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	commontest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestSyncLog(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, factory, cache, end := commontest.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(end)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.Background(), m, db)
	cdntest.ClearUnits(t, context.Background(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
		Cfg: Configuration{
			EnableLogProcessing: true,
		},
		Common: service.Common{
			GoRoutines: sdk.NewGoRoutines(),
		},
	}

	cdsConfig := &storage.CDSStorageConfiguration{
		Host:  "http://lolcat.host:8081",
		Token: "mytoken",
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name: "test-cds-backend",
				CDS:  cdsConfig,
			},
		},
	}, storage.LogConfig{NbJobLogsGoroutines: 0, NbServiceLogsGoroutines: 0, StepMaxSize: 100000, ServiceMaxSize: 10000, StepLinesRateLimit: 100})
	require.NoError(t, err)
	cdnUnits.Start(context.TODO(), sdk.NewGoRoutines())
	s.Units = cdnUnits

	cdsStorage, ok := s.Units.Storages[0].(*cds.CDS)
	require.True(t, ok)

	// Mock Http route
	gock.InterceptClient(cdsStorage.GetClient().HTTPClient())
	t.Cleanup(gock.Off)

	// 1 List project
	// 3 features enabled
	// 3 /nodes/ids

	gock.New("http://lolcat.host:8081").Post("/auth/consumer/builtin/signin").Times(-1).Reply(200).JSON(sdk.AuthConsumerSigninResponse{
		Token:  "",
		User:   &sdk.AuthentifiedUser{},
		APIURL: "http://lolcat.host:8081",
	})

	// Mock list project
	gock.New("http://lolcat.host:8081").Get("/project").Reply(200).JSON([]sdk.Project{{Key: "key1"}, {Key: "key2"}, {Key: "key3"}})

	// Mock feature enable
	gock.New("http://lolcat.host:8081").Post("/feature/enabled/cdn-job-logs").BodyString(`{"project_key": "key1"}`).Reply(200).JSON(sdk.FeatureEnabledResponse{
		Enabled: false,
	})
	gock.New("http://lolcat.host:8081").Post("/feature/enabled/cdn-job-logs").BodyString(`{"project_key": "key2"}`).Reply(200).JSON(sdk.FeatureEnabledResponse{
		Enabled: true,
	})
	gock.New("http://lolcat.host:8081").Post("/feature/enabled/cdn-job-logs").BodyString(`{"project_key": "key3"}`).Reply(200).JSON(sdk.FeatureEnabledResponse{
		Enabled: true,
	})

	// List node run identifiers for project 2
	gock.New("http://lolcat.host:8081").Get("/project/key2/workflows/runs/nodes/ids").Reply(200).JSON([]sdk.WorkflowNodeRunIdentifiers{
		{
			WorkflowID:    1000,
			WorkflowName:  "wkf1",
			RunNumber:     1000,
			NodeRunID:     1000,
			WorkflowRunID: 1000,
		},
		{
			WorkflowID:    1000,
			WorkflowName:  "wkf1",
			RunNumber:     1000,
			NodeRunID:     1001,
			WorkflowRunID: 1000,
		},
	})

	// List node run identifiers for project 3
	gock.New("http://lolcat.host:8081").Get("/project/key3/workflows/runs/nodes/ids").Reply(200).JSON([]sdk.WorkflowNodeRunIdentifiers{
		{
			WorkflowID:    2000,
			WorkflowName:  "wkf2",
			RunNumber:     2000,
			NodeRunID:     2000,
			WorkflowRunID: 2000,
		},
	})

	// List node run
	gock.New("http://lolcat.host:8081").Get("/project/key2/workflows/wkf1/runs/1000/nodes/1000").Reply(200).JSON(sdk.WorkflowNodeRun{
		WorkflowRunID:    1000,
		ID:               1000,
		WorkflowNodeName: "Node1000",
		Status:           sdk.StatusSuccess,
		Stages: []sdk.Stage{
			{
				RunJobs: []sdk.WorkflowNodeJobRun{
					{
						ID: 1000,
						Job: sdk.ExecutedJob{
							Job: sdk.Job{
								Action: sdk.Action{
									Name: "Job1000",
									Actions: []sdk.Action{
										{
											StepName: "stepAlreadyInCDN",
										},
									},
								},
							},
							StepStatus: []sdk.StepStatus{
								{
									StepOrder: 0,
								},
							},
						},
					},
				},
			},
		},
	})
	gock.New("http://lolcat.host:8081").Get("/project/key2/workflows/wkf1/runs/1000/nodes/1001").Reply(200).JSON(sdk.WorkflowNodeRun{
		WorkflowRunID:    1000,
		ID:               1001,
		WorkflowNodeName: "Node1000",
		Status:           sdk.StatusSuccess,
		Stages: []sdk.Stage{
			{
				RunJobs: []sdk.WorkflowNodeJobRun{
					{
						ID: 1001,
						Job: sdk.ExecutedJob{
							Job: sdk.Job{
								Action: sdk.Action{
									Name: "Job1001",
									Actions: []sdk.Action{
										{
											StepName: "step10",
										},
										{
											StepName: "step11",
										},
									},
									Requirements: sdk.RequirementList{
										{
											ID:    666,
											Name:  "pg",
											Type:  sdk.ServiceRequirement,
											Value: "postgres:5.1.12",
										},
									},
								},
							},
							StepStatus: []sdk.StepStatus{
								{
									StepOrder: 0,
								},
								{
									StepOrder: 1,
								},
							},
						},
					},
				},
			},
		},
	})
	gock.New("http://lolcat.host:8081").Get("/project/key3/workflows/wkf2/runs/2000/nodes/2000").Reply(200).JSON(sdk.WorkflowNodeRun{
		WorkflowRunID:    2000,
		ID:               2000,
		WorkflowNodeName: "Node2000",
		Status:           sdk.StatusBuilding,
		Stages: []sdk.Stage{
			{
				RunJobs: []sdk.WorkflowNodeJobRun{
					{
						ID: 2000,
						Job: sdk.ExecutedJob{
							Job: sdk.Job{
								Action: sdk.Action{
									Name: "Job2000",
									Actions: []sdk.Action{
										{
											StepName: "stepEncours",
										},
									},
								},
							},
							StepStatus: []sdk.StepStatus{
								{
									StepOrder: 0,
								},
							},
						},
					},
				},
			},
		},
	})

	// Get log
	gock.New("http://lolcat.host:8081").Get("/project/key2/workflows/wkf1/nodes/1001/job/1001/step/0/log").Reply(200).JSON(sdk.BuildState{
		StepLogs: sdk.Log{Val: "Je suis ton log step 1"},
	})
	gock.New("http://lolcat.host:8081").Get("/project/key2/workflows/wkf1/nodes/1001/job/1001/step/1/log").Reply(200).JSON(sdk.BuildState{
		StepLogs: sdk.Log{Val: "Je suis ton log step 2 et je suis plus long"},
	})

	// Get Service log ( call twice )
	gock.New("http://lolcat.host:8081").Times(2).Get("/project/key2/workflows/wkf1/nodes/1001/job/1001/service/pg/log").Reply(200).JSON(sdk.ServiceLog{
		ServiceRequirementName: "pg",
		Val:                    "Je suis un log de service",
	})

	// Insert item for wkf1, 1000
	apiRef1000 := sdk.CDNLogAPIRef{
		ProjectKey:     "key2",
		WorkflowName:   "wkf1",
		WorkflowID:     1000,
		RunID:          1000,
		NodeRunID:      1000,
		NodeRunName:    "Node1000",
		NodeRunJobID:   1000,
		NodeRunJobName: "Job1000",
		StepName:       "stepAlreadyInCDN",
		StepOrder:      0,
	}
	hash, err := apiRef1000.ToHash()
	require.NoError(t, err)
	itm := sdk.CDNItem{
		APIRef:     apiRef1000,
		APIRefHash: hash,
		Type:       sdk.CDNTypeItemStepLog,
	}
	err = item.Insert(context.TODO(), s.Mapper, db, &itm)
	if !sdk.ErrorIs(err, sdk.ErrConflictData) {
		require.NoError(t, err)
	}
	defer func() {
		_ = item.DeleteByID(db, itm.ID)
	}()

	unit, err := storage.LoadUnitByName(context.TODO(), s.Mapper, db, "test-cds-backend")
	require.NoError(t, err)

	// Clean before testing
	oneHundred := 100

	ius, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, unit.ID, &oneHundred)
	require.NoError(t, err)
	for _, iu := range ius {
		require.NoError(t, item.DeleteByID(db, iu.ItemID))
	}

	// Run Test
	require.NoError(t, s.SyncLogs(context.TODO(), cdsStorage))

	itemUnits, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, unit.ID, &oneHundred)
	require.NoError(t, err)
	require.Len(t, itemUnits, 3)

	for _, i := range itemUnits {
		t.Cleanup(func() {
			_ = item.DeleteByID(db, i.ItemID)
		})
	}

	item1, err := item.LoadByID(context.TODO(), s.Mapper, db, itemUnits[0].ItemID)
	require.NoError(t, err)
	require.Equal(t, int64(22), item1.Size)
	require.Equal(t, sdk.CDNTypeItemStepLog, item1.Type)

	item2, err := item.LoadByID(context.TODO(), s.Mapper, db, itemUnits[1].ItemID)
	require.NoError(t, err)
	require.Equal(t, int64(43), item2.Size)
	require.Equal(t, sdk.CDNTypeItemStepLog, item2.Type)

	item3, err := item.LoadByID(context.TODO(), s.Mapper, db, itemUnits[2].ItemID)
	require.NoError(t, err)
	require.Equal(t, int64(25), item3.Size)
	require.Equal(t, sdk.CDNTypeItemServiceLog, item3.Type)
}
