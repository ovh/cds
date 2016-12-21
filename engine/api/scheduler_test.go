package main

import (
	_ "testing"

	_ "github.com/proullon/ramsql/engine/log"

	_ "github.com/ovh/cds/engine/api/action"
	_ "github.com/ovh/cds/engine/api/pipeline"
	_ "github.com/ovh/cds/engine/api/project"
	_ "github.com/ovh/cds/engine/api/scheduler"
	_ "github.com/ovh/cds/engine/api/test"
	_ "github.com/ovh/cds/sdk"
)

// FIXME ramsql subqueries
/*
func TestPipelineScheduler(t *testing.T) {

	log.SetLevel(log.InfoLevel)
	db := test.Setup("TestPipelineScheduler", t)

	// 1. Insert project
	myProject := &sdk.Project{
		Name: "Foo",
		Key:  "FOO",
	}
	err := project.InsertProject(db, myProject)
	if err != nil {
		t.Fatalf("Cannot insert project: %s\n", err)
	}

	// 2. Insert pipeline
	myPipeline := &sdk.Pipeline{
		Name:      "BAR",
		ProjectID: myProject.ID,
	}
	err = pipeline.InsertPipeline(db, myPipeline)
	if err != nil {
		t.Fatalf("Cannot insert pipeline: %s\n", err)
	}

	// 3. Create 3 actions
	actionData := &sdk.Action{
		Name: "Action1",
	}
	err = action.InsertAction(db, actionData)
	if err != nil {
		t.Fatalf("Cannot insert action1: %s", err)
	}

	actionData = &sdk.Action{
		Name: "Action2",
	}
	err = action.InsertAction(db, actionData)
	if err != nil {
		t.Fatalf("Cannot insert action2: %s", err)
	}

	actionData = &sdk.Action{
		Name: "Action3",
	}
	err = action.InsertAction(db, actionData)
	if err != nil {
		t.Fatalf("Cannot insert action3: %s", err)
	}

	// 4. Add 2 stages
	stage1 := &sdk.Stage{
		ID:         0,
		PipelineID: myPipeline.ID,
		Name:       "Stage 1",
		BuildOrder: 1,
		Enabled:    true,
	}
	err = pipeline.InsertStage(db, stage1)
	if err != nil {
		t.Fatalf("Cannot insert stage 1: %s", err)
	}
	if stage1.ID == 0 {
		t.Fatalf("Stage.ID cannot be 0")
	}

	stage2 := &sdk.Stage{
		ID:         0,
		PipelineID: myPipeline.ID,
		Name:       "Stage 2",
		BuildOrder: 2,
		Enabled:    true,
	}
	err = pipeline.InsertStage(db, stage2)
	if err != nil {
		t.Fatalf("Cannot insert stage 2: %s", err)
	}
	if stage2.ID == 0 {
		t.Fatalf("Stage.ID cannot be 0")
	}

	// 5. Add Action to pipeline

	// add action 1 in stage 1
	err = pipeline.InsertPipelineAction(db, myProject.Key, myPipeline.Name, "Action1", "[{\"name\":\"yo\",\"value\":\"lol\"}]", stage1.ID)
	if err != nil {
		t.Fatalf("cannot insert pipeline action 1: %s", err)
	}

	err = pipeline.InsertPipelineAction(db, myProject.Key, myPipeline.Name, "Action2", "[{\"name\":\"yo\",\"value\":\"lol\"}]", stage1.ID)
	if err != nil {
		t.Fatalf("cannot insert pipeline action 2: %s", err)
	}

	err = pipeline.InsertPipelineAction(db, myProject.Key, myPipeline.Name, "Action3", "[{\"name\":\"yo\",\"value\":\"lol\"}]", stage2.ID)
	if err != nil {
		t.Fatalf("cannot insert pipeline action 3: %s", err)
	}

	// 6. Create Pipeline Build

	if err != nil {
		t.Fatalf("Cannot load pipeline: %s", err)
	}
	pb, err := pipeline.InsertPipelineBuild(db, myPipeline, []string{})
	if err != nil {
		t.Fatalf("cannot insert pipeline build: %s\n", err)
	}

	// 7. Run scheduler => should schedule the 2 actions of stage 1
	log.Critical("1st wave")
	pipelineBuilds, err := pipeline.LoadBuildingPipelines(db)
	if err != nil {
		t.Fatalf("cannot load pipelines builds: %s\n", err)

	}
	for i := range pipelineBuilds {
		scheduler.PipelineScheduler(db, pipelineBuilds[i])
	}

	builds, err := build.LoadBuildByPipelineBuildID(db, pb.ID)
	if err != nil {
		t.Fatalf("cannot load builds: %s\n", err)
	}
	if len(builds) != 2 {
		t.Fatalf("Should have 2 actions, got %d\n", len(builds))
	}
	if builds[0].PipelineActionID == builds[1].PipelineActionID {
		t.Fatalf("Should not schedule the same action twice")
	}
	if !(builds[0].Status == builds[1].Status && builds[0].Status == sdk.StatusWaiting) {
		t.Fatalf("Action should be StatusWaiting")
	}

	// 8. Update action_build status
	err = build.UpdateActionBuildStatus(db, builds[0], sdk.StatusBuilding)
	if err != nil {
		t.Fatalf("cannot update build[0] status : %s\n", err)
	}
	err = build.UpdateActionBuildStatus(db, builds[0], sdk.StatusSuccess)
	if err != nil {
		t.Fatalf("cannot update build[0] status : %s\n", err)
	}
	err = build.UpdateActionBuildStatus(db, builds[1], sdk.StatusBuilding)
	if err != nil {
		t.Fatalf("cannot update build[0] status : %s\n", err)
	}
	err = build.UpdateActionBuildStatus(db, builds[1], sdk.StatusSuccess)
	if err != nil {
		t.Fatalf("cannot update build[1] status : %s\n", err)
	}

	// FIXME RAMSQL PB ?
	/*
		// 9. Run scheduler => should schedule the action of stage 2
		pipelineBuilds, err = pipeline.LoadBuildingPipelines(db)
		if err != nil {
			t.Fatalf("cannot load pipelines builds: %s\n", err)

		}

		for i := range pipelineBuilds {
			scheduler.PipelineScheduler(db, pipelineBuilds[i])
		}

		log.Warning("2nd wave")

		// 10.

		builds, err = build.LoadBuildByPipelineBuildID(db, pb.ID)
		if err != nil {
			t.Fatalf("cannot load builds: %s\n", err)
		}
		if len(builds) != 3 {
			t.Fatalf("Should have 3 actions, got %d\n", len(builds))
		}
}
*/
