package pipeline_test

import (
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type args struct {
	pkey string
	pip  *sdk.Pipeline
	u    *sdk.User
}

type testcase struct {
	name     string
	args     args
	wantErr  bool
	setup    func(t *testing.T, args args)
	asserts  func(t *testing.T, pip sdk.Pipeline)
	tearDown func(t *testing.T, args args)
}

func testImportUpdate(t *testing.T, db gorp.SqlExecutor, store cache.Store, tt testcase) {
	msgChan := make(chan sdk.Message, 1)
	done := make(chan bool)

	go func() {
		for {
			msg, ok := <-msgChan
			if !ok {
				done <- true
				return
			}
			log.Debug("[TEST] %s >>> %s", tt.name, msg.String("en"))
		}
	}()

	if tt.setup != nil {
		tt.setup(t, tt.args)
	}

	proj, err := project.Load(db, store, tt.args.pip.ProjectKey, nil)
	test.NoError(t, err)

	if err := pipeline.ImportUpdate(db, proj, tt.args.pip, msgChan, tt.args.u); (err != nil) != tt.wantErr {
		t.Errorf("%q. ImportUpdate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
	}

	close(msgChan)
	<-done

	pip, err := pipeline.LoadPipeline(db, tt.args.pip.ProjectKey, tt.args.pip.Name, true)
	test.NoError(t, err)

	if tt.asserts != nil {
		tt.asserts(t, *pip)
	}

	if tt.tearDown != nil {
		tt.setup(t, tt.args)
	}
}

func TestImportUpdate(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}

	u, _ := assets.InsertAdminUser(db)

	//Define the testscases
	var test1 = testcase{
		name:    "import a new stage with one job on a empty pipeline",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))
			args.pip.Stages = []sdk.Stage{
				sdk.Stage{
					BuildOrder: 1,
					Enabled:    true,
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: false,
							Action: sdk.Action{
								Name:        "Job 1",
								Description: "This is the first job",
							},
						},
					},
					Name: "This is the first stage",
				},
			}
		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			assert.Equal(t, 1, len(pip.Stages))
			assert.Equal(t, 1, len(pip.Stages[0].Jobs))
		},
	}

	var test2 = testcase{
		name:    "import a new stage with one job on a pipeline with no job",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))

			args.pip.Stages = []sdk.Stage{
				{
					BuildOrder: 1,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the first stage. It has no jobs",
				},
			}

			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[0]))
			for _, j := range args.pip.Stages[0].Jobs {
				test.NoError(t, pipeline.InsertJob(db, &j, args.pip.Stages[0].ID, args.pip))
			}

			args.pip.Stages = append(args.pip.Stages,
				sdk.Stage{
					BuildOrder: 2,
					Enabled:    true,
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: false,
							Action: sdk.Action{
								Name:        "Job 1",
								Description: "This is the first job",
							},
						},
					},
					Name: "This is the second stage",
				},
			)
		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			t.Logf("Asserts on %+v", pip)
			assert.Equal(t, 2, len(pip.Stages))
			assert.Equal(t, 0, len(pip.Stages[0].Jobs))
			assert.Equal(t, 1, len(pip.Stages[1].Jobs))
		},
	}

	var test3 = testcase{
		name:    "remove stage on a pipeline with two stages",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))

			args.pip.Stages = []sdk.Stage{
				{
					BuildOrder: 1,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the first stage. It has no jobs",
				},
				{
					BuildOrder: 2,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the second stage. It has no jobs",
				},
			}

			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[0]))
			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[1]))

			args.pip.Stages = args.pip.Stages[:1]
		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			t.Logf("Asserts on %+v", pip)
			assert.Equal(t, 1, len(pip.Stages))
			assert.Equal(t, 0, len(pip.Stages[0].Jobs))
		},
	}

	var test4 = testcase{
		name:    "remove all the stages",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))

			args.pip.Stages = []sdk.Stage{
				{
					BuildOrder: 1,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the first stage. It has no jobs",
				},
				{
					BuildOrder: 2,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the second stage. It has no jobs",
				},
			}

			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[0]))
			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[1]))

			args.pip.Stages = nil
		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			t.Logf("Asserts on %+v", pip)
			assert.Equal(t, 0, len(pip.Stages))
		},
	}

	var test5 = testcase{
		name:    "Add a job on a stage",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))

			args.pip.Stages = []sdk.Stage{
				{
					BuildOrder: 1,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the first stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
				{
					BuildOrder: 2,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the second stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
			}

			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[0]))
			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[1]))
			for _, j := range args.pip.Stages[0].Jobs {
				test.NoError(t, pipeline.InsertJob(db, &j, args.pip.Stages[0].ID, args.pip))
			}
			for _, j := range args.pip.Stages[1].Jobs {
				test.NoError(t, pipeline.InsertJob(db, &j, args.pip.Stages[1].ID, args.pip))
			}

			args.pip.Stages[1].Jobs = append(args.pip.Stages[1].Jobs, sdk.Job{
				Enabled: true,
				Action: sdk.Action{
					Name: "Job n°3",
				},
			})

		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			t.Logf("Asserts on %+v", pip)
			assert.Equal(t, 2, len(pip.Stages))
			assert.Equal(t, 2, len(pip.Stages[0].Jobs))
			assert.Equal(t, 3, len(pip.Stages[1].Jobs))
		},
	}

	var test6 = testcase{
		name:    "Update a job on a stage",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			args.pip.Parameter = []sdk.Parameter{
				{Name: "test", Value: "test_value", Type: sdk.StringParameter, Description: "test_description"},
			}
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))

			args.pip.Parameter = []sdk.Parameter{
				{Name: "test", Value: "test_value_bis", Type: sdk.StringParameter, Description: "test_description_bis"},
			}
			args.pip.Stages = []sdk.Stage{
				{
					BuildOrder: 1,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the first stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
				{
					BuildOrder: 2,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the second stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
			}

			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[0]))
			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[1]))

			args.pip.Stages[1].Jobs[1] = sdk.Job{
				Enabled: true,
				Action: sdk.Action{
					Name: "Job n°2bis",
				},
			}

		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			t.Logf("Asserts on %+v", pip)
			assert.Equal(t, 2, len(pip.Stages))
			assert.Equal(t, 2, len(pip.Stages[0].Jobs))
			assert.Equal(t, 2, len(pip.Stages[1].Jobs))
			assert.Equal(t, 1, len(pip.Parameter))
			assert.Equal(t, "test_value_bis", pip.Parameter[0].Value)
			assert.Equal(t, "test_description_bis", pip.Parameter[0].Description)
			assert.Equal(t, "Job n°2bis", pip.Stages[1].Jobs[1].Action.Name)
		},
	}

	var test7 = testcase{
		name:    "Remove a job on a stage and add parameter",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))

			args.pip.Parameter = []sdk.Parameter{
				{Name: "test", Value: "test_value", Type: sdk.StringParameter, Description: "test_description"},
			}
			args.pip.Stages = []sdk.Stage{
				{
					BuildOrder: 1,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the first stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
				{
					BuildOrder: 2,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the second stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
			}

			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[0]))
			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[1]))

			args.pip.Stages[1].Jobs = args.pip.Stages[1].Jobs[1:]

		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			t.Logf("Asserts on %+v", pip)
			assert.Equal(t, 2, len(pip.Stages))
			assert.Equal(t, 2, len(pip.Stages[0].Jobs))
			assert.Equal(t, 1, len(pip.Stages[1].Jobs))
			assert.Equal(t, 1, len(pip.Parameter))
			assert.Equal(t, "Job n°2", pip.Stages[1].Jobs[0].Action.Name)
		},
	}

	var test8 = testcase{
		name:    "Change stage order",
		wantErr: false,
		args: args{
			u:    u,
			pkey: sdk.RandomString(7),
			pip: &sdk.Pipeline{
				Type: sdk.BuildPipeline,
			},
		},
		setup: func(t *testing.T, args args) {
			proj := assets.InsertTestProject(t, db, cache, args.pkey, args.pkey, nil)
			args.pip.Name = proj.Key + "_PIP"
			args.pip.ProjectID = proj.ID
			args.pip.ProjectKey = proj.Key
			test.NoError(t, pipeline.InsertPipeline(db, cache, proj, args.pip, nil))

			args.pip.Parameter = []sdk.Parameter{
				{Name: "test", Value: "test_value", Type: sdk.StringParameter, Description: "test_description"},
			}
			args.pip.Stages = []sdk.Stage{
				{
					BuildOrder: 1,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the first stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
				{
					BuildOrder: 2,
					Enabled:    true,
					PipelineID: args.pip.ID,
					Name:       "This is the second stage. It has 2 jobs",
					Jobs: []sdk.Job{
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°1",
							},
						},
						sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name: "Job n°2",
							},
						},
					},
				},
			}

			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[0]))
			test.NoError(t, pipeline.InsertStage(db, &args.pip.Stages[1]))

			args.pip.Stages[0].BuildOrder = 2
			args.pip.Stages[1].BuildOrder = 1

		},
		asserts: func(t *testing.T, pip sdk.Pipeline) {
			t.Logf("Asserts on %+v", pip)
			assert.Equal(t, 2, len(pip.Stages))
			assert.Equal(t, 1, pip.Stages[0].BuildOrder)
			assert.Equal(t, 2, pip.Stages[1].BuildOrder)
			assert.Equal(t, "This is the second stage. It has 2 jobs", pip.Stages[0].Name)
			assert.Equal(t, "This is the first stage. It has 2 jobs", pip.Stages[1].Name)
		},
	}

	//Run the tests
	var tests = []testcase{test1, test2, test3, test4, test5, test6, test7, test8}
	for _, tt := range tests {
		testImportUpdate(t, db, cache, tt)
	}
}
