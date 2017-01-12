package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestInsertAndLoadPipelineWith1StageAnd0ActionWithoutPrerequisite(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj := test.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(db, pip))

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
	test.NoError(t, pipeline.InsertStage(db, stage))

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, len(pip.Stages), len(loadedPip.Stages))
	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)
	assert.Equal(t, len(pip.Stages[0].Prerequisites), len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, len(pip.Stages[0].Actions), len(loadedPip.Stages[0].Actions))

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = test.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestInsertAndLoadPipelineWith1StageAnd1ActionWithoutPrerequisite(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj := test.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(db, pip))

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
	test.NoError(t, pipeline.InsertStage(db, stage))

	//Insert Action
	script, err := action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	actionID, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage.ID)
	test.NoError(t, err)
	assert.NotZero(t, actionID)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 1, len(loadedPip.Stages))
	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)
	assert.Equal(t, 0, len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, 1, len(loadedPip.Stages[0].Actions))
	assert.Equal(t, script.Name, loadedPip.Stages[0].Actions[0].Name)
	assert.Equal(t, true, loadedPip.Stages[0].Actions[0].Enabled)

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = test.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestInsertAndLoadPipelineWith2StagesWithAnEmptyStageAtFirstFollowedBy2ActionsStageWithoutPrerequisite(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj := test.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(db, pip))

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
	test.NoError(t, pipeline.InsertStage(db, stage0))

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
	test.NoError(t, pipeline.InsertStage(db, stage1))

	//Insert Action
	script, err := action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage1.Name, pip.Name, proj.Name)
	actionID, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage1.ID)
	test.NoError(t, err)
	assert.NotZero(t, actionID)

	//Insert Action
	script, err = action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage1.Name, pip.Name, proj.Name)
	actionID, err = pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage1.ID)
	test.NoError(t, err)
	assert.NotZero(t, actionID)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)

	assert.Equal(t, 2, len(loadedPip.Stages))

	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)
	assert.Equal(t, 0, len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, 0, len(loadedPip.Stages[0].Actions))

	assert.Equal(t, pip.Stages[1].Name, loadedPip.Stages[1].Name)
	assert.Equal(t, pip.Stages[1].Enabled, loadedPip.Stages[1].Enabled)
	assert.Equal(t, 0, len(loadedPip.Stages[1].Prerequisites))
	assert.Equal(t, 2, len(loadedPip.Stages[1].Actions))

	assert.Equal(t, script.Name, loadedPip.Stages[1].Actions[0].Name)
	assert.Equal(t, true, loadedPip.Stages[1].Actions[0].Enabled)

	assert.Equal(t, script.Name, loadedPip.Stages[1].Actions[1].Name)
	assert.Equal(t, true, loadedPip.Stages[1].Actions[1].Enabled)

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = test.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestInsertAndLoadPipelineWith1StageWithoutPrerequisiteAnd1StageWith2Prerequisites(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj := test.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(db, pip))

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
	test.NoError(t, pipeline.InsertStage(db, stage))

	//Insert Action
	script, err := action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action %s(%d) on Stage %s(%d) for Pipeline %s(%d) of Project %s", script.Name, script.ID, stage.Name, stage.ID, pip.Name, pip.ID, proj.Name)
	actionID, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage.ID)
	test.NoError(t, err)
	assert.NotZero(t, actionID)

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
	err = pipeline.InsertStage(db, stage1)
	test.NoError(t, err)
	assert.NotZero(t, stage1.ID)

	//Insert Action
	script1, err := action.LoadPublicAction(db, "Artifact Upload")
	t.Logf("Insert Action %s(%d) on Stage %s(%d) for Pipeline %s(%d) of Project %s", script1.Name, script1.ID, stage1.Name, stage1.ID, pip.Name, pip.ID, proj.Name)
	actionID1, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script1.ID, "[]", stage1.ID)
	test.NoError(t, err)
	assert.NotZero(t, actionID1)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 2, len(loadedPip.Stages))
	assert.Equal(t, pip.Stages[0].Name, loadedPip.Stages[0].Name)
	assert.Equal(t, pip.Stages[0].Enabled, loadedPip.Stages[0].Enabled)

	assert.Equal(t, 0, len(loadedPip.Stages[0].Prerequisites))
	assert.Equal(t, 1, len(loadedPip.Stages[0].Actions))

	assert.Equal(t, pip.Stages[1].Name, loadedPip.Stages[1].Name)
	assert.Equal(t, pip.Stages[1].Enabled, loadedPip.Stages[1].Enabled)

	assert.Equal(t, 2, len(loadedPip.Stages[1].Prerequisites))
	assert.Equal(t, 1, len(loadedPip.Stages[1].Actions))

	assert.Equal(t, script.Name, loadedPip.Stages[0].Actions[0].Name)
	assert.Equal(t, true, loadedPip.Stages[0].Actions[0].Enabled)

	assert.Equal(t, script1.Name, loadedPip.Stages[1].Actions[0].Name)
	assert.Equal(t, true, loadedPip.Stages[1].Actions[0].Enabled)

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
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = test.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestDeleteStageByIDShouldDeleteStagePrerequisites(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj := test.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(db, pip))

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
	test.NoError(t, pipeline.InsertStage(db, stage))

	t.Logf("Delete Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.DeleteStageByID(db, stage, 1))

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	test.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 0, len(loadedPip.Stages))

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = test.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}

func TestUpdateSTageShouldUpdateStagePrerequisites(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj := test.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertPipeline(db, pip))

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
	test.NoError(t, pipeline.InsertStage(db, stage))

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
	test.NoError(t, pipeline.UpdateStage(db, stage))

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
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
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	test.NoError(t, err)

	//Delete Project
	err = test.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	test.NoError(t, err)
}
