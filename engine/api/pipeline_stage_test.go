package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"
)

func TestInsertStageWithoutPrerequisite(t *testing.T) {
	dba := test.Setup("TestInsertStageWithoutPrerequisite", t)
	db, err := dba.Begin()
	assert.NoError(t, err)

	stage := &sdk.Stage{
		Name:          "stage_0",
		PipelineID:    1,
		BuildOrder:    1,
		Enabled:       true,
		Prerequisites: []sdk.Prerequisite{},
	}

	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), stage.ID)

}

func TestInsertStageWithOnePrerequisite(t *testing.T) {
	dba := test.Setup("TestInsertStageWithOnePrerequisite", t)
	db, err := dba.Begin()
	assert.NoError(t, err)

	stage := &sdk.Stage{
		Name:       "stage_0",
		PipelineID: 1,
		BuildOrder: 1,
		Enabled:    true,
		Prerequisites: []sdk.Prerequisite{
			sdk.Prerequisite{
				Parameter:     "param0",
				ExpectedValue: "expected_value0",
			},
		},
	}

	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), stage.ID)
}

func TestInsertStageWithSeveralPrerequisites(t *testing.T) {
	//Skip until https://github.com/proullon/ramsql/issues/16 is resolved
	t.SkipNow()
	dba := test.Setup("TestInsertStageWithSeveralPrerequisites", t)
	db, err := dba.Begin()
	assert.NoError(t, err)

	stage := &sdk.Stage{
		Name:       "stage_0",
		PipelineID: 1,
		BuildOrder: 1,
		Enabled:    true,
		Prerequisites: []sdk.Prerequisite{
			sdk.Prerequisite{
				Parameter:     "param0",
				ExpectedValue: "expected_value0",
			},
			sdk.Prerequisite{
				Parameter:     "param1",
				ExpectedValue: "expected_value1",
			},
		},
	}

	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), stage.ID)
}

func TestInsertAndLoadPipelineWith1StageAnd0ActionWithoutPrerequisite(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj, err := testwithdb.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

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
	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	//Delete Project
	err = testwithdb.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	assert.NoError(t, err)
}

func TestInsertAndLoadPipelineWith1StageAnd1ActionWithoutPrerequisite(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj, err := testwithdb.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

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
	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

	//Insert Action
	script, err := action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	actionID, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage.ID)
	assert.NoError(t, err)
	assert.NotZero(t, actionID)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	//Delete Project
	err = testwithdb.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	assert.NoError(t, err)
}

func TestInsertAndLoadPipelineWith2StagesWithAnEmptyStageAtFirstFollowedBy2ActionsStageWithoutPrerequisite(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj, err := testwithdb.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

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
	err = pipeline.InsertStage(db, stage0)
	assert.NoError(t, err)

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
	err = pipeline.InsertStage(db, stage1)
	assert.NoError(t, err)

	//Insert Action
	script, err := action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage1.Name, pip.Name, proj.Name)
	actionID, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage1.ID)
	assert.NoError(t, err)
	assert.NotZero(t, actionID)

	//Insert Action
	script, err = action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage1.Name, pip.Name, proj.Name)
	actionID, err = pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage1.ID)
	assert.NoError(t, err)
	assert.NotZero(t, actionID)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	//Delete Project
	err = testwithdb.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	assert.NoError(t, err)
}

func TestInsertAndLoadPipelineWith1StageWithoutPrerequisiteAnd1StageWith2Prerequisites(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj, err := testwithdb.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

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
	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

	//Insert Action
	script, err := action.LoadPublicAction(db, "Script")
	t.Logf("Insert Action %s(%d) on Stage %s(%d) for Pipeline %s(%d) of Project %s", script.Name, script.ID, stage.Name, stage.ID, pip.Name, pip.ID, proj.Name)
	actionID, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script.ID, "[]", stage.ID)
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotZero(t, stage1.ID)

	//Insert Action
	script1, err := action.LoadPublicAction(db, "Artifact Upload")
	t.Logf("Insert Action %s(%d) on Stage %s(%d) for Pipeline %s(%d) of Project %s", script1.Name, script1.ID, stage1.Name, stage1.ID, pip.Name, pip.ID, proj.Name)
	actionID1, err := pipeline.InsertPipelineAction(db, proj.Key, pip.Name, script1.ID, "[]", stage1.ID)
	assert.NoError(t, err)
	assert.NotZero(t, actionID1)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	//Delete Project
	err = testwithdb.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	assert.NoError(t, err)
}

func TestDeleteStageByIDShouldDeleteStagePrerequisites(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj, err := testwithdb.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

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
	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

	t.Logf("Delete Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	err = pipeline.DeleteStageByID(db, stage, 1)
	assert.NoError(t, err)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, err)

	//Check all the things
	assert.NotNil(t, loadedPip)
	assert.Equal(t, 0, len(loadedPip.Stages))

	//Delete pipeline
	t.Logf("Delete Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.DeletePipeline(db, pip.ID, 1)
	assert.NoError(t, err)

	//Delete Project
	err = testwithdb.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	assert.NoError(t, err)
}

func TestUpdateSTageShouldUpdateStagePrerequisites(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	deleteAll(t, db, "TESTPIPELINESTAGES")

	//Insert Project
	proj, err := testwithdb.InsertTestProject(t, db, "TESTPIPELINESTAGES", "TESTPIPELINESTAGES")
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       "TEST_PIPELINE",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	err = pipeline.InsertPipeline(db, pip)
	assert.NoError(t, err)

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
	err = pipeline.InsertStage(db, stage)
	assert.NoError(t, err)

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
	err = pipeline.UpdateStage(db, stage)
	assert.NoError(t, err)

	//Loading Pipeline
	t.Logf("Reload Pipeline %s for Project %s", pip.Name, proj.Name)
	loadedPip, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, true)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	//Delete Project
	err = testwithdb.DeleteTestProject(t, db, "TESTPIPELINESTAGES")
	assert.NoError(t, err)
}
