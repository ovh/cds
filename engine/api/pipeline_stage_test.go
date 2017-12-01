package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestInsertAndLoadPipelineWith1StageAnd0ActionWithoutPrerequisite(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)
	deleteAll(t, api, "TESTPIPELINESTAGES")

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES", nil)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, nil))

	//Insert Stage
	stage := &sdk.Stage{
		Name:          "stage_Test_0",
		PipelineID:    pip.ID,
		BuildOrder:    1,
		Enabled:       true,
		Prerequisites: []sdk.Prerequisite{},
	}
	pip.Stages = append(pip.Stages, *stage)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(api.mustDB(), stage))

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, len(pip.Stages), len(loadedPip.Stages))
	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)
	assert.Equal(t, len(pip.Stages[0].Prerequisites), len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, len(pip.Stages[0].Jobs), len(loadedPip.Stages[0].Jobs))

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(api.mustDB(), pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = assets.DeleteTestProject(t, db, api.Cache, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestInsertAndLoadPipelineWith1StageAnd1ActionWithoutPrerequisite(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	deleteAll(t, api, "TESTPIPELINESTAGES")

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES", nil)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, nil))

	//Insert Stage
	stage := &sdk.Stage{
		Name:          "stage_Test_0",
		PipelineID:    pip.ID,
		BuildOrder:    1,
		Enabled:       true,
		Prerequisites: []sdk.Prerequisite{},
	}
	pip.Stages = append(pip.Stages, *stage)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(api.mustDB(), stage))

	//Insert Action
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)

	job := &sdk.Job{
		Action: sdk.Action{
			Name:    "NewAction",
			Enabled: true,
		},
		Enabled: true,
	}
	errJob := pipeline.InsertJob(api.mustDB(), job, stage.ID, pip)
	test.NoError(t, errJob)
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 1, len(loadedPip.Stages))
	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)
	assert.Equal(t, 0, len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, 1, len(loadedPip.Stages[0].Jobs))
	assert.Equal(t, job.Action.Name, loadedPip.Stages[0].Jobs[0].Action.Name)
	assert.Equal(t, true, loadedPip.Stages[0].Jobs[0].Enabled)

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(api.mustDB(), pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = assets.DeleteTestProject(t, db, api.Cache, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestInsertAndLoadPipelineWith2StagesWithAnEmptyStageAtFirstFollowedBy2ActionsStageWithoutPrerequisite(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	deleteAll(t, api, "TESTPIPELINESTAGES")

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES", nil)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, nil))

	//Insert Stage
	stage0 := &sdk.Stage{
		Name:          "stage_Test_0",
		PipelineID:    pip.ID,
		BuildOrder:    1,
		Enabled:       true,
		Prerequisites: []sdk.Prerequisite{},
	}
	pip.Stages = append(pip.Stages, *stage0)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage0.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(api.mustDB(), stage0))

	//Insert Stage
	stage1 := &sdk.Stage{
		Name:          "stage_Test_1",
		PipelineID:    pip.ID,
		BuildOrder:    2,
		Enabled:       true,
		Prerequisites: []sdk.Prerequisite{},
	}
	pip.Stages = append(pip.Stages, *stage1)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage1.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(api.mustDB(), stage1))

	//Insert Action
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage1.Name, pip.Name, proj.Name)
	job := &sdk.Job{
		Action: sdk.Action{
			Name:    "NewAction1",
			Enabled: true,
		},
		Enabled: true,
	}
	errJob := pipeline.InsertJob(api.mustDB(), job, stage1.ID, pip)
	test.NoError(t, errJob)
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	job2 := &sdk.Job{
		Action: sdk.Action{
			Name:    "NewAction2",
			Enabled: true,
		},
		Enabled: true,
	}
	errJob2 := pipeline.InsertJob(api.mustDB(), job2, stage1.ID, pip)
	test.NoError(t, errJob2)
	assert.NotZero(t, job2.PipelineActionID)
	assert.NotZero(t, job2.Action.ID)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)

	assert.Equal(t, 2, len(loadedPip.Stages))

	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)
	assert.Equal(t, 0, len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, 0, len(loadedPip.Stages[0].Jobs))

	assert.Equal(t, pip.Stages[1].Name, loadedPip.Stages[1].Name)
	assert.Equal(t, pip.Stages[1].Enabled, loadedPip.Stages[1].Enabled)
	assert.Equal(t, 0, len(loadedPip.Stages[1].Prerequisites))
	assert.Equal(t, 2, len(loadedPip.Stages[1].Jobs))

	assert.Equal(t, job.Action.Name, loadedPip.Stages[1].Jobs[0].Action.Name)
	assert.Equal(t, true, loadedPip.Stages[1].Jobs[0].Enabled)

	assert.Equal(t, job2.Action.Name, loadedPip.Stages[1].Jobs[1].Action.Name)
	assert.Equal(t, true, loadedPip.Stages[1].Jobs[1].Enabled)

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(api.mustDB(), pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = assets.DeleteTestProject(t, db, api.Cache, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestInsertAndLoadPipelineWith1StageWithoutPrerequisiteAnd1StageWith2Prerequisites(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	deleteAll(t, api, "TESTPIPELINESTAGES")

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES", nil)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, nil))

	//Insert Stage
	stage := &sdk.Stage{
		Name:          "stage_Test_0",
		PipelineID:    pip.ID,
		BuildOrder:    1,
		Enabled:       true,
		Prerequisites: []sdk.Prerequisite{},
	}
	pip.Stages = append(pip.Stages, *stage)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(api.mustDB(), stage))

	//Insert Action
	script, err := action.LoadPublicAction(api.mustDB(), "Script")
	test.NoError(t, err)
	t.Logf("Insert Action %s(%d) on Stage %s(%d) for Pipeline %s(%d) of Project %s", script.Name, script.ID, stage.Name, stage.ID, pip.Name, pip.ID, proj.Name)
	job := &sdk.Job{
		Action: sdk.Action{
			Name:    "NewAction1",
			Enabled: true,
		},
		Enabled: true,
	}
	errJob := pipeline.InsertJob(api.mustDB(), job, stage.ID, pip)
	test.NoError(t, errJob)
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	//Insert Stage
	stage1 := &sdk.Stage{
		Name:       "stage_Test_1",
		PipelineID: pip.ID,
		BuildOrder: 2,
		Enabled:    true,
		Prerequisites: []sdk.Prerequisite{
			sdk.Prerequisite{
				Parameter:     ".git.branch",
				ExpectedValue: "master",
			},
			sdk.Prerequisite{
				Parameter:     ".git.author",
				ExpectedValue: "someone@somewhere.com",
			},
		},
	}
	pip.Stages = append(pip.Stages, *stage1)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage1.Name, pip.Name, proj.Name)
	err = pipeline.InsertStage(api.mustDB(), stage1)
	test.NoError(t, err)
	assert.NotZero(t, stage1.ID)

	//Insert Action
	job1 := &sdk.Job{
		Action: sdk.Action{
			Name:    "NewAction2",
			Enabled: true,
		},
		Enabled: true,
	}
	errJob2 := pipeline.InsertJob(api.mustDB(), job1, stage1.ID, pip)
	test.NoError(t, errJob2)
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 2, len(loadedPip.Stages))
	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)

	assert.Equal(t, 0, len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, 1, len(loadedPip.Stages[0].Jobs))

	assert.Equal(t, pip.Stages[1].Name, loadedPip.Stages[1].Name)
	assert.Equal(t, pip.Stages[1].Enabled, loadedPip.Stages[1].Enabled)

	assert.Equal(t, 2, len(loadedPip.Stages[1].Prerequisites))
	assert.Equal(t, 1, len(loadedPip.Stages[1].Jobs))

	assert.Equal(t, job.Action.Name, loadedPip.Stages[0].Jobs[0].Action.Name)
	assert.Equal(t, true, loadedPip.Stages[0].Jobs[0].Enabled)

	assert.Equal(t, job1.Action.Name, loadedPip.Stages[1].Jobs[0].Action.Name)
	assert.Equal(t, true, loadedPip.Stages[1].Jobs[0].Enabled)

	assert.Equal(t, 2, len(loadedPip.Stages[1].Prerequisites))

	var foundGitBranch, foundGitAuthor bool
	for _, p := range loadedPip.Stages[1].Prerequisites {
		if p.Parameter == ".git.branch" {
			assert.Equal(t, "master", p.ExpectedValue)
			foundGitBranch = true
		}
		if p.Parameter == ".git.author" {
			assert.Equal(t, "someone@somewhere.com", p.ExpectedValue)
			foundGitAuthor = true
		}
	}

	assert.True(t, foundGitBranch)
	assert.True(t, foundGitAuthor)

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(api.mustDB(), pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = assets.DeleteTestProject(t, db, api.Cache, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestDeleteStageByIDShouldDeleteStagePrerequisites(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	deleteAll(t, api, "TESTPIPELINESTAGES")

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES", nil)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, nil))

	//Insert Stage
	stage := &sdk.Stage{
		Name:       "stage_Test_0",
		PipelineID: pip.ID,
		BuildOrder: 1,
		Enabled:    true,
		Prerequisites: []sdk.Prerequisite{
			sdk.Prerequisite{
				Parameter:     ".git.branch",
				ExpectedValue: "master",
			},
		},
	}
	pip.Stages = append(pip.Stages, *stage)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(api.mustDB(), stage))

	t.Logf("Delete Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.DeleteStageByID(api.mustDB(), stage, 1))

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 0, len(loadedPip.Stages))

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(api.mustDB(), pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = assets.DeleteTestProject(t, db, api.Cache, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestUpdateSTageShouldUpdateStagePrerequisites(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	deleteAll(t, api, "TESTPIPELINESTAGES")

	//Insert Project
	proj := assets.InsertTestProject(t, db, api.Cache, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES", nil)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip, nil))

	//Insert Stage
	stage := &sdk.Stage{
		Name:       "stage_Test_0",
		PipelineID: pip.ID,
		BuildOrder: 1,
		Enabled:    true,
		Prerequisites: []sdk.Prerequisite{
			sdk.Prerequisite{
				Parameter:     ".git.branch",
				ExpectedValue: "master",
			},
		},
	}
	pip.Stages = append(pip.Stages, *stage)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(api.mustDB(), stage))

	stage.Prerequisites = []sdk.Prerequisite{
		sdk.Prerequisite{
			Parameter:     "param1",
			ExpectedValue: "value1",
		},
		sdk.Prerequisite{
			Parameter:     "param2",
			ExpectedValue: "value2",
		},
	}

	t.Logf("Update Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.UpdateStage(api.mustDB(), stage))

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 1, len(loadedPip.Stages))
	assert.Equal(t, 2, len(loadedPip.Stages[0].Prerequisites))

	var foundParam1, foundParam2 bool
	for _, p := range loadedPip.Stages[0].Prerequisites {
		if p.Parameter == "param1" {
			assert.Equal(t, "value1", p.ExpectedValue)
			foundParam1 = true
		}
		if p.Parameter == "param2" {
			assert.Equal(t, "value2", p.ExpectedValue)
			foundParam2 = true
		}
	}

	assert.True(t, foundParam1)
	assert.True(t, foundParam2)

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(api.mustDB(), pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = assets.DeleteTestProject(t, db, api.Cache, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}
